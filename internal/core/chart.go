package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
	"github.com/SimonSchneider/pefigo/internal/finance"
	"github.com/SimonSchneider/pefigo/internal/pdb"
	"github.com/SimonSchneider/pefigo/internal/ui"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

func ChartPage() http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		p := PredictionParams{}
		if err := srvu.Decode(r, &p, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		return NewView(ctx, w, r).Render(Page("Chart", PageChart(p)))
	})
}

type SSEPredictionEventHandler struct {
	w *srvu.SSESender
}

func (s *SSEPredictionEventHandler) Setup(e PredictionSetupEvent) error {
	return s.w.SendNamedJson("setup", e)
}
func (s *SSEPredictionEventHandler) Snapshot(e PredictionBalanceSnapshot) error {
	return s.w.SendNamedJson("balanceSnapshot", e)
}
func (s *SSEPredictionEventHandler) Close() error {
	return s.w.SendEventWithoutData("close")
}

func HandlerChartsDataStream(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var params PredictionParams
		if err := srvu.Decode(r, &params, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		if err := RunPrediction(ctx, db, &SSEPredictionEventHandler{w: srvu.SSEResponse(w)}, params); err != nil {
			return fmt.Errorf("running prediction: %w", err)
		}
		return nil
	})
}

type PredictionParams struct {
	Duration         date.Duration
	Samples          int64
	Quantile         float64
	SnapshotInterval date.Cron
	GroupBy          GroupBy
}

type GroupBy string

const (
	GroupByNone  GroupBy = "none"
	GroupByType  GroupBy = "type"
	GroupByTotal GroupBy = "total"
)

func ParseGroupBy(val string) (GroupBy, error) {
	switch val {
	case "none":
		return GroupByNone, nil
	case "type":
		return GroupByType, nil
	case "total":
		return GroupByTotal, nil
	default:
		return GroupByNone, fmt.Errorf("invalid group by: %s", val)
	}
}

func (p *PredictionParams) FromForm(r *http.Request) error {
	if err := shttp.Parse(&p.Duration, date.ParseDuration, r.FormValue("duration"), 365); err != nil {
		return fmt.Errorf("parsing duration: %w", err)
	}
	if err := shttp.Parse(&p.Samples, ui.ParseHumanNumber(ui.ParseInt64), r.FormValue("samples"), 2000); err != nil {
		return fmt.Errorf("parsing samples: %w", err)
	}
	if err := shttp.Parse(&p.Quantile, shttp.ParseFloat, r.FormValue("quantile"), 0.8); err != nil {
		return fmt.Errorf("parsing quantile: %w", err)
	}
	if err := shttp.Parse(&p.SnapshotInterval, ui.ParseDateCron, r.FormValue("snapshot_interval"), "*-*-28"); err != nil {
		return fmt.Errorf("parsing snapshot interval: %w", err)
	}
	if err := shttp.Parse(&p.GroupBy, ParseGroupBy, r.FormValue("group_by"), GroupByType); err != nil {
		return fmt.Errorf("parsing group by: %w", err)
	}
	return nil
}

type PredictionBalanceSnapshot struct {
	ID         string  `json:"id"`
	Day        int64   `json:"day"`
	Balance    float64 `json:"balance"`
	LowerBound float64 `json:"lowerBound"`
	UpperBound float64 `json:"upperBound"`
}
type PredictionFinancialEntity struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	Color     string                      `json:"color"`
	Snapshots []PredictionBalanceSnapshot `json:"snapshots"`
}
type Markline struct {
	Date int64  `json:"date"`
	Name string `json:"name"`
}
type PredictionSetupEvent struct {
	Max       int64                       `json:"max"`
	Entities  []PredictionFinancialEntity `json:"entities"`
	Marklines []Markline                  `json:"marklines"`
}

type PredictionEventHandler interface {
	Setup(PredictionSetupEvent) error
	Snapshot(PredictionBalanceSnapshot) error
	Close() error
}

func RunPrediction(ctx context.Context, db *sql.DB, eventHandler PredictionEventHandler, params PredictionParams) error {
	q := pdb.New(db)
	q1, q2 := (1-params.Quantile)/2, (1+params.Quantile)/2

	transfers := make([]finance.TransferTemplate, 0)
	entities := make([]finance.Entity, 0)
	accs, err := q.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("listing accounts for Prediction: %w", err)
	}
	trans, err := ListTransferTemplates(ctx, db)
	if err != nil {
		return fmt.Errorf("listing transfers for Prediction: %w", err)
	}
	accountTypes, err := q.ListAccountTypes(ctx)
	if err != nil {
		return fmt.Errorf("listing account types for Prediction: %w", err)
	}
	specialDates, err := q.GetSpecialDates(ctx)
	if err != nil {
		return fmt.Errorf("listing special dates for Prediction: %w", err)
	}
	accountTypesById := KeyBy(accountTypes, func(at pdb.AccountType) string { return at.ID })
	accsById := make(map[string]pdb.Account, len(accs))
	startDate := date.Today()
	for _, acc := range accs {
		accsById[acc.ID] = acc
		snaps, err := ListAccountSnapshots(ctx, db, acc.ID)
		if err != nil {
			return fmt.Errorf("getting snapshots for account %s: %w", acc.ID, err)
		}
		gms, err := ListAccountGrowthModels(ctx, db, acc.ID)
		if err != nil {
			return fmt.Errorf("getting growth models for account %s: %w", acc.ID, err)
		}
		var balanceLimit finance.BalanceLimit
		if acc.BalanceUpperLimit != nil {
			balanceLimit = finance.BalanceLimit{
				Upper: uncertain.NewFixed(*acc.BalanceUpperLimit),
			}
		}
		entity := finance.Entity{
			ID:           acc.ID,
			Name:         acc.Name,
			BalanceLimit: balanceLimit,
			Snapshots:    make([]finance.BalanceSnapshot, 0, len(snaps)),
		}
		if acc.CashFlowFrequency != nil || acc.CashFlowDestinationID != nil {
			entity.CashFlow = &finance.CashFlowModel{
				Frequency:     date.Cron(ui.OrDefault(acc.CashFlowFrequency)),
				DestinationID: ui.OrDefault(acc.CashFlowDestinationID),
			}
		}
		for _, snap := range snaps {
			entity.Snapshots = append(entity.Snapshots, finance.BalanceSnapshot{
				Date:    snap.Date,
				Balance: snap.Balance,
			})
			if snap.Date.After(startDate) {
				startDate = snap.Date
			}
		}
		fgms := make([]finance.GrowthModel, 0, len(gms))
		for _, gm := range gms {
			switch gm.Type {
			case "fixed":
				fgms = append(fgms, &finance.FixedGrowth{
					TimeFrameGrowth: finance.TimeFrameGrowth{
						StartDate: gm.StartDate,
						EndDate:   gm.EndDate,
					},
					AnnualRate: gm.AnnualRate,
				})
			case "lognormal":
				fgms = append(fgms, &finance.LogNormalGrowth{
					TimeFrameGrowth: finance.TimeFrameGrowth{
						StartDate: gm.StartDate,
						EndDate:   gm.EndDate,
					},
					AnnualRate:       gm.AnnualRate,
					AnnualVolatility: gm.AnnualVolatility,
				})
			}
		}
		if len(fgms) == 1 {
			entity.GrowthModel = finance.NewGrowthCombined(fgms...)
		}
		if len(entity.Snapshots) > 0 {
			entities = append(entities, entity)
		}
	}
	for _, t := range trans {
		transfers = append(transfers, finance.TransferTemplate{
			ID:            t.ID,
			Name:          t.Name,
			FromAccountID: t.FromAccountID,
			ToAccountID:   t.ToAccountID,
			AmountType:    finance.TransferAmountType(t.AmountType),
			AmountFixed: finance.TransferFixed{
				Amount: t.AmountFixed,
			},
			AmountPercent: finance.TransferPercent{
				Percent: t.AmountPercent,
			},
			Priority:      t.Priority,
			EffectiveFrom: t.StartDate,
			EffectiveTo:   t.EndDate,
			Recurrence:    t.Recurrence,
			Enabled:       t.Enabled,
		})
	}

	startDate += 1
	endDate := startDate.Add(params.Duration)
	ucfg := uncertain.NewConfig(time.Now().UnixMilli(), params.Samples)

	h := &GroupingEventHandler{eventHandler: eventHandler, ucfg: ucfg, accsById: accsById, accountTypesById: accountTypesById, groupBy: params.GroupBy, q1: q1, q2: q2}

	if err := h.Setup(entities, endDate, specialDates); err != nil {
		return fmt.Errorf("setting up grouping event handler: %w", err)
	}

	snapshotRecorder := finance.SnapshotRecorderFunc(func(accountID string, day date.Date, balance uncertain.Value) error {
		return h.Snapshot(accountID, day, balance)
	})
	if err := finance.RunPrediction(ctx, ucfg, startDate, endDate, params.SnapshotInterval, entities, transfers, finance.CompositeRecorder{SnapshotRecorder: snapshotRecorder}); err != nil {
		return fmt.Errorf("running prediction for SSE: %w", err)
	}
	return h.Close()
}

type GroupingEventHandler struct {
	eventHandler     PredictionEventHandler
	ucfg             *uncertain.Config
	accsById         map[string]pdb.Account
	accountTypesById map[string]pdb.AccountType
	groupBy          GroupBy
	q1               float64
	q2               float64

	currentDate date.Date
	currentAccs map[string]uncertain.Value
}

func (h *GroupingEventHandler) Setup(entities []finance.Entity, endDate date.Date, specialDates []pdb.SpecialDate) error {
	for _, e := range h.accsById {
		if e.TypeID == nil {
			h.accountTypesById[""] = pdb.AccountType{
				ID:   "",
				Name: "unknown",
			}
		}
	}

	type GroupedEntities struct {
		name  string
		color string
		dates map[date.Date]uncertain.Value
	}

	groupedEntities := make(map[string]GroupedEntities)
	switch h.groupBy {
	case GroupByTotal:
		groupedEntities["total"] = GroupedEntities{
			name:  "total",
			color: "",
			dates: make(map[date.Date]uncertain.Value),
		}
	case GroupByType:
		for _, accountType := range h.accountTypesById {
			groupedEntities[accountType.ID] = GroupedEntities{
				name:  accountType.Name,
				color: ui.OrDefault(accountType.Color),
				dates: make(map[date.Date]uncertain.Value),
			}
		}
	case GroupByNone:
		for _, account := range h.accsById {
			groupedEntities[account.ID] = GroupedEntities{
				name:  account.Name,
				color: "",
				dates: make(map[date.Date]uncertain.Value),
			}
		}
	}

	for _, e := range entities {
		key := h.getKey(e.ID)

		ent := groupedEntities[key]

		for _, s := range e.Snapshots {
			amount := ent.dates[s.Date]
			if amount.Zero() {
				amount = s.Balance
			} else {
				amount = amount.Add(h.ucfg, s.Balance)
			}
			ent.dates[s.Date] = amount
		}
	}

	sssEntities := make([]PredictionFinancialEntity, 0, len(groupedEntities))
	for id, e := range groupedEntities {
		ent := PredictionFinancialEntity{
			ID:        id,
			Name:      e.name,
			Color:     e.color,
			Snapshots: make([]PredictionBalanceSnapshot, 0, len(e.dates)),
		}
		for day, amount := range e.dates {
			q := amount.Quantiles()
			ent.Snapshots = append(ent.Snapshots, PredictionBalanceSnapshot{
				ID:         id,
				Day:        day.ToStdTime().UnixMilli(),
				Balance:    amount.Mean(),
				LowerBound: q(h.q1),
				UpperBound: q(h.q2),
			})
		}
		sort.Slice(ent.Snapshots, func(i, j int) bool {
			return ent.Snapshots[i].Day < ent.Snapshots[j].Day
		})
		sssEntities = append(sssEntities, ent)
	}
	marklines := make([]Markline, 0, len(specialDates))
	for _, sd := range specialDates {
		day, err := date.ParseDate(sd.Date)
		if err != nil {
			return fmt.Errorf("parsing special date: %w", err)
		}
		marklines = append(marklines, Markline{
			Date: day.ToStdTime().UnixMilli(),
			Name: sd.Name,
		})
	}
	h.currentDate = endDate
	h.currentAccs = make(map[string]uncertain.Value, len(h.accsById))
	return h.eventHandler.Setup(PredictionSetupEvent{
		Max:       endDate.ToStdTime().UnixMilli(),
		Entities:  sssEntities,
		Marklines: marklines,
	})
}

func (h *GroupingEventHandler) getKey(id string) string {
	switch h.groupBy {
	case GroupByType:
		return h.accountTypesById[ui.OrDefault(h.accsById[id].TypeID)].ID
	case GroupByNone:
		return id
	case GroupByTotal:
		return "total"
	default:
		panic("invalid group by")
	}
}

func (h *GroupingEventHandler) Snapshot(id string, day date.Date, balance uncertain.Value) error {
	if h.currentDate != day {
		if err := h.Flush(); err != nil {
			return err
		}
		h.currentDate = day
	}

	key := h.getKey(id)
	acc, ok := h.currentAccs[key]
	if !ok {
		h.currentAccs[key] = balance
	} else {
		h.currentAccs[key] = acc.Add(h.ucfg, balance)
	}
	return nil
}

func (h *GroupingEventHandler) Flush() error {
	for id, balance := range h.currentAccs {
		q := balance.Quantiles()
		err := h.eventHandler.Snapshot(PredictionBalanceSnapshot{
			ID:         id,
			Day:        h.currentDate.ToStdTime().UnixMilli(),
			Balance:    balance.Mean(),
			LowerBound: q(h.q1),
			UpperBound: q(h.q2),
		})
		if err != nil {
			return err
		}
	}
	clear(h.currentAccs)
	return nil
}

func (h *GroupingEventHandler) Close() error {
	if err := h.Flush(); err != nil {
		return err
	}
	return h.eventHandler.Close()
}

package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"

	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/goslu/srvu"
	"github.com/SimonSchneider/goslu/static/shttp"
)

func SnapshotsTablePage(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		accounts, err := ListAccounts(ctx, db)
		if err != nil {
			return fmt.Errorf("listing accounts: %w", err)
		}
		ids := make([]string, len(accounts))
		for i, acc := range accounts {
			ids[i] = acc.ID
		}
		// Snapshots are ordered by date
		snapshots, err := ListAccountsSnapshots(ctx, db, ids)
		if err != nil {
			return fmt.Errorf("listing account snapshots: %w", err)
		}
		type DateIDKey struct {
			Date date.Date
			ID   string
		}
		dates := make([]date.Date, 0)
		snaps := make(map[DateIDKey]AccountSnapshot)
		for _, s := range snapshots {
			snaps[DateIDKey{Date: s.Date, ID: s.AccountID}] = s
			if len(dates) == 0 || dates[len(dates)-1] != s.Date {
				dates = append(dates, s.Date)
			}
		}
		slices.Reverse(dates)
		rows := make([]SnapshotsTableRow, 0)
		for _, d := range dates {
			rows = append(rows, SnapshotsTableRow{
				Date:      d,
				Snapshots: make([]AccountSnapshot, 0, len(accounts)),
			})
			for _, acc := range accounts {
				if snap, ok := snaps[DateIDKey{Date: d, ID: acc.ID}]; ok {
					rows[len(rows)-1].Snapshots = append(rows[len(rows)-1].Snapshots, snap)
				} else {
					rows[len(rows)-1].Snapshots = append(rows[len(rows)-1].Snapshots, AccountSnapshot{
						AccountID: acc.ID,
						Date:      d,
					})
				}
			}
		}

		return NewView(ctx, w, r).Render(Page("Snapshots Table", PageSnapshotsTable(&SnapshotsTableView{
			Accounts: accounts,
			Rows:     rows,
		})))
	})
}

func HandlerAccountSnapshotUpsert(db *sql.DB) http.Handler {
	return srvu.ErrHandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		var inp AccountSnapshotInput
		if err := srvu.Decode(r, &inp, false); err != nil {
			return fmt.Errorf("decoding input: %w", err)
		}
		accID := r.PathValue("id")
		var snap AccountSnapshot
		if inp.EmptyBalance {
			if err := DeleteAccountSnapshot(ctx, db, accID, inp.Date); err != nil {
				return fmt.Errorf("deleting existing snapshot: %w", err)
			}
			snap = AccountSnapshot{
				AccountID: accID,
				Date:      inp.Date,
			}
		} else {
			s, err := UpsertAccountSnapshot(ctx, db, accID, inp)
			if err != nil {
				return fmt.Errorf("upserting snapshot: %w", err)
			}
			snap = s
		}
		if r.Header.Get("HX-Request") == "true" {
			return NewView(ctx, w, r).Render(SnapshotCell(accID, inp.Date, snap))
		} else {
			shttp.RedirectToNext(w, r, fmt.Sprintf("/accounts/%s", accID))
		}
		return nil
	})
}

type SnapshotsTableView struct {
	*RequestDetails
	Accounts []Account
	Rows     []SnapshotsTableRow
}

type SnapshotsTableRow struct {
	Date      date.Date
	Snapshots []AccountSnapshot
}

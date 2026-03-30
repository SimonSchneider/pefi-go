package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/SimonSchneider/pefigo/internal/pdb"
)

const (
	settingDefaultCurrency    = "default_currency"
	settingForecastConfidence = "forecast_confidence"
	settingForecastSamples    = "forecast_samples"
)

func (s *Service) GetDefaultCurrency(ctx context.Context) (string, error) {
	val, err := s.q.GetSetting(ctx, settingDefaultCurrency)
	if errors.Is(err, sql.ErrNoRows) {
		return "SEK", nil
	}
	if err != nil {
		return "", fmt.Errorf("getting default currency: %w", err)
	}
	return val, nil
}

func (s *Service) SetDefaultCurrency(ctx context.Context, code string) error {
	if err := s.q.UpsertSetting(ctx, pdb.UpsertSettingParams{
		Key:   settingDefaultCurrency,
		Value: code,
	}); err != nil {
		return fmt.Errorf("setting default currency: %w", err)
	}
	return nil
}

func (s *Service) GetForecastConfidence(ctx context.Context) (float64, error) {
	val, err := s.q.GetSetting(ctx, settingForecastConfidence)
	if errors.Is(err, sql.ErrNoRows) {
		return 0.80, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting forecast confidence: %w", err)
	}
	f, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing forecast confidence: %w", err)
	}
	return f, nil
}

func (s *Service) SetForecastConfidence(ctx context.Context, confidence float64) error {
	if err := s.q.UpsertSetting(ctx, pdb.UpsertSettingParams{
		Key:   settingForecastConfidence,
		Value: strconv.FormatFloat(confidence, 'f', -1, 64),
	}); err != nil {
		return err
	}
	s.invalidateForecast()
	return nil
}

func (s *Service) GetForecastSamples(ctx context.Context) (int64, error) {
	val, err := s.q.GetSetting(ctx, settingForecastSamples)
	if errors.Is(err, sql.ErrNoRows) {
		return 10000, nil
	}
	if err != nil {
		return 0, fmt.Errorf("getting forecast samples: %w", err)
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parsing forecast samples: %w", err)
	}
	return n, nil
}

func (s *Service) SetForecastSamples(ctx context.Context, samples int64) error {
	if err := s.q.UpsertSetting(ctx, pdb.UpsertSettingParams{
		Key:   settingForecastSamples,
		Value: strconv.FormatInt(samples, 10),
	}); err != nil {
		return err
	}
	s.invalidateForecast()
	return nil
}

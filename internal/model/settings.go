package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/SimonSchneider/pefigo/internal/pdb"
)

const settingDefaultCurrency = "default_currency"

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

package finance

import (
	"github.com/SimonSchneider/goslu/date"
	"github.com/SimonSchneider/pefigo/internal/uncertain"
)

type SnapshotRecorder interface {
	OnSnapshot(accountID string, day date.Date, balance uncertain.Value) error
}

type SnapshotRecorderFunc func(accountID string, day date.Date, balance uncertain.Value) error

func (f SnapshotRecorderFunc) OnSnapshot(accountID string, day date.Date, balance uncertain.Value) error {
	return f(accountID, day, balance)
}

func EmptySnapshotRecorder() SnapshotRecorder {
	return SnapshotRecorderFunc(func(accountID string, day date.Date, balance uncertain.Value) error {
		return nil
	})
}

type TransferRecorder interface {
	OnTransfer(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error
}

type TransferRecorderFunc func(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error

func (f TransferRecorderFunc) OnTransfer(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error {
	return f(sourceAccountID, destinationAccountID, day, amount)
}
func EmptyTransferRecorder() TransferRecorder {
	return TransferRecorderFunc(func(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error {
		return nil
	})
}

type Recorder interface {
	SnapshotRecorder
	TransferRecorder
}

type CompositeRecorder struct {
	SnapshotRecorder
	TransferRecorder
}

func (r CompositeRecorder) OnSnapshot(accountID string, day date.Date, balance uncertain.Value) error {
	if r.SnapshotRecorder == nil {
		return nil
	}
	return r.SnapshotRecorder.OnSnapshot(accountID, day, balance)
}

func (r CompositeRecorder) OnTransfer(sourceAccountID, destinationAccountID string, day date.Date, amount uncertain.Value) error {
	if r.TransferRecorder == nil {
		return nil
	}
	return r.TransferRecorder.OnTransfer(sourceAccountID, destinationAccountID, day, amount)
}

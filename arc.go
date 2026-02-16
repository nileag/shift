package shift

import (
	"context"
	"database/sql"

	"github.com/luno/jettison/errors"
	"github.com/luno/jettison/j"
	"github.com/luno/reflex"
	"github.com/luno/reflex/rsql"
)

// NewArcFSM returns a new ArcFSM builder.
func NewArcFSM[T primary](events eventInserter[T], opts ...option) arcbuilder[T] {
	fsm := ArcFSM[T]{
		updates: make(map[int][]tuple),
		events:  events,
	}

	for _, opt := range opts {
		opt(&fsm.options)
	}

	return arcbuilder[T](fsm)
}

type arcbuilder[T primary] ArcFSM[T]

func (b arcbuilder[T]) Insert(st Status, inserter Inserter[T]) arcbuilder[T] {
	b.inserts = append(b.inserts, tuple{
		Status: st.ShiftStatus(),
		Type:   inserter,
	})
	return b
}

func (b arcbuilder[T]) Update(from, to Status, updater Updater[T]) arcbuilder[T] {
	tups := b.updates[from.ShiftStatus()]

	tups = append(tups, tuple{
		Status: to.ShiftStatus(),
		Type:   updater,
	})

	b.updates[from.ShiftStatus()] = tups

	return b
}

func (b arcbuilder[T]) Build() *ArcFSM[T] {
	fsm := ArcFSM[T](b)
	return &fsm
}

type tuple struct {
	Status int
	Type   interface{}
}

// ArcFSM is a defined Finite-State-Machine that allows specific mutations of
// the domain model in the underlying sql table via inserts and updates.
// All mutations update the status of the model, mutates some fields and
// inserts a reflex event.
//
// ArcFSM doesn't have the restriction of FSM and can be defined with arbitrary transitions.
type ArcFSM[T primary] struct {
	options
	events  eventInserter[T]
	inserts []tuple
	updates map[int][]tuple
}

// IsValidTransition validates status transition without committing the transaction
func (fsm *ArcFSM[T]) IsValidTransition(from Status, to Status) bool {
	s, ok := fsm.updates[from.ShiftStatus()]
	if !ok {
		return false
	}

	for _, tup := range s {
		if tup.Status == to.ShiftStatus() {
			return true
		}
	}
	return false
}

func (fsm *ArcFSM[T]) Insert(ctx context.Context, dbc *sql.DB, st Status, inserter Inserter[T]) (id T, err error) {
	tx, err := dbc.Begin()
	if err != nil {
		return id, err
	}
	defer tx.Rollback()

	id, notify, err := fsm.InsertTx(ctx, tx, st, inserter)
	if err != nil {
		return id, err
	}

	err = tx.Commit()
	if err != nil {
		return id, err
	}

	notify()
	return id, nil
}

func (fsm *ArcFSM[T]) InsertTx(ctx context.Context, tx *sql.Tx, st Status, inserter Inserter[T]) (id T, notify rsql.NotifyFunc, err error) {
	var found bool
	for _, tup := range fsm.inserts {
		if tup.Status == st.ShiftStatus() && sameType(tup.Type, inserter) {
			found = true
			break
		}
	}
	if !found {
		return id, nil, errors.Wrap(ErrInvalidStateTransition, "invalid insert status and inserter", j.KV("status", st.ShiftStatus()))
	}

	return insertTx(ctx, tx, st, inserter, fsm.events, reflex.EventType(st), fsm.options)
}

func (fsm *ArcFSM[T]) Update(ctx context.Context, dbc *sql.DB, from, to Status, updater Updater[T]) error {
	tx, err := dbc.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	notify, err := fsm.UpdateTx(ctx, tx, from, to, updater)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	notify()
	return nil
}

func (fsm *ArcFSM[T]) UpdateTx(ctx context.Context, tx *sql.Tx, from, to Status, updater Updater[T]) (rsql.NotifyFunc, error) {
	tl, ok := fsm.updates[from.ShiftStatus()]
	if !ok {
		return nil, errors.Wrap(ErrInvalidStateTransition, "invalid update from status", j.KV("shift.status", from.ShiftStatus()))
	}

	var found bool
	for _, tup := range tl {
		if tup.Status == to.ShiftStatus() && sameType(tup.Type, updater) {
			found = true
			break
		}
	}
	if !found {
		return nil, errors.Wrap(ErrInvalidStateTransition, "invalid update to status and updater", j.KV("shift.status", from.ShiftStatus()))
	}

	return updateTx(ctx, tx, from, to, updater, fsm.events, reflex.EventType(to), fsm.options)
}

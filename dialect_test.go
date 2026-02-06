package shift_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/luno/reflex"
	"github.com/luno/reflex/rsql"

	"github.com/nileag/shift"
)

var testDialect = getTestDialect()

func getTestDialect() string {
	d := os.Getenv("SHIFT_TEST_DIALECT")
	if d == "" {
		return "mysql"
	}
	return d
}

// ph returns the placeholder for the given 1-based parameter index.
// MySQL uses "?" and PostgreSQL uses "$N".
func ph(n int) string {
	if testDialect == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}

// pgEventType satisfies reflex.EventType for use in the custom PG loader.
type pgEventType int

func (t pgEventType) ReflexType() int { return int(t) }

// pgEventsInserter returns a PostgreSQL-compatible event inserter for rsql.
func pgEventsInserter(table string) func(ctx context.Context, dbc rsql.DBC, foreignID string, typ reflex.EventType, metadata []byte) error {
	return func(ctx context.Context, dbc rsql.DBC, foreignID string, typ reflex.EventType, metadata []byte) error {
		q := fmt.Sprintf(
			"INSERT INTO %s (foreign_id, timestamp, type, metadata) VALUES ($1, $2, $3, $4)",
			table,
		)
		_, err := dbc.ExecContext(ctx, q, foreignID, time.Now(), typ.ReflexType(), metadata)
		return err
	}
}

// pgEventsLoader returns a PostgreSQL-compatible event loader for rsql.
func pgEventsLoader(table string) func(ctx context.Context, dbc *sql.DB, prevCursor int64, lag time.Duration) ([]*reflex.Event, error) {
	return func(ctx context.Context, dbc *sql.DB, prevCursor int64, lag time.Duration) ([]*reflex.Event, error) {
		q := fmt.Sprintf(
			"SELECT id, foreign_id, timestamp, type, metadata FROM %s WHERE id > $1 ORDER BY id ASC LIMIT 1000",
			table,
		)
		rows, err := dbc.QueryContext(ctx, q, prevCursor)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var events []*reflex.Event
		for rows.Next() {
			var (
				id        int64
				foreignID string
				ts        time.Time
				typ       int
				metadata  []byte
			)
			if err := rows.Scan(&id, &foreignID, &ts, &typ, &metadata); err != nil {
				return nil, err
			}
			events = append(events, &reflex.Event{
				ID:        strconv.FormatInt(id, 10),
				ForeignID: foreignID,
				Timestamp: ts,
				Type:      pgEventType(typ),
				MetaData:  metadata,
			})
		}
		return events, rows.Err()
	}
}

func makeEventsInt(name string) *rsql.EventsTableInt {
	if testDialect == "postgres" {
		return rsql.NewEventsTableInt(name,
			rsql.WithEventsInserter(pgEventsInserter(name)),
			rsql.WithEventsLoader(pgEventsLoader(name)),
			rsql.WithoutEventsCache(),
		)
	}
	return rsql.NewEventsTableInt(name, rsql.WithoutEventsCache())
}

func makeEventsTable(name string) *rsql.EventsTable {
	if testDialect == "postgres" {
		return rsql.NewEventsTable(name,
			rsql.WithEventsInserter(pgEventsInserter(name)),
			rsql.WithEventsLoader(pgEventsLoader(name)),
			rsql.WithoutEventsCache(),
		)
	}
	return rsql.NewEventsTable(name)
}

// Wrapper methods that delegate to dialect-specific generated methods.
// These satisfy the shift.Inserter and shift.Updater interfaces.

func (i insert) Insert(ctx context.Context, tx *sql.Tx, st shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return i.InsertPostgres(ctx, tx, st)
	}
	return i.InsertMySQL(ctx, tx, st)
}

func (u update) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return u.UpdatePostgres(ctx, tx, from, to)
	}
	return u.UpdateMySQL(ctx, tx, from, to)
}

func (c complete) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return c.UpdatePostgres(ctx, tx, from, to)
	}
	return c.UpdateMySQL(ctx, tx, from, to)
}

func (ii i) Insert(ctx context.Context, tx *sql.Tx, st shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return ii.InsertPostgres(ctx, tx, st)
	}
	return ii.InsertMySQL(ctx, tx, st)
}

func (uu u) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return uu.UpdatePostgres(ctx, tx, from, to)
	}
	return uu.UpdateMySQL(ctx, tx, from, to)
}

func (it i_t) Insert(ctx context.Context, tx *sql.Tx, st shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return it.InsertPostgres(ctx, tx, st)
	}
	return it.InsertMySQL(ctx, tx, st)
}

func (ut u_t) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return ut.UpdatePostgres(ctx, tx, from, to)
	}
	return ut.UpdateMySQL(ctx, tx, from, to)
}

func (i2 insert2) Insert(ctx context.Context, tx *sql.Tx, st shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return i2.InsertPostgres(ctx, tx, st)
	}
	return i2.InsertMySQL(ctx, tx, st)
}

func (m move) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (int64, error) {
	if testDialect == "postgres" {
		return m.UpdatePostgres(ctx, tx, from, to)
	}
	return m.UpdateMySQL(ctx, tx, from, to)
}

func (is insertStr) Insert(ctx context.Context, tx *sql.Tx, st shift.Status) (string, error) {
	if testDialect == "postgres" {
		return is.InsertPostgres(ctx, tx, st)
	}
	return is.InsertMySQL(ctx, tx, st)
}

func (us updateStr) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (string, error) {
	if testDialect == "postgres" {
		return us.UpdatePostgres(ctx, tx, from, to)
	}
	return us.UpdateMySQL(ctx, tx, from, to)
}

func (cs completeStr) Update(ctx context.Context, tx *sql.Tx, from shift.Status, to shift.Status) (string, error) {
	if testDialect == "postgres" {
		return cs.UpdatePostgres(ctx, tx, from, to)
	}
	return cs.UpdateMySQL(ctx, tx, from, to)
}

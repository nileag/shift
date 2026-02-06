package shift_test

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"strconv"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

var mysqlSchemas = []string{`
  create temporary table users (
    id bigint not null auto_increment,
    name varchar(255) not null,
    dob datetime not null,
    amount varchar(255),

    status     tinyint not null,
    created_at datetime not null,
    updated_at datetime not null,

    primary key (id)
  );`, `
  create temporary table events (
    id bigint not null auto_increment,
    foreign_id bigint not null,
    timestamp datetime not null,
    type tinyint not null,
    metadata blob,

    primary key (id)
  );`, `
  create temporary table usersStr (
    id     varchar(255) not null,
    name   varchar(255) not null,
    dob    datetime     not null,
    amount varchar(255),

    status     tinyint  not null,
    created_at datetime not null,
    updated_at datetime not null,

    primary key (id)
  );`, `
  create temporary table eventsStr (
    id         bigint       not null auto_increment,
    foreign_id varchar(255) not null,
    timestamp  datetime     not null,
    type       tinyint      not null,
    metadata   blob,

    primary key (id)
  );`, `
  create temporary table tests (
    id bigint not null auto_increment,
    i1 bigint not null,
    i2 varchar(255) not null,
    i3 datetime not null,
    u1 bool,
    u2 varchar(255),
    u3 datetime,
    u4 varchar(255),
    u5 binary(64),

    status     tinyint not null,
    created_at datetime not null,
    updated_at datetime not null,

    primary key (id)
  );`}

var pgSchemas = []string{`
  create temporary table users (
    id bigserial primary key,
    name varchar(255) not null,
    dob timestamptz not null,
    amount varchar(255),

    status     smallint not null,
    created_at timestamptz not null,
    updated_at timestamptz not null
  );`, `
  create temporary table events (
    id bigserial primary key,
    foreign_id bigint not null,
    timestamp timestamptz not null,
    type smallint not null,
    metadata bytea
  );`, `
  create temporary table usersStr (
    id     varchar(255) not null primary key,
    name   varchar(255) not null,
    dob    timestamptz  not null,
    amount varchar(255),

    status     smallint    not null,
    created_at timestamptz not null,
    updated_at timestamptz not null
  );`, `
  create temporary table eventsStr (
    id         bigserial    primary key,
    foreign_id varchar(255) not null,
    timestamp  timestamptz  not null,
    type       smallint     not null,
    metadata   bytea
  );`, `
  create temporary table tests (
    id bigserial primary key,
    i1 bigint not null,
    i2 varchar(255) not null,
    i3 timestamptz not null,
    u1 boolean,
    u2 varchar(255),
    u3 timestamptz,
    u4 varchar(255),
    u5 bytea,

    status     smallint not null,
    created_at timestamptz not null,
    updated_at timestamptz not null
  );`}

func getDBTestURI() string {
	if env := os.Getenv("SHIFT_TEST_DB"); env != "" {
		return env
	}
	if testDialect == "postgres" {
		return ""
	}
	// Default MySQL connection via unix socket.
	sock := "/tmp/mysql.sock"
	if _, err := os.Stat(sock); os.IsNotExist(err) {
		sock = "/var/run/mysqld/mysqld.sock"
	}
	return "root@unix(" + sock + ")/test?"
}

func connect() (*sql.DB, error) {
	uri := getDBTestURI()
	if uri == "" {
		return nil, fmt.Errorf("no database URI configured (set SHIFT_TEST_DB)")
	}

	var dbc *sql.DB
	var err error

	if testDialect == "postgres" {
		dbc, err = sql.Open("pgx", uri)
	} else {
		dbc, err = sql.Open("mysql", uri+"parseTime=true&collation=utf8mb4_general_ci")
	}
	if err != nil {
		return nil, err
	}

	dbc.SetMaxOpenConns(1)

	if testDialect == "mysql" {
		if _, err := dbc.Exec("set time_zone='+00:00';"); err != nil {
			dbc.Close()
			return nil, fmt.Errorf("error setting db time_zone: %w", err)
		}
	}

	return dbc, nil
}

func setup(t *testing.T) *sql.DB {
	dbc, err := connect()
	if err != nil {
		t.Skip("Skipping: database not available:", err)
	}
	t.Cleanup(func() { require.NoError(t, dbc.Close()) })

	schemas := mysqlSchemas
	if testDialect == "postgres" {
		schemas = pgSchemas
	}

	for _, s := range schemas {
		_, err := dbc.Exec(s)
		require.NoError(t, err)
	}

	return dbc
}

// Currency is a custom "currency" type stored a string in the DB.
type Currency struct {
	Valid  bool
	Amount int64
}

func (c *Currency) Scan(src interface{}) error {
	var s sql.NullString
	if err := s.Scan(src); err != nil {
		return err
	}
	if !s.Valid {
		*c = Currency{
			Valid:  false,
			Amount: 0,
		}
		return nil
	}
	i, err := strconv.ParseInt(s.String, 10, 64)
	if err != nil {
		return err
	}
	*c = Currency{
		Valid:  true,
		Amount: i,
	}
	return nil
}

func (c Currency) Value() (driver.Value, error) {
	return strconv.FormatInt(c.Amount, 10), nil
}

package external

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	// mysql driver

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/lib/pq"
	"github.com/pkg/errors"
	sqltrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/database/sql"
	sqlxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/jmoiron/sqlx"
)

type PostgresDAO struct {
	ReadDB *sqlx.DB
	DB     *sqlx.DB
}

type Q interface {
	// Non contextual, to be deprecated soon
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	Exec(query string, args ...interface{}) (sql.Result, error)
	NamedExec(query string, arg interface{}) (sql.Result, error)
	NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Prepare(query string) (*sql.Stmt, error)

	// Contextual - missing named query context
	// only implemented for *sqlx.DB but NOT for *sqlx.Tx multiple PRs open to address this
	// https://github.com/jmoiron/sqlx/issues/447
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	NamedExecContext(ctx context.Context, query string, arg interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	// workaround for missing named query context, refactor when NamedQueryContext is implemented in *sqlx.Tx
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
}

type Settings struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     string `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
}

func validateSettings(s *Settings) error {
	missing := []string{}
	if s.Username == "" {
		missing = append(missing, "username")
	}
	if s.Password == "" {
		missing = append(missing, "password")
	}
	if s.Host == "" {
		missing = append(missing, "host")
	}
	if s.Port == "" {
		missing = append(missing, "port")
	}
	if s.Database == "" {
		missing = append(missing, "database")
	}
	if len(missing) > 0 {
		return fmt.Errorf("following values missing: %s", strings.Join(missing, ", "))
	}
	return nil
}

func getMasterSettings() (*Settings, error) {
	s := &Settings{
		Username: os.Getenv("MASTER_DB_USERNAME"),
		Password: os.Getenv("MASTER_DB_PASSWORD"),
		Host:     os.Getenv("MASTER_DB_HOST"),
		Port:     os.Getenv("MASTER_DB_PORT"),
		Database: os.Getenv("MASTER_DB_DATABASE"),
	}
	if err := validateSettings(s); err != nil {
		return nil, errors.Wrap(err, "validating master db settings")
	}
	return s, nil
}

func getReadSettings() (*Settings, error) {
	s := &Settings{
		Username: os.Getenv("READ_DB_USERNAME"),
		Password: os.Getenv("READ_DB_PASSWORD"),
		Host:     os.Getenv("READ_DB_HOST"),
		Port:     os.Getenv("READ_DB_PORT"),
		Database: os.Getenv("READ_DB_DATABASE"),
	}
	if err := validateSettings(s); err != nil {
		return nil, errors.Wrap(err, "validating read db settings")
	}
	return s, nil
}

func NewDB(s *Settings) (*sqlx.DB, error) {
	// Isolation level details: https://www.postgresql.org/docs/current/static/sql-set-transaction.html
	//
	// TLDR: The default transaction isolation level is "read comitted", which
	// is unlikely to do what you expect. With read committed, a transaction
	// can see the results of other concurrently run transaction commits
	// partway through its execution. Repeatable read or fully serialized give
	// much higher isolation guarentees, but at the cost of a larger number of
	// transaction abortions during concurrent access.

	uri := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s application_name=%s sslmode=require default_transaction_isolation='repeatable read'",
		s.Username,
		s.Password,
		s.Host,
		s.Port,
		s.Database,
		"api",
	)

	// REGISTER TRACING
	sqltrace.Register(
		"postgres",
		&pq.Driver{},
		sqltrace.WithServiceName("database"),
		sqltrace.WithAnalytics(true),
	)
	// Connect = Open + Ping
	db, err := sqlxtrace.Connect("postgres", uri)

	// If unable to connect, sleep a second and try again x5
	for i := 0; i < 5 && err != nil; i++ {
		time.Sleep(time.Second)
		db, err = sqlxtrace.Connect("postgres", uri)
	}

	// If still unable to connect, give up
	if err != nil {
		return nil, err
	}

	// Stops us having to add the db tags
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)

	return db, nil
}

func NewDAO() (*PostgresDAO, error) {
	masterSettings, err := getMasterSettings()
	if err != nil {
		return nil, errors.Wrap(err, "loading master db settings")
	}
	masterDB, err := NewDB(masterSettings)
	if err != nil {
		return nil, errors.Wrap(err, "creating master db conn")
	}

	readSettings, err := getReadSettings()
	if err != nil {
		return nil, errors.Wrap(err, "loading read db settings")
	}
	readDB, err := NewDB(readSettings)
	if err != nil {
		return nil, errors.Wrap(err, "creating read db conn")
	}
	return &PostgresDAO{
		DB:     masterDB,
		ReadDB: readDB,
	}, nil
}

func (d *PostgresDAO) GetReadOnlyTx(ctx context.Context) (*sqlx.Tx, error) {
	return d.ReadDB.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
}

func (d *PostgresDAO) GetTx(ctx context.Context) (*sqlx.Tx, error) {
	return d.DB.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly: false,
	})
}

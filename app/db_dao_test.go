package external_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-txdb"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	external "github.com/johankaito/api.external/app"
)

func init() {
	txdb.Register("pgx", "postgres", ConnString())
}

func NewTestDAO(t *testing.T) *external.PostgresDAO {
	db := NewTestDB(t, "")
	readDB := NewTestDB(t, "")

	// Comment out readonly for now -
	// at the moment the "2 dbs" are one, so making one readonly, breaks master db.
	// Cant just separate as too many tests rely on the 2 dbs being the
	// same underneath I THINK. Think we can emulate replication though.
	// readDB.MustExec("SET TRANSACTION READ ONLY")

	return &external.PostgresDAO{
		DB:     db,
		ReadDB: readDB,
	}
}

func ConnString() string {
	return fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable default_transaction_isolation='repeatable read'",
		"ggwp",
		"ggwp",
		"localhost",
		"6666",
		"ggwp",
	)
}

func NewTestDB(t *testing.T, suffix string) *sqlx.DB {
	// Connect = Open + Ping
	db, err := sqlx.Open("pgx", t.Name()+suffix)
	if err != nil {
		t.Fatal(err)
	}

	// Stops us having to add the db tags
	db.Mapper = reflectx.NewMapperFunc("json", strings.ToLower)

	return db
}

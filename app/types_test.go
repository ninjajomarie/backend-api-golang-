package external_test

import (
	"testing"

	external "github.com/johankaito/api.external/app"
)

type TestHelper struct {
	T *testing.T
}

type DAOFixture struct {
	*TestHelper
	DAO *external.PostgresDAO
}

type Auth struct {
	UserID       int
	AccessToken  string
	RefreshToken string
}

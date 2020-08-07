package external_test

import (
	"net/http"
	"os"
	"testing"

	external "github.com/johankaito/api.external/app"
	"github.com/sirupsen/logrus"
)

type Fixture struct {
	*DAOFixture
	*TestHelper
	Logger   *logrus.Entry
	Server   *external.External
	router   http.Handler
	Facebook *FakeFacebookClient
	Twitter  *FakeTwitterClient
}

func NewFixture(t *testing.T) *Fixture {
	if os.Getenv("PARALLEL_TESTS") != "" {
		t.Parallel()
	}
	logger := logrus.NewEntry(logrus.New()).WithField("version", "unit tests")

	// LOGGING ON - comment out if you want logging off
	// logger.Logger.Out = ioutil.Discard

	facebook := &FakeFacebookClient{}
	twitter := &FakeTwitterClient{}
	dao := NewTestDAO(t)
	server := external.New(logger, dao, facebook, twitter)
	testHelper := &TestHelper{T: t}

	handler, router, err := external.Router(server, logger, []string{})
	if err != nil {
		t.Fatalf("Couldn't create router: %v", err)
	}
	server.Router = router

	f := &Fixture{
		TestHelper: testHelper,
		DAOFixture: &DAOFixture{testHelper, dao},
		Server:     server,
		router:     handler,
		Logger:     logger,
		Facebook:   facebook,
		Twitter:    twitter,
	}

	return f
}

func (f *Fixture) InsertUser(
	newUser external.NewUser,
) int {
	var id int
	err := f.DAO.DB.Get(
		&id,
		`
			INSERT INTO ggwp.users
			(
				email, password_hash, user_type, last_online
			)
			VALUES
			(
				lower($1), $2, 'player', NOW()
			)
			RETURNING id
		`,
		newUser.Email,
		newUser.Password,
	)
	f.ExpectNoError(err)

	_, err = f.DAO.DB.Exec(
		`
			INSERT INTO ggwp.players
			(
				user_id, first_name, last_name, created_at, updated_at
			)
			VALUES
			(
				$1, $2, $3, NOW(), NOW()
			)
		`,
		id,
		newUser.FirstName,
		newUser.LastName,
	)
	f.ExpectNoError(err)

	return id
}

func (f *Fixture) InsertReferralCode(
	referralCode external.ReferralCode,
) {
	_, err := f.DAO.DB.NamedExec(
		`
			INSERT INTO ggwp.referral_codes
			(
				user_id,
				referral_code,
				referral_type,
				value,
				is_active
			)
			VALUES
			(
				:user_id, :referral_code, :referral_type, :value, :is_active
			)
			RETURNING id
		`,
		referralCode,
	)
	f.ExpectNoError(err)
}

func (f *Fixture) ExpectEmailWithReferralCode(
	referralCode external.ReferralCode,
) {
	_, err := f.DAO.DB.NamedExec(
		`
			INSERT INTO ggwp.referral_codes
			(
				user_id,
				referral_code,
				referral_type,
				value,
				is_active
			)
			VALUES
			(
				:user_id, :referral_code, :referral_type, :value, :is_active
			)
			RETURNING id
		`,
		referralCode,
	)
	f.ExpectNoError(err)
}

func (f *Fixture) InsertSocial(
	userID int,
	accessToken string,
	socialNetwork external.SocialNetwork,
) {
	_, err := f.DAO.DB.Exec(
		`
			INSERT INTO ggwp.social
			(
				user_id, access_token, social_network, is_active
			)
			VALUES
			(
				$1, $2, $3, TRUE
			)
		`,
		userID,
		accessToken,
		socialNetwork,
	)
	f.ExpectNoError(err)
}

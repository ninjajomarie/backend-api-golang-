package external_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	fb "github.com/huandu/facebook"
	external "github.com/johankaito/api.external/app"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleLoginHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	password := "test123"
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	f.ExpectNoError(err)
	f.InsertUser(external.NewUser{
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
	})

	rr := f.UnAuthedRequest(
		http.MethodPost,
		"/user/login",
		fmt.Sprintf(`{
			"email": "%s",
			"password": "%s"
		}`, email, password),
	)
	f.ExpectStatus(rr, http.StatusOK)

	// auth headers set
	f.ExpectAuthHeaders(rr)
}

func TestHandleSocialLoginFacebookHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	firstName := "Test"
	lastName := "User"
	newAccessToken := "new-token"
	socialNetwork := external.SocialNetwork_Facebook
	userID := f.InsertUser(external.NewUser{
		Email:     email,
		Password:  "",
		FirstName: firstName,
		LastName:  lastName,
	})
	f.InsertSocial(userID, "old-token", socialNetwork)

	// setup facebook response
	f.Facebook.GetResponse = fb.Result{
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
	}

	rr := f.UnAuthedRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/user/social/login?access_token=%s&social_network=%s&access_secret",
			newAccessToken, socialNetwork,
		),
		"",
	)

	// ok
	f.ExpectStatus(rr, http.StatusOK)
	// auth set
	f.ExpectAuthHeaders(rr)
	// facebook called
	f.ExpectDeepEq(f.Facebook.GetSessionCallCount, 1)
	f.ExpectDeepEq(f.Facebook.GetSessionLastCalledWith, newAccessToken)
	f.ExpectDeepEq(f.Facebook.GetLastCalledWith.Path, "/me")
	// social token saved
	f.ExpectRowCountWhere(
		"ggwp.social",
		fmt.Sprintf("access_token='%s'", newAccessToken),
		1,
	)
	// auth headers set
	f.ExpectAuthHeaders(rr)
	// user obj returned
	var response struct {
		User external.User `json:"user"`
	}
	f.Bind(rr, &response)
	f.ExpectDeepEq(response.User.Email, email)
}

func TestHandleSocialSignUpFacebookHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	firstName := "Test"
	lastName := "User"
	accessToken := "access-token"
	socialNetwork := external.SocialNetwork_Facebook

	// setup facebook response
	f.Facebook.GetResponse = fb.Result{
		"id":         "1234",
		"gender":     "male",
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"birthday":   "01/20/1990",
	}

	rr := f.UnAuthedRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/user/social/signup?access_token=%s&social_network=%s&access_secret",
			accessToken, socialNetwork,
		),
		"",
	)

	// ok
	f.ExpectStatus(rr, http.StatusOK)
	// auth set
	f.ExpectAuthHeaders(rr)
	// facebook called
	f.ExpectDeepEq(f.Facebook.GetSessionCallCount, 1)
	f.ExpectDeepEq(f.Facebook.GetSessionLastCalledWith, accessToken)
	f.ExpectDeepEq(f.Facebook.GetLastCalledWith.Path, "/me")
	// social token saved
	f.ExpectRowCountWhere(
		"ggwp.social",
		fmt.Sprintf("access_token='%s'", accessToken),
		1,
	)
	// user inserted
	f.ExpectRowCountWhere(
		"ggwp.users",
		fmt.Sprintf("email = '%s'", email),
		1,
	)
	// referral code inserted for new user
	f.ExpectRowCountWithJoinWhere(
		"ggwp.referral_codes c",
		"JOIN ggwp.users u ON u.id = c.user_id",
		fmt.Sprintf("u.email = '%s'", email),
		1,
	)
	// auth headers set
	f.ExpectAuthHeaders(rr)
	// user obj returned
	var response struct {
		User external.User `json:"user"`
	}
	f.Bind(rr, &response)
	f.ExpectDeepEq(response.User.Email, email)
	f.ExpectDeepEq(response.User.Player.FirstName, firstName)
	f.ExpectDeepEq(response.User.Player.LastName, lastName)
}

func TestHandleSocialSignUpFacebookEmailExists(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	firstName := "Test"
	lastName := "User"
	accessToken := "access-token"
	socialNetwork := external.SocialNetwork_Facebook

	// setup existing user
	userID := f.InsertUser(external.NewUser{
		Email:     email,
		Password:  "",
		FirstName: firstName,
		LastName:  lastName,
	})
	f.InsertSocial(userID, "a-token", socialNetwork)
	// setup facebook response
	f.Facebook.GetResponse = fb.Result{
		"id":         "1234",
		"gender":     "male",
		"email":      email,
		"first_name": firstName,
		"last_name":  lastName,
		"birthday":   "01/20/1990",
	}

	rr := f.UnAuthedRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/user/social/signup?access_token=%s&social_network=%s&access_secret",
			accessToken, socialNetwork,
		),
		"",
	)
	f.ExpectBodyContains(rr, "associated user email already exists, please login instead")

	// bad requests
	f.ExpectStatus(rr, http.StatusBadRequest)
	// no auth set
	f.ExpectNoAuthHeaders(rr)
	// facebook called
	f.ExpectDeepEq(f.Facebook.GetSessionCallCount, 1)
	f.ExpectDeepEq(f.Facebook.GetSessionLastCalledWith, accessToken)
	f.ExpectDeepEq(f.Facebook.GetLastCalledWith.Path, "/me")
	// social token not saved
	f.ExpectRowCountWhere(
		"ggwp.social",
		fmt.Sprintf("access_token='%s'", accessToken),
		0,
	)
}

func TestHandleSocialLoginTwitterHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	firstName := "Test"
	lastName := "User"
	newAccessToken := "new-token"
	newAccessSecret := "new-secret"
	socialNetwork := external.SocialNetwork_Twitter
	userID := f.InsertUser(external.NewUser{
		Email:     email,
		Password:  "",
		FirstName: firstName,
		LastName:  lastName,
	})
	f.InsertSocial(userID, "old-token", socialNetwork)

	// setup twitter response
	f.Twitter.VerifyCredentialsResponse = &external.TwitterUser{
		FirstName:    firstName,
		LastName:     lastName,
		EmailAddress: email,
	}

	rr := f.UnAuthedRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/user/social/login?access_token=%s&social_network=%s&access_secret=%s",
			newAccessToken, socialNetwork, newAccessSecret,
		),
		"",
	)

	// ok
	f.ExpectStatus(rr, http.StatusOK)
	// auth set
	f.ExpectAuthHeaders(rr)
	// facebook called
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsCallCount, 1)
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsLastCalledWith.AccessToken, newAccessToken)
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsLastCalledWith.AccessSecret, newAccessSecret)
	// social token saved
	f.ExpectRowCountWhere(
		"ggwp.social",
		fmt.Sprintf("access_token='%s'", newAccessToken),
		1,
	)
	// auth headers set
	f.ExpectAuthHeaders(rr)
	// user obj returned
	var response struct {
		User external.User `json:"user"`
	}
	f.Bind(rr, &response)
	f.ExpectDeepEq(response.User.Email, email)
}

func TestHandleSocialSignUpTwitterHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	firstName := "Test"
	lastName := "User"
	accessToken := "new-token"
	accessSecret := "new-secret"
	socialNetwork := external.SocialNetwork_Twitter

	// setup twitter response
	f.Twitter.VerifyCredentialsResponse = &external.TwitterUser{
		FirstName:    firstName,
		LastName:     lastName,
		EmailAddress: email,
	}

	rr := f.UnAuthedRequest(
		http.MethodPost,
		fmt.Sprintf(
			"/user/social/signup?access_token=%s&access_secret=%s&social_network=%s",
			accessToken, accessSecret, socialNetwork,
		),
		"",
	)

	// ok
	f.ExpectStatus(rr, http.StatusOK)
	// auth set
	f.ExpectAuthHeaders(rr)
	// twitter called
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsCallCount, 1)
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsLastCalledWith.AccessToken, accessToken)
	f.ExpectDeepEq(f.Twitter.VerifyCredentialsLastCalledWith.AccessSecret, accessSecret)
	// social token saved
	f.ExpectRowCountWhere(
		"ggwp.social",
		fmt.Sprintf("access_token='%s'", accessToken),
		1,
	)

	// user inserted
	f.ExpectRowCountWhere(
		"ggwp.users",
		fmt.Sprintf("email = '%s'", email),
		1,
	)
	// referral code inserted for new user
	f.ExpectRowCountWithJoinWhere(
		"ggwp.referral_codes c",
		"JOIN ggwp.users u ON u.id = c.user_id",
		fmt.Sprintf("u.email = '%s'", email),
		1,
	)
	// auth headers set
	f.ExpectAuthHeaders(rr)
	// user obj returned
	var response struct {
		User external.User `json:"user"`
	}
	f.Bind(rr, &response)
	f.ExpectDeepEq(response.User.Email, email)
	f.ExpectDeepEq(response.User.Player.FirstName, firstName)
	f.ExpectDeepEq(response.User.Player.LastName, lastName)
}

func TestHandleSignUpHappyPath(t *testing.T) {
	f := NewFixture(t)
	defer f.Close()

	email := "test.user@ggwpacademy.com"
	newUser := &external.NewUser{
		Email:     email,
		Password:  "test123",
		FirstName: "Test",
		LastName:  "User",
	}
	j, err := json.Marshal(newUser)
	f.ExpectNoError(err)

	rr := f.UnAuthedRequest(
		http.MethodPost,
		"/user",
		string(j),
	)
	f.ExpectStatus(rr, http.StatusOK)

	// auth headers set
	f.ExpectAuthHeaders(rr)

	// user row created
	f.ExpectRowCountWhere("ggwp.users", fmt.Sprintf("email = '%s'", email), 1)

	// referral code inserted for new user
	f.ExpectRowCountWithJoinWhere(
		"ggwp.referral_codes c",
		"JOIN ggwp.users u ON u.id = c.user_id",
		fmt.Sprintf("u.email = '%s'", email),
		1,
	)
}

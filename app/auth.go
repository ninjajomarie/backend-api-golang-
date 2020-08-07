package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/AmirSoleimani/VoucherCodeGenerator/vcgen"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	fb "github.com/huandu/facebook"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var accessTokenExpiration = 5 * time.Minute // 5 minutes
var AccessTokenHeader = "Authorization"
var refreshTokenExpiration = 60 * 24 * time.Hour // 60 days
var RefreshTokenHeader = "X-Ggwp-Refresh-Token"
var passwordResetTokenExpiration = 10 * time.Minute // 10 minutes

type AccessToken struct {
	UserID   int    `json:"user_id,omitempty"`
	UserType string `json:"user_type,omitempty"`
	jwt.StandardClaims
}

type RefreshToken struct {
	UserID int `json:"userID,omitempty"`
	jwt.StandardClaims
}

type PasswordResetRequest struct {
	Token        string `json:"password_reset_token,omitempty"`
	EmailAddress string `json:"email_address,omitempty"`
	NewPassword  string `json:"new_password,omitempty"`
}

func (e *External) HandlePing(w http.ResponseWriter, r *http.Request) {
	e.returnJSON(w, "pong")
}

func (e *External) HandleLogin(w http.ResponseWriter, r *http.Request) {
	account := &Account{}
	err := json.NewDecoder(r.Body).Decode(account) // decode the request body into struct and failed if any error occur
	if err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	user, err := GetUserByEmail(e.dao.ReadDB, account.Email)
	if err != nil && err == sql.ErrNoRows {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid password or email address"))
		return
	} else if err != nil {
		e.writeError(w, r, http.StatusBadRequest, errors.Wrap(err, "getting user by email"))
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(account.Password),
	)
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { // password does not match!
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid password or email address"))
		return
	} else if err != nil {
		e.writeError(w, r, http.StatusBadRequest, err)
		return
	}

	userID := user.ID
	user, err = e.GetUserByID(user.ID, r.Context().Value("device_unique_id").(string))
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "log in getting user by id: %d", userID),
		)
		return
	}
	if err := e.createAndWriteJWTTokens(w, r, user, true); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT tokens for user id: %d", user.ID),
		)
		return
	}

	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func (e *External) HandleSocialLogin(w http.ResponseWriter, r *http.Request) {
	accessToken := mux.Vars(r)["access_token"]
	accessSecret := mux.Vars(r)["access_secret"]

	var socialNetwork SocialNetwork
	socialNetwork.Scan(mux.Vars(r)["social_network"])

	var emailAddress string
	// only FB and Twitter login is currently supported
	switch socialNetwork {
	case SocialNetwork_Facebook:
		// validate token
		session := e.facebook.GetSession(accessToken)
		if err := session.Validate(); err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "validating access token"),
			)
			return
		}
		// valid token, proceed to use it
		res, err := session.Get("/me", fb.Params{
			"fields":       "email",
			"access_token": accessToken,
		})
		if err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "getting user details from facebook"),
			)
			return
		}
		res.DecodeField("email", &emailAddress)

	case SocialNetwork_Twitter:
		user, err := e.twitter.VerifyCredentials(VerifyCredentialsParams{accessToken, accessSecret})
		if err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "getting user details from twitter"),
			)
			return
		}
		emailAddress = user.EmailAddress

	case SocialNetwork_Unknown, SocialNetwork_Instagram:
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("login with %s is currently not supported", socialNetwork))
		return
	}

	// confirm that user email is in our system
	user, err := GetUserByEmail(e.dao.ReadDB, emailAddress)
	if err != nil && err == sql.ErrNoRows {
		e.writeError(
			w, r, http.StatusBadRequest,
			fmt.Errorf(
				"unknown email address: %s, please sign up with %s instead",
				emailAddress, socialNetwork,
			),
		)
		return
	} else if err != nil {
		e.writeError(w, r, http.StatusBadRequest, errors.Wrap(err, "getting user by email"))
		return
	}

	userID := user.ID
	user, err = e.GetUserByID(user.ID, r.Context().Value("device_unique_id").(string))
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "log in getting user by id: %d", userID),
		)
		return
	}
	if err := e.createAndWriteJWTTokens(w, r, user, true); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT tokens for user id: %d", user.ID),
		)
		return
	}

	// update auth token
	if err = UpdateSocialToken(e.dao.DB, userID, accessToken, accessSecret, socialNetwork); err != nil {
		e.writeError(
			w, r, http.StatusBadRequest,
			errors.Wrap(err, "updating social token"),
		)
		return
	}

	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func (e *External) HandleSocialSignUp(w http.ResponseWriter, r *http.Request) {
	accessToken := mux.Vars(r)["access_token"]
	accessSecret := mux.Vars(r)["access_secret"]

	var socialNetwork SocialNetwork
	socialNetwork.Scan(mux.Vars(r)["social_network"])

	var emailAddress string
	var lastName string
	var firstName string

	switch socialNetwork {
	case SocialNetwork_Facebook:
		// validate token
		session := e.facebook.GetSession(accessToken)
		if err := session.Validate(); err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "validating access token"),
			)
			return
		}

		// valid token, proceed to use it
		res, err := session.Get("/me", fb.Params{
			"fields":       "email, last_name, first_name",
			"access_token": accessToken,
		})
		if err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "getting user details from facebook"),
			)
			return
		}
		res.DecodeField("email", &emailAddress)
		res.DecodeField("last_name", &lastName)
		res.DecodeField("first_name", &firstName)

	case SocialNetwork_Twitter:
		user, err := e.twitter.VerifyCredentials(VerifyCredentialsParams{accessToken, accessSecret})
		if err != nil {
			e.writeError(
				w, r, http.StatusBadRequest,
				errors.Wrap(err, "getting user details from twitter"),
			)
			return
		}
		emailAddress = user.EmailAddress
		firstName = user.FirstName
		lastName = user.LastName

	default:
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("sign up with %s is currently not supported", socialNetwork))
		return
	}
	l := e.log.WithFields(logrus.Fields{
		"email":      emailAddress,
		"last_name":  lastName,
		"first_name": firstName,
	})
	l.Infof("social sign up user info: %s", socialNetwork)

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(emailAddress),
		bcrypt.DefaultCost,
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		l.Error("creating transaction")
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
	}
	defer tx.Rollback()

	// confirm that user email is NOT in our system
	_, err = GetUserByEmail(tx, emailAddress)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "checking user by email"),
		)
		return
	} else if err == nil {
		e.writeError(
			w, r, http.StatusBadRequest,
			fmt.Errorf("associated user email already exists, please login instead"),
		)
		return
	}

	// sign up flow
	user, err := CreatePlayer(tx, &NewUser{
		Email:        emailAddress,
		Password:     string(hashedPassword),
		FirstName:    firstName,
		LastName:     lastName,
		ReferralCode: "",
	})
	if err != nil {
		l.Error("creating player")
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "creating player"),
		)
		return
	}
	l2 := l.WithField("user_id", user.ID)

	// generate and inject referral code for user
	referralCode, err := GenerateReferralCode(tx, user.ID)
	if err != nil {
		l2.Error("unable to get generated referral code on sign up")
		return
	}
	user.ReferralCode = referralCode
	user.PasswordHash = ""

	// insert social token
	if err = InsertSocialToken(tx, user.ID, accessToken, accessSecret, socialNetwork); err != nil {
		e.writeError(
			w, r, http.StatusBadRequest,
			errors.Wrap(err, "inserting social token"),
		)
		return
	}

	// link this user id to a waitlist user if applicable
	if err := AddUserIDToWaitlistIfApplicable(e.dao.DB, user.ID, user.Email); err != nil {
		l2.Error("unable to add user id to waitlist")
	}

	// commit changes
	if err := tx.Commit(); err != nil {
		l2.Error("unable to commit tx for referral code on sign up")
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "comitting user sign up tx"),
		)
		return
	}

	if err := e.createAndWriteJWTTokens(w, r, user, true); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT tokens for user: %v", user.ID),
		)
		return
	}

	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func (e *External) HandleSignUp(w http.ResponseWriter, r *http.Request) {
	newUser := &NewUser{}
	// decode the request body into struct and failed if any error occur
	if err := json.NewDecoder(r.Body).Decode(newUser); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newUser.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	newUser.Password = string(hashedPassword)

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
	}
	defer tx.Rollback()

	if ok, err := newUser.IsValid(); !ok {
		e.writeError(
			w, r, http.StatusBadRequest, errors.Wrapf(err, "missing attribute"),
		)
		return
	}

	// checks
	// check if email exists
	_, err = GetUserByEmail(tx, newUser.Email)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "checking user by email"),
		)
		return
	} else if err == nil {
		e.writeError(
			w, r, http.StatusBadRequest,
			fmt.Errorf("email exists, please login instead"),
		)
		return
	}

	// sign up flow
	user, err := CreatePlayer(tx, newUser)
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "creating player"),
		)
		return
	}

	// commit changes
	if err := tx.Commit(); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "comitting tx"),
		)
		return
	}

	// generate and inject referral code for user
	tx, err = e.dao.GetTx(r.Context())
	if err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{
			"user_id": user.ID,
		}).Error("unable to get tx for referral code setup on sign up")
	}
	defer tx.Rollback()
	referralCode, err := GenerateReferralCode(tx, user.ID)
	if err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{
			"user_id": user.ID,
		}).Error("unable to get generated referral code on sign up")
		return
	}
	if err := tx.Commit(); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{
			"user_id": user.ID,
		}).Error("unable to commit tx for referral code on sign up")
		return
	}
	user.ReferralCode = referralCode

	// link this user id to a waitlist user if applicable
	if err := AddUserIDToWaitlistIfApplicable(e.dao.DB, user.ID, user.Email); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{
			"user_id": user.ID,
		}).Error("unable to add user id to waitlist")
	}

	if err := e.createAndWriteJWTTokens(w, r, user, true); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT tokens for user: %v", user.ID),
		)
		return
	}

	user.PasswordHash = ""
	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func (e *External) createAndWriteJWTTokens(
	w http.ResponseWriter, r *http.Request, user *User, writeRefreshToken bool,
) error {
	// create JWT token
	now := time.Now()
	atk := &AccessToken{
		user.ID,
		user.UserAdminLevel,
		jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(accessTokenExpiration).Unix(),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), atk)
	accessTokenStr, err := accessToken.SignedString([]byte(os.Getenv("TOKEN_PASSWORD")))
	if err != nil {
		return errors.Wrap(err, "signing access token")
	}
	w.Header().Set(AccessTokenHeader, fmt.Sprintf("Bearer %s", accessTokenStr))

	if writeRefreshToken {
		rtk := &RefreshToken{
			UserID: user.ID,
			StandardClaims: jwt.StandardClaims{
				IssuedAt:  now.Unix(),
				ExpiresAt: now.Add(refreshTokenExpiration).Unix(),
			},
		}
		refreshToken := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), rtk)
		refreshTokenStr, err := refreshToken.SignedString([]byte(os.Getenv("TOKEN_PASSWORD")))
		if err != nil {
			return errors.Wrap(err, "signing refresh token")
		}

		w.Header().Set(RefreshTokenHeader, refreshTokenStr)
	}

	// update last online
	if err := UpdateLastOnline(e.dao.ReadDB, user.ID); err != nil {
		e.log.WithError(err).WithFields(logrus.Fields{
			"user_id": user.ID,
		}).Error("updating last online")
	}

	return nil
}

func (e *External) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	tokenString := r.Header.Get(RefreshTokenHeader) // grab the token from the header

	if tokenString == "" { // token is missing, returns with error code 403 Unauthorized
		e.writeError(w, r, http.StatusForbidden, fmt.Errorf("missing refresh token"))
		return
	}

	tk := &RefreshToken{}
	token, err := jwt.ParseWithClaims(tokenString, tk, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("TOKEN_PASSWORD")), nil
	})
	if err != nil { // malformed token, returns with http code 403 as usual
		e.writeError(w, r, http.StatusForbidden, fmt.Errorf("malformed refresh token"))
		return
	}

	if !token.Valid { // token is invalid, maybe not signed on this server
		e.writeError(w, r, http.StatusForbidden, fmt.Errorf("token is not valid"))
		return
	}

	user, err := GetUserByID(e.dao.ReadDB, tk.UserID)
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "getting user by id: %d", user.ID),
		)
		return
	}

	if err := e.createAndWriteJWTTokens(w, r, user, false); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT access token for user: %d", user.ID),
		)
		return
	}

	user.PasswordHash = ""
	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func (e *External) HandlePasswordChange(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(int)
	req := &PasswordChangeRequest{}
	// decode the request body into struct and failed if any error occur
	if err := json.NewDecoder(r.Body).Decode(req); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	user, err := GetUserByID(e.dao.ReadDB, userID)
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "getting user by id: %d", user.ID),
		)
		return
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.CurrentPassword),
	)
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword { // password does not match!
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("wrong current password"))
		return
	} else if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(req.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError,
			errors.Wrap(err, "unable to generate hash for password"),
		)
		return
	}

	if err := UpdateUserPassword(e.dao.DB, userID, string(hashedPassword)); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "updating user password"))
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	fP := &ForgotPassword{}
	if err := json.NewDecoder(r.Body).Decode(fP); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	user, err := GetUserByEmail(e.dao.ReadDB, fP.EmailAddress)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	if err == sql.ErrNoRows {
		// prevent leaking information on emails addresses we have on record
		e.returnJSON(w, nil)
		return
	}

	// check there currently exists an active password reset token
	resets, err := GetAllPasswordReset(e.dao.ReadDB, user.ID)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(
			w, r, http.StatusInternalServerError,
			fmt.Errorf("checking for existing password reset tokens"),
		)
		return
	}
	var token string
	if len(resets) > 0 {
		for _, r := range resets {
			if isActiveResetToken(r) {
				// has an active password reset token
				token = r.Token
				break
			}
		}
	}

	if token == "" {
		// Generate password reset token
		vcPrefix := vcgen.New(
			&vcgen.Generator{
				Count:   1,
				Pattern: "######",
			},
		)
		res, err := vcPrefix.Run()
		if err != nil {
			e.writeError(
				w, r, http.StatusInternalServerError,
				fmt.Errorf(
					"unable to generate password reset token",
				),
			)
			return
		}
		if res == nil {
			e.writeError(
				w, r, http.StatusInternalServerError,
				fmt.Errorf(
					"no password reset tokens returned from library",
				),
			)
			return
		}
		generatedRefCodes := *res
		if len(generatedRefCodes) != 1 {
			e.writeError(
				w, r, http.StatusInternalServerError,
				fmt.Errorf(
					"invalid number of generated password reset tokens",
				),
			)
			return
		}
		token = generatedRefCodes[0]

		if err := CreatePasswordReset(e.dao.DB, user.ID, token); err != nil {
			e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "unable to create password reset token"))
			return
		}
	}

	m := NewMailer(e.log)
	if err := m.SendForgotPassword(r.Context(), user.Email, token); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleResetPassword(w http.ResponseWriter, r *http.Request) {
	pR := &PasswordResetRequest{}
	if err := json.NewDecoder(r.Body).Decode(pR); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	user, err := GetUserByEmail(e.dao.ReadDB, pR.EmailAddress)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}
	if user.Email == "" {
		e.writeError(
			w, r, http.StatusBadRequest,
			fmt.Errorf(
				"missing user email",
			),
		)
		return
	}

	reset, err := GetPasswordReset(e.dao.ReadDB, user.ID, pR.Token)
	if err != nil && err == sql.ErrNoRows {
		e.writeError(
			w, r, http.StatusBadRequest,
			fmt.Errorf("unknown password reset token"),
		)
		return
	}
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "unable to get password reset"))
		return
	}
	if !isActiveResetToken(reset) {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("reset token has expired"))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(pR.NewPassword),
		bcrypt.DefaultCost,
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError,
			errors.Wrap(err, "unable to generate hash for password"),
		)
		return
	}

	if err := UpdateUserPassword(e.dao.DB, user.ID, string(hashedPassword)); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "updating user password"))
		return
	}

	if err := e.createAndWriteJWTTokens(w, r, user, true); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrapf(err, "creating JWT tokens for user id: %d", user.ID),
		)
		return
	}

	user.PasswordHash = ""
	e.returnJSON(w, struct {
		User *User `json:"user,omitempty"`
	}{
		User: user,
	})
}

func isActiveResetToken(r *PasswordReset) bool {
	return !time.Now().After(r.CreatedAt.Add(passwordResetTokenExpiration))
}

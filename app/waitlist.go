package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AmirSoleimani/VoucherCodeGenerator/vcgen"
	"github.com/badoux/checkmail"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func (e *External) HandleWaitlistUserAdd(w http.ResponseWriter, r *http.Request) {
	wR := &WaitlistRequest{}
	// decode the request body into struct and failed if any error occur
	if err := json.NewDecoder(r.Body).Decode(wR); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}

	if wR.EmailAddress == "" {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("missing email address"))
		return
	}

	if err := checkmail.ValidateFormat(wR.EmailAddress); err != nil {
		e.writeError(w, r, http.StatusBadRequest, errors.Wrap(err, "validating email address format"))
		return
	}

	// TODO: Bring this back and attempt to send user an email for manual verification
	// if err := checkmail.ValidateHost(wR.EmailAddress); err != nil {
	// e.writeError(w, r, http.StatusBadRequest, errors.Wrap(err, "validating email address host"))
	// return
	// }

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
		return
	}
	defer tx.Rollback()

	// check if email exists
	_, err = GetUserByEmail(tx, wR.EmailAddress)
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

	// Get waitlist code
	vcPrefix := vcgen.New(
		&vcgen.Generator{
			Count: 1, Pattern: "######", Prefix: "WAIT-",
		},
	)
	res, err := vcPrefix.Run()
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, fmt.Errorf("unable to generate ref code"))
		return
	}
	if res == nil {
		e.writeError(w, r, http.StatusInternalServerError, fmt.Errorf("no ref codes returned from library"))
		return
	}
	generatedRefCodes := *res
	if len(generatedRefCodes) != 1 {
		e.writeError(w, r, http.StatusInternalServerError, fmt.Errorf("invalid number of generated ref codes"))
		return
	}
	ownerWaitlistCode := generatedRefCodes[0]

	// check if any of the original codes have been provided
	// if so, check validity
	var waitlistCodeID *int
	if wR.OriginalWaitlistCode != "" {
		waitlistItem, err := GetWaitlistItemByCode(tx, wR.OriginalWaitlistCode)
		if err != nil {
			if err == sql.ErrNoRows {
				e.writeError(
					w, r, http.StatusBadRequest,
					fmt.Errorf("waitlist code provided does not exist"),
				)
				return
			}
			e.writeError(
				w, r, http.StatusInternalServerError,
				errors.Wrapf(err, "getting waitlist item by code: %s", wR.OriginalWaitlistCode),
			)
			return
		}
		waitlistCodeID = &waitlistItem.ID
	}

	var referralCodeID *int
	if wR.OriginalReferralCode != "" {
		referralCode, err := GetReferralCodeByCode(tx, wR.OriginalReferralCode)
		if err != nil {
			if err == sql.ErrNoRows {
				e.writeError(
					w, r, http.StatusBadRequest,
					fmt.Errorf("referral code provided does not exist"),
				)
				return
			}
			e.writeError(
				w, r, http.StatusInternalServerError,
				errors.Wrapf(err, "getting referral code by code: %s", wR.OriginalReferralCode),
			)
			return
		}
		referralCodeID = &referralCode.ID
	}

	createdItem, err := CreateWaitlistItem(
		tx,
		wR.EmailAddress,
		ownerWaitlistCode,
		referralCodeID,
		waitlistCodeID,
	)
	if err != nil {
		pqErr, ok := err.(*pq.Error)
		if ok && pqErr.Code == "23505" {
			e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("email already in waitlist"))
			return
		}
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "creating waitlist item"))
		return
	}

	if err := tx.Commit(); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "comitting tx"),
		)
		return
	}

	e.returnJSON(w, createdItem)
}

func (e *External) HandleGetWaitlist(w http.ResponseWriter, r *http.Request) {
	l, err := GetWaitlist(e.dao.ReadDB)
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError,
			errors.Wrap(err, "getting waitlist"),
		)
		return
	}

	e.returnJSON(w, l)
}

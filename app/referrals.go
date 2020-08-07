package external

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/AmirSoleimani/VoucherCodeGenerator/vcgen"
	"github.com/pkg/errors"
)

func GenerateReferralCode(q Q, userID int) (*ReferralCode, error) {
	vcPrefix := vcgen.New(
		&vcgen.Generator{
			Count: 1, Pattern: "######", Prefix: "WELC-",
		},
	)
	res, err := vcPrefix.Run()
	if err != nil {
		return nil, fmt.Errorf("unable to generate ref code")
	}
	if res == nil {
		return nil, fmt.Errorf("no ref codes returned from library")
	}

	generatedRefCodes := *res
	if len(generatedRefCodes) != 1 {
		return nil, fmt.Errorf("invalid number of generated ref codes")
	}
	if err := CreateReferralCode(q, userID, generatedRefCodes[0]); err != nil {
		return nil, errors.Wrapf(err, "unable to save referral code: %s", generatedRefCodes[0])
	}
	referralCode, err := GetReferralCodeByUserID(q, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get generated referral code ref_code: %s", generatedRefCodes[0])
	}

	return referralCode, nil
}

func ValidateReferralCode(q Q, referralCode string) (*ReferralCode, bool, error) {
	inviteReferralCode, err := GetReferralCodeByCode(q, referralCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, false, fmt.Errorf("unknown referral code used: %s", referralCode)
		} else if err != sql.ErrNoRows {
			return nil, false, errors.Wrapf(
				err, "unable to get referral code: %s", referralCode,
			)
		}
	}

	switch inviteReferralCode.ReferralType {
	case ReferralTypeHours:
		if time.Now().After(inviteReferralCode.CreatedAt.Add(time.Duration(inviteReferralCode.Value) * time.Hour)) {
			// expired
			return inviteReferralCode, false, fmt.Errorf("referral code %q is expired", referralCode)
		}
	case ReferralTypeQuantity:
		// check the total number of times this has been used
		count, err := GetTotalReferralCodeRedemptions(q, inviteReferralCode.ReferralCode)
		if err != nil {
			return nil, false, errors.Wrap(err, "unable to get total redemptions for code")
		}
		if count >= inviteReferralCode.Value {
			// reached capacity
			return inviteReferralCode, false, fmt.Errorf("referral code %q is at capacity", referralCode)
		}
	}

	return inviteReferralCode, true, nil
}

func GenerateAndWriteWaitlistCode(q Q, email string) error {
	vcPrefix := vcgen.New(
		&vcgen.Generator{
			Count: 1, Pattern: "######", Prefix: "WAIT-",
		},
	)
	res, err := vcPrefix.Run()
	if err != nil {
		return fmt.Errorf("unable to generate ref code")
	}
	if res == nil {
		return fmt.Errorf("no ref codes returned from library")
	}
	generatedRefCodes := *res
	if len(generatedRefCodes) != 1 {
		return fmt.Errorf("invalid number of generated ref codes")
	}
	ownerWaitlistCode := generatedRefCodes[0]

	if err = UpdateWaitlistCode(
		q, email, ownerWaitlistCode,
	); err != nil {
		return errors.Wrap(err, "updating waitlist item")
	}
	return nil
}

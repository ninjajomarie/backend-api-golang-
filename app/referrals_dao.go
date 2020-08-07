package external

import "time"

type ReferralType string

const ReferralTypeHours ReferralType = "HOURS"
const ReferralTypeQuantity ReferralType = "QUANTITY"

type ReferralCode struct {
	ID           int          `json:"id,omitempty"`
	UserID       int          `json:"user_id,omitempty"`
	ReferralCode string       `json:"referral_code,omitempty"`
	ReferralType ReferralType `json:"referral_type,omitempty"`
	Value        int          `json:"value,omitempty"`
	IsActive     bool         `json:"is_active,omitempty"`
	CreatedAt    *time.Time   `json:"created_at,omitempty"`
	UpdatedAt    *time.Time   `json:"updated_at,omitempty"`
}

type ReferralRedemption struct {
	ID             int        `json:"id,omitempty"`
	UserID         int        `json:"user_id,omitempty"`
	ReferralCodeID int        `json:"referral_code_id,omitempty"`
	ReferralUserID bool       `json:"referral_user_id,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type Waitlist struct {
	ID                   int        `json:"id,omitempty"`
	EmailAddress         string     `json:"email_address,omitempty"`
	OwnerWaitlistCode    int        `json:"owner_waitlist_code,omitempty"`
	ReferralCodeID       bool       `json:"referral_code_id,omitempty"`
	OriginalReferralCode bool       `json:"original_referral_code,omitempty"`
	OriginalWaitlistCode bool       `json:"original_waitlist_code,omitempty"`
	CreatedAt            *time.Time `json:"created_at,omitempty"`
	UpdatedAt            *time.Time `json:"updated_at,omitempty"`
}

func GetReferralCodeByUserID(q Q, userID int) (*ReferralCode, error) {
	var i ReferralCode
	if err := q.Get(
		&i,
		`
			SELECT
				id,
				user_id,
				referral_code,
				referral_type,
				value,
				is_active,
				created_at,
				updated_at
			FROM ggwp.referral_codes
			WHERE user_id = $1
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return &i, nil
}

func GetReferralCodeByCode(q Q, code string) (*ReferralCode, error) {
	var i ReferralCode
	if err := q.Get(
		&i,
		`
			SELECT
				id,
				user_id,
				referral_code,
				referral_type,
				value,
				is_active,
				created_at,
				updated_at
			FROM ggwp.referral_codes
			WHERE referral_code = $1
		`,
		code,
	); err != nil {
		return nil, err
	}

	return &i, nil
}

func CreateReferralCode(q Q, userID int, refCode string) error {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.referral_codes
			(
				user_id, referral_code, referral_type, value, is_active
			)
			VALUES
			(
				$1, $2, 'QUANTITY', 2, TRUE
			)
		`,
		userID,
		refCode,
	); err != nil {
		return err
	}

	return nil
}

func CreateReferralRedemption(q Q, userID, code int) error {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.referral_redemptions
			(
				referral_code_id, referral_user_id
			)
			VALUES
			(
				$1, $2
			)
		`,
		code,
		userID,
	); err != nil {
		return err
	}

	return nil
}

func GetUsersIDsMissingReferralCodes(q Q) ([]int, error) {
	var uIDs []int
	if err := q.Select(
		&uIDs,
		`
      SELECT
        u.id
      FROM
        ggwp.users u
      LEFT JOIN
        ggwp.referral_codes r
        ON r.user_id = u.id
      WHERE r.id IS NULL
		`,
	); err != nil {
		return nil, err
	}

	return uIDs, nil
}

func GetTotalReferralCodeRedemptions(q Q, code string) (int, error) {
	var c int
	if err := q.Get(
		&c,
		`
			SELECT
        count(r.*)
			FROM
        ggwp.referral_redemptions r
      JOIN
        ggwp.referral_codes c
        ON c.id = r.referral_code_id
			WHERE
        c.referral_code = $1
		`,
		code,
	); err != nil {
		return 0, err
	}

	return c, nil
}

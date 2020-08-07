package external

import (
	"strings"
)

func CreateWaitlistItem(
	q Q,
	emailAddress,
	ownerWaitlistCode string,
	originalReferralCodeID,
	originalWaitlistCodeID *int,
) (*WaitlistItem, error) {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.waitlist
			(
				email_address, owner_waitlist_code, original_referral_code_id, original_waitlist_code_id
			)
			VALUES
			(
				$1, $2, $3, $4
			)
		`,
		emailAddress,
		ownerWaitlistCode,
		originalReferralCodeID,
		originalWaitlistCodeID,
	); err != nil {
		return nil, err
	}

	var rI WaitlistItem
	if err := q.Get(
		&rI,
		`
			SELECT
				id,
				email_address,
				owner_waitlist_code,
				original_referral_code_id,
				original_waitlist_code_id,
				created_at,
				updated_at
			FROM
				ggwp.waitlist
			WHERE
				email_address = $1
		`,
		strings.ToLower(emailAddress),
	); err != nil {
		return nil, err
	}

	return &rI, nil
}

func UpdateWaitlistCode(
	q Q,
	emailAddress, ownerWaitlistCode string,
) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.waitlist
			SET owner_waitlist_code = $2
			WHERE email_address = $1
		`,
		emailAddress,
		ownerWaitlistCode,
	); err != nil {
		return err
	}

	return nil
}

func GetWaitlistItemByCode(
	q Q,
	code string,
) (*WaitlistItem, error) {
	var i WaitlistItem
	if err := q.Get(
		&i,
		`
			SELECT
				id,
				email_address,
				owner_waitlist_code,
				original_referral_code_id,
				original_waitlist_code_id,
				created_at,
				updated_at
			FROM
				ggwp.waitlist
			WHERE
				owner_waitlist_code = $1
		`,
		code,
	); err != nil {
		return nil, err
	}

	return &i, nil
}

func GetWaitlist(q Q) ([]WaitlistItem, error) {
	var items []WaitlistItem
	if err := q.Select(
		&items,
		`
			SELECT
				id,
				-- email_address, intentionally not returning email_address
				owner_waitlist_code,
				original_referral_code_id,
				original_waitlist_code_id,
				created_at,
				updated_at
			FROM
				ggwp.waitlist
			WHERE
				owner_waitlist_code IS NOT NULL
			ORDER BY
				created_at
		`,
	); err != nil {
		return nil, err
	}

	return items, nil
}

func GetEmailsMissingWaitlistCodes(q Q) ([]string, error) {
	var emails []string
	if err := q.Select(
		&emails,
		`
      SELECT
        email_address
      FROM
        ggwp.waitlist
      WHERE
				owner_waitlist_code IS NULL
		`,
	); err != nil {
		return nil, err
	}

	return emails, nil
}

func GetWaitlistItemsWithoutAQueuedEmail(q Q) ([]WaitlistItem, error) {
	var i []WaitlistItem
	if err := q.Select(
		&i,
		`
			SELECT
				w.id,
				w.email_address,
				w.owner_waitlist_code,
				w.original_referral_code_id,
				w.original_waitlist_code_id,
				w.created_at,
				w.updated_at
			FROM
				ggwp.waitlist w
			LEFT JOIN
				ggwp.emails e
				ON e.email_address = w.email_address
				AND e.type = $1
			WHERE
				e.email_address IS NULL
			ORDER BY
				w.created_at
		`,
		EmailType_Waitlist,
	); err != nil {
		return nil, err
	}

	return i, nil
}

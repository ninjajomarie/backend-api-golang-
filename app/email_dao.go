package external

func CreateEmail(
	q Q,
	userID *int,
	emailAddress,
	templateName string,
	templateVars HStoreMap,
	t EmailType,
	status EmailStatus,
) error {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.emails
			(
				user_id, email_address, template_name, template_vars, type, status
			)
			VALUES
			(
				$1, $2, $3, $4, $5, $6
			)
		`,
		userID,
		emailAddress,
		templateName,
		templateVars,
		t,
		status,
	); err != nil {
		return err
	}

	return nil
}

func GetPendingEmails(
	q Q,
) ([]*Email, error) {
	var e []*Email
	if err := q.Select(
		&e,
		`
			SELECT
				id,
				user_id,
				email_address,
				template_name,
				template_vars,
				type,
				status,
				created_at,
				updated_at,
				sent_at
			FROM
				ggwp.emails
			WHERE
				status = $1
		`,
		EmailStatus_Pending,
	); err != nil {
		return nil, err
	}

	return e, nil
}

func MarkEmailAsSent(
	q Q,
	id int,
) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.emails
			SET status = $1, sent_at = NOW()
			WHERE id = $2
		`,
		EmailStatus_Sent,
		id,
	); err != nil {
		return err
	}

	return nil
}

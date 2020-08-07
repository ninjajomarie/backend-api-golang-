package external

import (
	"fmt"
)

func selectFromUsersWhere(w string) string {
	return fmt.Sprintf(
		`
			SELECT
				u.id,
				COALESCE(u.about, '') about,
				u.date_of_birth date_of_birth,
				u.email,
				COALESCE(u.phone,0) phone,
				COALESCE(u.location, '') as location,
				COALESCE(u.sports, '') sports,
				COALESCE(u.user_type, '') user_type,
				COALESCE(u.hashtags, '') hashtags,
				u.is_active,
				u.last_online last_online,
				u.user_admin_level,
				u.is_verified,
				u.password_hash,
				u.created_at,
				u.updated_at,

				-- player details
				p.id as "player.id",
				p.user_id as "player.user_id",
				COALESCE(p.first_name, '') "player.first_name",
				COALESCE(p.last_name, '') "player.last_name",
				COALESCE(p.gender, '') "player.gender",
				p.created_at as "player.created_at",
				p.updated_at as "player.updated_at"

			FROM ggwp.users u
			LEFT JOIN ggwp.players p
				ON p.user_id = u.id
				%s
		`,
		w,
	)
}

func GetProfileImageByUserID(q Q, userID int) (*ProfileImage, error) {
	var i ProfileImage
	if err := q.Get(
		&i,
		`
			SELECT
				-- profile image
				pi.id as "id",
				pi.user_id as "user_id",
				pi.file_id as "file_id",
				pi.is_active as "is_active",
				pi.created_at as "created_at",
				pi.updated_at as "updated_at",

				-- file details
				f.id as "file.id",
				f.user_id as "file.user_id",
				f.name as "file.name",
				f.description as "file.description",
				f.extension as "file.extension",
				f.type as "file.type",
				f.size as "file.size",
				f.is_active as "file.is_active",
				f.created_at as "file.created_at",
				f.updated_at as "file.updated_at"
			FROM ggwp.profile_images pi
			LEFT JOIN ggwp.files	f
				ON f.id = pi.file_id
			WHERE pi.user_id = $1
				AND pi.is_active
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return &i, nil
}

func GetLearningProgressesByUserIDAndDeviceUniqueID(
	q Q,
	userID int,
	deviceUniqueID string,
) ([]*LearningProgress, error) {
	var l []*LearningProgress
	if err := q.Select(
		&l,
		`
			SELECT
				id,
				module_id,
				user_id,
				module_file_ranking,
				COALESCE(seek, 0) seek,
				device_unique_id,
				created_at,
				updated_at
			FROM ggwp.learning_progresses
			WHERE user_id = $1
				-- only include device_unique_id info if provided device_unique_id is not empty
				OR (
					NULLIF(TRUE, $2 = '')
					AND
					device_unique_id = $2
				)
			ORDER BY created_at
		`,
		userID,
		deviceUniqueID,
	); err != nil {
		return nil, err
	}

	return l, nil
}

func GetLearningProgressesByDeviceUniqueID(q Q, uniqueID string) ([]*LearningProgress, error) {
	var l []*LearningProgress
	if err := q.Select(
		&l,
		`
			SELECT
				id,
				module_id,
				user_id,
				module_file_ranking,
				device_unique_id,
				COALESCE(seek, 0) seek,
				created_at,
				updated_at
			FROM ggwp.learning_progresses
			WHERE device_unique_id = $1
			ORDER BY created_at
		`,
		uniqueID,
	); err != nil {
		return nil, err
	}

	return l, nil
}

func GetQuizGradingsByUserID(q Q, userID int) ([]*QuizGrading, error) {
	var g []*QuizGrading
	if err := q.Select(
		&g,
		`
			SELECT
				*
			FROM ggwp.quiz_gradings
			WHERE user_id = $1
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return g, nil
}

func GetGoalsByUserID(q Q, userID int) ([]*UserGoal, error) {
	var g []*UserGoal
	if err := q.Select(
		&g,
		`
			SELECT
				id,
				user_id,
				description,
				value,
				rate,
				deadline,
				completed_at,
				is_active,
				created_at,
				updated_at
			FROM ggwp.user_goals
			WHERE user_id = $1
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return g, nil
}

func CreateGoal(q Q, g *UserGoal) error {
	if _, err := q.NamedExec(
		`
			INSERT INTO ggwp.user_goals
			(
				user_id, description, value, rate, deadline
			)
			VALUES
			(
				:user_id, :description, :value, :rate, :deadline
			)
		`,
		g,
	); err != nil {
		return err
	}

	return nil
}

func CompleteGoal(q Q, goalID int) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.user_goals
			SET completed_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`,
		goalID,
	); err != nil {
		return err
	}

	return nil
}

func IncompleteGoal(q Q, goalID int) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.user_goals
			SET completed_at = NULL, updated_at = NOW()
			WHERE id = $1
		`,
		goalID,
	); err != nil {
		return err
	}

	return nil
}

func GetUserByEmail(q Q, email string) (*User, error) {
	var user User
	if err := q.Get(
		&user,
		selectFromUsersWhere(
			`
				WHERE email = lower($1)
			`,
		),
		email,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByID(q Q, ID int) (*User, error) {
	var user User
	if err := q.Get(
		&user,
		selectFromUsersWhere(
			`
				WHERE u.id = $1
			`,
		),
		ID,
	); err != nil {
		return nil, err
	}

	return &user, nil
}

func CreatePlayer(q Q, newUser *NewUser) (*User, error) {
	if _, err := q.NamedExec(
		`
			INSERT INTO ggwp.users
			(
				email, password_hash, user_type, last_online
			)
			VALUES
			(
				lower(:email), :password, 'player', NOW()
			)
		`,
		newUser,
	); err != nil {
		return nil, err
	}

	var createdUserID int
	if err := q.Get(
		&createdUserID,
		`
			SELECT id
			FROM ggwp.users
			WHERE email = lower($1)
		`,
		newUser.Email,
	); err != nil {
		return nil, err
	}

	if _, err := q.Exec(
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
		createdUserID,
		newUser.FirstName,
		newUser.LastName,
	); err != nil {
		return nil, err
	}

	finalCreatedUser, err := GetUserByEmail(q, newUser.Email)
	if err != nil {
		return nil, err
	}

	return finalCreatedUser, nil
}

func UpdateUserPassword(q Q, userID int, password string) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.users
			SET password_hash = $2, updated_at = NOW()
			WHERE id = $1
		`,
		userID,
		password,
	); err != nil {
		return err
	}

	return nil
}

func UpdateUser(q Q, u *User) error {
	if _, err := q.NamedExec(
		`
			UPDATE ggwp.users new
			SET
				email = COALESCE(NULLIF(lower(:email), ''), existing.email),
				about = COALESCE(NULLIF(:about, ''), existing.about),
				date_of_birth = COALESCE(CAST(NULLIF(:date_of_birth, '') AS DATE), existing.date_of_birth),
				phone = COALESCE(NULLIF(:phone, 0), existing.phone),
				location = COALESCE(NULLIF(:location, ''), existing.location),
				sports = COALESCE(NULLIF(:sports, ''), existing.sports),
				hashtags = COALESCE(NULLIF(:hashtags, ''), existing.hashtags),
				updated_at = NOW()
			FROM (
				SELECT * FROM ggwp.users WHERE id = :id
			) existing
			WHERE new.id = :id
		`,
		u,
	); err != nil {
		return err
	}

	return nil
}

func UpdatePlayer(q Q, p *Player) error {
	if _, err := q.NamedExec(
		`
			UPDATE
				ggwp.players new
			SET
				first_name = COALESCE(NULLIF(:first_name, ''), existing.first_name),
				last_name = COALESCE(NULLIF(:last_name, ''), existing.last_name),
				gender = COALESCE(NULLIF(:gender, ''), existing.gender),
				updated_at = NOW()
			FROM (
				SELECT * FROM ggwp.players WHERE user_id = :user_id
			) existing
			WHERE new.user_id = :user_id
		`,
		p,
	); err != nil {
		return err
	}

	return nil
}

func CreateFile(q Q, f *File) (*File, error) {
	if _, err := q.NamedExec(
		`
			INSERT INTO ggwp.files
			(
				user_id, name, description, extension, type, size, created_at, updated_at
			)
			VALUES
			(
				:user_id, :name, :description, :extension, :type, :size, NOW(), NOW()
			)
		`,
		f,
	); err != nil {
		return nil, err
	}

	var iF File
	if err := q.Get(
		&iF,
		`
			SELECT
				id,
				user_id,
				name,
				description,
				extension,
				type,
				size,
				is_active,
				created_at,
				updated_at
			FROM ggwp.files
			WHERE user_id = $1
			ORDER BY created_at DESC
			LIMIT 1
		`,
		f.UserID,
	); err != nil {
		return nil, err
	}

	return &iF, nil
}

func AddProfileImage(q Q, p *ProfileImage) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.profile_images
			SET is_active = false, updated_at = NOW()
			WHERE user_id = $1
		`,
		p.UserID,
	); err != nil {
		return err
	}

	if _, err := q.NamedExec(
		`
			INSERT INTO ggwp.profile_images
			(
				file_id, user_id, created_at, updated_at
			)
			VALUES
			(
				:file_id, :user_id, NOW(), NOW()
			)
		`,
		p,
	); err != nil {
		return err
	}

	return nil
}

func GetAllGoalTemplates(q Q) ([]*GoalTemplate, error) {
	var t []*GoalTemplate
	if err := q.Select(
		&t,
		`
			SELECT
				id,
				name,
				quantitative,
				created_at,
				updated_at
			FROM ggwp.user_goal_templates
		`,
	); err != nil {
		return nil, err
	}

	return t, nil
}

func CreatePasswordReset(q Q, userID int, token string) error {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.user_password_reset
			(
				user_id, token
			)
			VALUES
			(
				$1, $2
			)
		`,
		userID,
		token,
	); err != nil {
		return err
	}

	return nil
}

func GetPasswordReset(q Q, userID int, token string) (*PasswordReset, error) {
	var pR PasswordReset
	if err := q.Get(
		&pR,
		`
			SELECT
				id,
				user_id,
				token,
				created_at
			FROM
				ggwp.user_password_reset
			WHERE
				user_id = $1
				AND token = $2
		`,
		userID,
		token,
	); err != nil {
		return nil, err
	}

	return &pR, nil
}

func GetAllPasswordReset(q Q, userID int) ([]*PasswordReset, error) {
	var pR []*PasswordReset
	if err := q.Select(
		&pR,
		`
			SELECT
				id,
				user_id,
				token,
				created_at
			FROM
				ggwp.user_password_reset
			WHERE
				user_id = $1
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return pR, nil
}

func UpdateLastOnline(q Q, userID int) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.users
			SET last_online = NOW()
			WHERE id = $1
		`,
		userID,
	); err != nil {
		return err
	}

	return nil
}

func AddUserIDToWaitlistIfApplicable(q Q, userID int, emailAddress string) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.waitlist
			SET user_id = $1
			WHERE email_address = $2
		`,
		userID,
		emailAddress,
	); err != nil {
		return err
	}

	return nil
}

func InsertSocialToken(
	q Q,
	userID int,
	aT, aS string,
	sN SocialNetwork,
) error {
	if _, err := q.Exec(
		`
			INSERT INTO ggwp.social
			(
				user_id, access_token, access_secret, social_network, is_active
			)
			VALUES
			(
				$1, $2, $3, $4, TRUE
			)
		`,
		userID,
		aT,
		aS,
		sN,
	); err != nil {
		return err
	}

	return nil
}

func UpdateSocialToken(q Q, uid int, aT, aS string, sN SocialNetwork) error {
	if _, err := q.Exec(
		`
			UPDATE ggwp.social
			SET access_token = $1, access_secret = $2, updated_at = NOW()
			WHERE user_id = $3 AND social_network = $4
		`,
		aT,
		aS,
		uid,
		sN,
	); err != nil {
		return err
	}

	return nil
}

package external

import (
	"fmt"

	"github.com/lib/pq"
)

func selectFromModulesWhere(where string) string {
	return fmt.Sprintf(
		`
			SELECT
				id,
				user_id,
				name,
				description,
				ranking,
				hashtags,
				category_id,
				free,
				is_active,
				created_at,
				updated_at
			FROM
				ggwp.modules
			%s
	`,
		where,
	)
}

func GetAllModules(q Q) ([]*Module, error) {
	var m []*Module
	if err := q.Select(
		&m,
		selectFromModulesWhere(`
			WHERE is_active
		`),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetModulesByIDs(q Q, ids []int) ([]*Module, error) {
	var m []*Module
	if err := q.Select(
		&m,
		selectFromModulesWhere(
			`
				WHERE id = ANY($1)
			`,
		),
		pq.Array(ids),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetModuleByRanking(q Q, r int) ([]*Module, error) {
	var m []*Module
	if err := q.Select(
		&m,
		selectFromModulesWhere(
			`
				WHERE ranking = $1
			`,
		),
		r,
	); err != nil {
		return nil, err
	}

	return m, nil
}

func SearchModules(q Q, query string) ([]*Module, error) {

	searchTerm := "%" + query + "%"
	var m []*Module
	if err := q.Select(
		&m,
		selectFromModulesWhere(
			`
				WHERE
					(
						name ilike $1
						OR description ilike $1
					)
					AND is_active
			`,
		),
		searchTerm,
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetQuizzesByModuleID(q Q, ID int) ([]*Quiz, error) {
	var qu []*Quiz
	if err := q.Select(
		&qu,
		`
			SELECT
				*
			FROM
				ggwp.quizzes
			WHERE
				moduleID = $1
		`,
		ID,
	); err != nil {
		return nil, err
	}

	return qu, nil
}

func GetQuizByID(q Q, ID int) (*Quiz, error) {
	var qu Quiz
	if err := q.Get(
		&qu,
		`
			SELECT
				*
			FROM
				ggwp.quizzes
			WHERE
				id = $1
		`,
		ID,
	); err != nil {
		return nil, err
	}

	return &qu, nil
}

func GetAllModulePariticipants(q Q) (map[int]int, error) {
	pL := []struct {
		ModuleID     int `json:"module_id,omitempty"`
		Participants int `json:"participants,omitempty"`
	}{}
	if err := q.Select(
		&pL,
		`
			SELECT
				modules.id module_id,
				COUNT(DISTINCT(learning_progresses.user_id)) participants
			FROM
				ggwp.modules
			LEFT OUTER JOIN
				ggwp.learning_progresses
				ON
					modules.id = learning_progresses.module_id
			WHERE
				modules.is_active = true
			GROUP BY
				modules.id
		`,
	); err != nil {
		return nil, err
	}

	p := map[int]int{}
	for _, r := range pL {
		p[r.ModuleID] = r.Participants
	}

	return p, nil
}

func GetFirstFileOfModuleByModuleRanking(q Q, ranking int) (*ModuleFile, error) {
	m := &ModuleFile{}
	if err := q.Get(
		m,
		`
			SELECT
				mF.id,
				mF.module_id,
				mF.file_id,
				mF.ranking,
				mF.created_at,
				mF.updated_at
			FROM
				ggwp.modules m
			JOIN
				ggwp.module_files mF
				ON mF.module_id = m.id
			WHERE
				m.ranking = $1
			ORDER BY mF.id
			LIMIT 1
		`,
		ranking,
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetModulesFilesByModuleIDs(q Q, IDs []int) ([]*ModuleFile, error) {
	var m []*ModuleFile
	if err := q.Select(
		&m,
		fmt.Sprintf(
			`
				SELECT
					mF.id,
					mF.module_id,
					mF.file_id,
					mF.ranking,
					mF.created_at,
					mF.updated_at,

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
					f.updated_at as "file.updated_at",

					-- video details
					v.id as "file.video_details.id",
					v.file_id as "file.video_details.file_id",
					v.duration as "file.video_details.duration",
					v.created_at "file.video_details.created_at",
					v.updated_at "file.video_details.updated_at"
				FROM
					ggwp.module_files mF
				JOIN
					ggwp.files	f
					ON f.id = mF.file_id
				LEFT JOIN
					ggwp.video_details v
					ON v.file_id = f.id
				WHERE
					module_id IN (%s)
			`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetQuizzesByModuleIDs(q Q, IDs []int) ([]*Quiz, error) {
	var quizzes []*Quiz
	if err := q.Select(
		&quizzes,
		fmt.Sprintf(
			`
			SELECT
				id,
				module_id,
				name,
				description,
				passing_grade,
				is_active,
				created_at,
				updated_at
			FROM
				ggwp.quizzes
			WHERE
				module_id IN (%s)
		`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return quizzes, nil
}

func GetModulesLearningOutcomesByModuleIDs(q Q, IDs []int) ([]*ModuleLearningOutcome, error) {
	var m []*ModuleLearningOutcome
	if err := q.Select(
		&m,
		fmt.Sprintf(
			`
				SELECT
					id,
					module_id,
					description,
					ranking,
					is_active,
					created_at,
					updated_at
				FROM
					ggwp.module_learning_outcomes
				WHERE
					module_id IN (%s)
			`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetModulesSupportingMaterialByModuleIDs(q Q, IDs []int) ([]*ModuleSupportingMaterial, error) {
	var m []*ModuleSupportingMaterial
	if err := q.Select(
		&m,
		fmt.Sprintf(
			`
				SELECT
					id,
					module_id,
					name,
					description,
					url,
					is_active,
					created_at,
					updated_at
				FROM
					ggwp.module_supporting_material
				WHERE
					module_id IN (%s)
				AND
					is_active
			`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetModulesCategoryByIDs(q Q, IDs []int) ([]*ModuleCategory, error) {
	var m []*ModuleCategory
	if err := q.Select(
		&m,
		fmt.Sprintf(
			`
				SELECT
					id,
					name,
					description,
					created_at,
					updated_at
				FROM
					ggwp.module_categories
				WHERE
					id IN (%s)
			`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func intsToString(s []int) string {
	l := len(s)
	if l == 0 {
		return "0"
	}

	res := ""
	for counter, i := range s {
		if counter == l-1 {
			res = fmt.Sprintf(res+"%d", i)
		} else {
			res = fmt.Sprintf(res+"%d,", i)
		}
	}
	return res
}

func RecordModuleProgress(q Q, p *LearningProgress) error {
	if _, err := q.NamedExec(
		`
			INSERT INTO ggwp.learning_progresses
			(
				module_id, user_id, module_file_ranking, seek, created_at, updated_at, device_unique_id
			)
			VALUES
			(
				:module_id, :user_id, :module_file_ranking, :seek, NOW(), NOW(), :device_unique_id
			)
		`,
		p,
	); err != nil {
		return err
	}

	return nil
}

func GetQuizTakesByUserIDModuleIDAndQuizID(q Q, userID, moduleID, quizID int) (int, error) {
	var t int
	if err := q.Get(
		&t,
		`
			SELECT
				take_number
			FROM
				ggwp.quiz_gradings
			WHERE
				user_id = $1
				AND quiz_id = $2
				AND module_id = $3
			ORDER BY
				take_number DESC
			LIMIT 1
		`,
		userID,
		moduleID,
		quizID,
	); err != nil {
		return 0, err
	}

	return t, nil
}

func GetQuizQuestionsByQuizIDs(q Q, quizIDs []int) ([]*Question, error) {
	var questions []*Question
	if err := q.Select(
		&questions,
		fmt.Sprintf(
			`
				SELECT
					id,
					quiz_id,
					name,
					description,
					ranking,
					answer_option_ranking,
					created_at,
					updated_at
				FROM
					ggwp.quiz_questions
				WHERE
					quiz_id IN (%s)
			`,
			intsToString(quizIDs),
		),
	); err != nil {
		return nil, err
	}
	return questions, nil
}

func GetQuizQuestionOptionsByQuestionIDs(q Q, questionIDs []int) ([]*QuestionOption, error) {
	var options []*QuestionOption
	if err := q.Select(
		&options,
		fmt.Sprintf(
			`
				SELECT
					id,
					quiz_question_id,
					name,
					description,
					ranking,
					created_at,
					updated_at
				FROM
					ggwp.quiz_question_options
				WHERE
					quiz_question_id IN (%s)
			`,
			intsToString(questionIDs),
		),
	); err != nil {
		return nil, err
	}

	return options, nil
}

func InsertQuizGradings(q Q, gS []*QuizGrading) error {
	for _, g := range gS {
		if _, err := q.NamedExec(
			`
			INSERT INTO ggwp.quiz_gradings
			(
				module_id, quiz_id, question_id, user_id, user_answer_ranking,
				correct, take_number, created_at, updated_at
			)
			VALUES
			(
				:module_id, :quiz_id, :question_id, :user_id, :user_answer_ranking,
				:correct, :take_number, :created_at, :updated_at
			)
		`,
			g,
		); err != nil {
			return err
		}

	}
	return nil
}

func GetModulesBannerByModuleIDs(q Q, IDs []int) ([]*ModuleBanner, error) {
	var m []*ModuleBanner
	if err := q.Select(
		&m,
		fmt.Sprintf(
			`
				SELECT
					b.id,
					b.module_id,
					b.file_id,
					b.created_at,
					b.updated_at,
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
				FROM
					ggwp.module_banners b
				LEFT JOIN
					ggwp.files f
					ON f.id = b.file_id
				WHERE
					module_id IN (%s)
			`,
			intsToString(IDs),
		),
	); err != nil {
		return nil, err
	}

	return m, nil
}

func GetCompletedModuleIDsByUserID(q Q, userID int) ([]int, error) {
	var c []int
	if err := q.Select(
		&c,
		`
			SELECT
				DISTINCT module_id
			FROM
				ggwp.quiz_gradings
			WHERE
				user_id = $1
		`,
		userID,
	); err != nil {
		return nil, err
	}

	return c, nil
}

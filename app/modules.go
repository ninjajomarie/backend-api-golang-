package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

type GradingRequest struct {
	UserID   int       `json:"user_id,omitempty"`
	ModuleID int       `json:"module_id,omitempty"`
	QuizID   int       `json:"quiz_id,omitempty"`
	Answers  []*Answer `json:"answers,omitempty"`
}

func (e *External) HandleGetAllModules(w http.ResponseWriter, r *http.Request) {
	m, err := GetAllModules(e.dao.ReadDB)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting all modules"))
		return
	}

	mWD, err := e.injectModuleDetails(m)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "adding module details"))
		return
	}
	e.returnJSON(
		w,
		mWD,
	)
}

func (e *External) HandleSearchModules(w http.ResponseWriter, r *http.Request) {
	m, err := SearchModules(e.dao.ReadDB, mux.Vars(r)["query"])
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "searching modules"))
		return
	}

	mWD, err := e.injectModuleDetails(m)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "adding module details"))
	}
	e.returnJSON(
		w,
		mWD,
	)
}

func (e *External) injectModuleDetails(
	modules []*Module,
) ([]*Module, error) {
	var categoryIDs []int
	var moduleIDs []int
	for _, module := range modules {
		categoryIDs = append(categoryIDs, module.CategoryID)
		moduleIDs = append(moduleIDs, module.ID)
	}

	moduleBanners, err := GetModulesBannerByModuleIDs(e.dao.ReadDB, moduleIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module banner")
	}

	moduleCategories, err := GetModulesCategoryByIDs(e.dao.ReadDB, categoryIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module category")
	}

	moduleQuizzes, err := GetQuizzesByModuleIDs(e.dao.ReadDB, moduleIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module quizzes")
	}
	if moduleQuizzes, err = e.injectQuizDetails(moduleQuizzes); err != nil {
		return nil, errors.Wrap(err, "adding module quizzes details")
	}

	moduleFiles, err := GetModulesFilesByModuleIDs(e.dao.ReadDB, moduleIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module files")
	}

	moduleLearningOutcomes, err := GetModulesLearningOutcomesByModuleIDs(e.dao.ReadDB, moduleIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module learning outcomes")
	}

	moduleSupportingMaterial, err := GetModulesSupportingMaterialByModuleIDs(e.dao.ReadDB, moduleIDs)
	if err != nil {
		return nil, errors.Wrap(err, "getting module supporting material")
	}

	// map of module id to module
	moduleIDToModule := map[int]*Module{}
	for _, m := range modules {
		// include categories
		for _, mC := range moduleCategories {
			if m.CategoryID == mC.ID {
				m.Category = mC
			}
		}
		moduleIDToModule[m.ID] = m
	}

	// include banners
	for _, mB := range moduleBanners {
		moduleIDToModule[mB.ModuleID].Banner = mB
	}

	// include quizzes
	for _, mQ := range moduleQuizzes {
		moduleIDToModule[mQ.ModuleID].Quizzes = append(
			moduleIDToModule[mQ.ModuleID].Quizzes,
			mQ,
		)
	}

	// include files
	for _, mF := range moduleFiles {
		moduleIDToModule[mF.ModuleID].Files = append(
			moduleIDToModule[mF.ModuleID].Files,
			mF,
		)
	}

	// include learning outcomes
	for _, mL := range moduleLearningOutcomes {
		moduleIDToModule[mL.ModuleID].LearningOutcomes = append(
			moduleIDToModule[mL.ModuleID].LearningOutcomes,
			mL,
		)
	}

	// include supporting material
	for _, s := range moduleSupportingMaterial {
		moduleIDToModule[s.ModuleID].SupportingMaterial = append(
			moduleIDToModule[s.ModuleID].SupportingMaterial,
			s,
		)
	}

	var finalModules []*Module
	for _, m := range modules {
		finalModules = append(finalModules, m)
	}

	return finalModules, nil
}

func (e *External) injectQuizDetails(
	quizzes []*Quiz,
) ([]*Quiz, error) {
	// map of quiz id to quiz
	quizIDToQuiz := map[int]*Quiz{}
	quizIDs := []int{}
	for _, q := range quizzes {
		quizIDToQuiz[q.ID] = q
		quizIDs = append(quizIDs, q.ID)
	}

	questions, err := GetQuizQuestionsByQuizIDs(e.dao.ReadDB, quizIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "getting quiz questions by quiz IDs: %v", quizIDs)
	}

	// map of question id to question
	quizQuestionIDs := []int{}
	questionIDToQuestion := map[int]*Question{}
	for _, q := range questions {
		questionIDToQuestion[q.ID] = q
		quizQuestionIDs = append(quizQuestionIDs, q.ID)
	}

	options, err := GetQuizQuestionOptionsByQuestionIDs(e.dao.ReadDB, quizQuestionIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "getting quiz question options by question ID: %v", quizQuestionIDs)
	}

	// include question options
	for _, o := range options {
		questionIDToQuestion[o.QuizQuestionID].Options = append(
			questionIDToQuestion[o.QuizQuestionID].Options,
			o,
		)
	}

	// include questions
	for _, q := range questionIDToQuestion {
		quizIDToQuiz[q.QuizID].Questions = append(
			quizIDToQuiz[q.QuizID].Questions,
			q,
		)
	}

	var finalQuizzes []*Quiz
	for _, q := range quizzes {
		finalQuizzes = append(finalQuizzes, q)
	}

	return finalQuizzes, nil
}

func (e *External) HandleGetAllModuleParticipants(w http.ResponseWriter, r *http.Request) {
	m, err := GetAllModulePariticipants(e.dao.ReadDB)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, m)
}

func (e *External) HandleRecordModuleProgress(w http.ResponseWriter, r *http.Request) {
	p := &LearningProgress{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	p.UserID = sql.NullInt64{
		Int64: int64(r.Context().Value("user_id").(int)),
		Valid: true,
	}
	p.DeviceUniqueID = sql.NullString{
		String: r.Context().Value("device_unique_id").(string),
		Valid:  true,
	}

	if err := RecordModuleProgress(e.dao.DB, p); err != nil {
		// e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "recording module progress"))
		// return
		// just log the error
		e.log.WithError(err).Error("recording module progress")
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleGradeQuiz(w http.ResponseWriter, r *http.Request) {
	gR := &GradingRequest{}
	if err := json.NewDecoder(r.Body).Decode(gR); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	gR.UserID = r.Context().Value("user_id").(int)

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
	}
	defer tx.Rollback()

	quiz, err := GetQuizByID(tx, gR.QuizID)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrapf(err, "getting quiz by id %d", gR.QuizID))
		return
	}
	quizzes, err := e.injectQuizDetails([]*Quiz{quiz})
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "adding module quiz details"))
		return
	}
	if len(quizzes) != 1 {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "wrong total number of quizzes obtaines after injecting quiz details"))
		return
	}
	quiz = quizzes[0]

	quizTakes, err := GetQuizTakesByUserIDModuleIDAndQuizID(tx, gR.UserID, gR.ModuleID, gR.QuizID)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting quiz takes"))
		return
	} else if err == sql.ErrNoRows {
		quizTakes = 0
	}
	takeNumber := quizTakes + 1

	questionIDToQuestion := map[int]*Question{}
	for _, qu := range quiz.Questions {
		questionIDToQuestion[qu.ID] = qu
	}

	questionIDToGrading := map[int]*QuizGrading{}
	for _, a := range gR.Answers {
		question, ok := questionIDToQuestion[a.QuestionID]
		if !ok {
			e.writeError(
				w, r, http.StatusInternalServerError,
				fmt.Errorf("missing question id: %d in map: %#v", a.QuestionID, questionIDToQuestion),
			)
			return
		}

		var correct bool
		if question.AnswerOptionRanking == 0 {
			// quiz question has no true "correct" answer, mark as correct
			correct = true
		} else {
			correct = question.AnswerOptionRanking == a.AnswerRanking
		}
		questionIDToGrading[a.QuestionID] = &QuizGrading{
			ModuleID:          gR.ModuleID,
			QuizID:            gR.QuizID,
			QuestionID:        a.QuestionID,
			UserID:            gR.UserID,
			UserAnswerRanking: a.AnswerRanking,
			Correct:           correct,
			TakeNumber:        takeNumber,
			CreatedAt: &NullTime{
				NullTime: pq.NullTime{
					Time:  time.Now(),
					Valid: true,
				},
			},
			UpdatedAt: &NullTime{
				NullTime: pq.NullTime{
					Time:  time.Now(),
					Valid: true,
				},
			},
		}
	}

	gradings := []*QuizGrading{}
	for _, g := range questionIDToGrading {
		gradings = append(gradings, g)
	}

	// save grading
	if err := InsertQuizGradings(tx, gradings); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "inserting quiz grading"))
		return
	}

	// bump user to next module
	modules, err := GetModulesByIDs(tx, []int{gR.ModuleID})
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting current module by id"))
		return
	}
	var currentModule *Module
	for _, module := range modules {
		if module.ID == gR.ModuleID {
			currentModule = module
			break
		}
	}
	// there is a next module, bump the user
	nextModuleFile, err := GetFirstFileOfModuleByModuleRanking(e.dao.ReadDB, currentModule.Ranking+1)
	if err != nil && err != sql.ErrNoRows {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting first file of next module by ranking"))
		return
	} else if err == sql.ErrNoRows {
		if err := tx.Commit(); err != nil {
			e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "commiting grading"))
			return
		}
		// no next module, we are done here
		e.returnJSON(w, gradings)
		return
	}

	// there is a next module, bump the user to it by adding a learning progress
	userID := sql.NullInt64{
		Int64: int64(gR.UserID),
		Valid: true,
	}
	deviceUniqueID := sql.NullString{
		String: r.Context().Value("device_unique_id").(string),
		Valid:  true,
	}
	if err := RecordModuleProgress(tx, &LearningProgress{
		UserID:            userID,
		ModuleID:          nextModuleFile.ModuleID,
		ModuleFileRanking: 1,
		DeviceUniqueID:    deviceUniqueID,
	}); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "adding learning progress "))
		return
	}

	// TODO:
	// Get the user grade and insert it into the db
	// Make this the returned value to the frontend as well

	if err := tx.Commit(); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "commiting grading"))
		return
	}
	e.returnJSON(w, gradings)
}

package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (e *External) HandleGetLearningProgress(w http.ResponseWriter, r *http.Request) {
	l, err := GetLearningProgressesByUserIDAndDeviceUniqueID(
		e.dao.ReadDB,
		r.Context().Value("user_id").(int),
		r.Context().Value("device_unique_id").(string),
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, l)
}

func (e *External) HandleGetQuizGradings(w http.ResponseWriter, r *http.Request) {
	l, err := GetQuizGradingsByUserID(e.dao.ReadDB, r.Context().Value("user_id").(int))
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, l)
}

func (e *External) HandleGetSelf(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, err := e.GetUserByID(
		ctx.Value("user_id").(int),
		ctx.Value("device_unique_id").(string),
	)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "self getting user by id"))
		return
	}

	e.returnJSON(w, user)
}

func (e *External) HandleFileUpload(w http.ResponseWriter, r *http.Request) {
	// Parse our multipart form, 10 << 20 specifies a maximum
	// upload of 10 MB files.
	r.ParseMultipartForm(10 << 20)
	// FormFile returns the first file for the given key `myFile`
	// it also returns the FileHeader so we can get the Filename,
	// the Header and the size of the file
	file, handler, err := r.FormFile("items")
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting file from form"))
		return
	}
	defer file.Close()

	contentType := handler.Header.Get("Content-Type")
	contentTypeArr := strings.Split(contentType, "/")
	if handler.Size <= 0 ||
		!(strings.Contains(contentType, "png") || strings.Contains(contentType, "jpeg")) {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "invalid file uplaod"))
		return
	}

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
		return
	}
	defer tx.Rollback()

	// save file details in db
	fileType := contentTypeArr[0]
	fileExtension := contentTypeArr[1]
	createdFile, err := CreateFile(
		tx,
		&File{
			UserID:    NewNullInt64(int64(r.Context().Value("user_id").(int))),
			Extension: NewNullString(fileExtension),
			Name:      NewNullString(strings.Split(handler.Filename, ".")[0]),
			Size:      NewNullInt64(handler.Size),
			Type:      NewNullString(fileType),
		},
	)
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "creating file"),
		)
		return
	}

	if err := AddProfileImage(
		tx,
		&ProfileImage{
			UserID: r.Context().Value("user_id").(int),
			FileID: int(createdFile.ID.Int64),
		},
	); err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "creating file"),
		)
		return
	}

	if err := tx.Commit(); err != nil {
		e.log.WithError(err).Errorf("commiting tx for %v", createdFile.ID)
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "commiting tx for %v"),
		)
		return
	}

	// run in go routine to not block user
	e.log.Info("starting upload routine")
	go func(
		fileHeader *multipart.FileHeader,
		file multipart.File,
		createdFile *File,
	) {
		e.log.Info("inside upload routine, starting session")
		// upload file to s3
		s, err := NewAWS(e.log)
		if err != nil {
			e.writeError(
				w, r, http.StatusInternalServerError, errors.Wrap(err, "creating upload session"),
			)
			return
		}
		e.log.Info("inside upload routine, uploading file to s3")
		_, err = s.UploadFileToS3(handler, file, createdFile)
		if err != nil {
			e.log.WithError(err).Errorf("uploading file to S3: id %v", createdFile.ID)
		}
		e.log.Info("inside upload routine, DONE uploading file to s3")
	}(handler, file, createdFile)

	e.returnJSON(w, nil)
}

func (e *External) HandleUserUpdate(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	if err := json.NewDecoder(r.Body).Decode(user); err != nil {
		e.writeError(w, r, http.StatusBadRequest, errors.Wrap(err, "invalid request"))
		return
	}
	user.ID = r.Context().Value("user_id").(int)
	e.log.WithFields(logrus.Fields{
		"user": user,
		// "struser": string(r.Body),
		"player": user.Player,
	}).Info("user is")

	tx, err := e.dao.GetTx(r.Context())
	if err != nil {
		e.writeError(
			w, r, http.StatusInternalServerError, errors.Wrap(err, "begin tx"),
		)
		return
	}
	defer tx.Rollback()

	if err := UpdateUser(tx, user); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "updating user"))
		return
	}

	if user.Player != nil {
		user.Player.UserID = user.ID
		if err := UpdatePlayer(tx, user.Player); err != nil {
			e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "updating player"))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "committing"))
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) GetUserByID(userID int, deviceUniqueID string) (*User, error) {
	user, err := GetUserByID(e.dao.ReadDB, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting user by id: %d", userID)
	}

	l, err := GetLearningProgressesByUserIDAndDeviceUniqueID(
		e.dao.ReadDB,
		userID,
		deviceUniqueID,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "getting user learning progresses by user id: %d", userID)
	}
	user.LearningProgress = l

	g, err := GetQuizGradingsByUserID(e.dao.ReadDB, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting user quiz gradings by user id: %d", userID)
	}
	user.QuizGradings = g

	c, err := GetCompletedModuleIDsByUserID(e.dao.ReadDB, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting user completed module ids by user id: %d", userID)
	}
	user.CompletedModuleIDs = c

	goals, err := GetGoalsByUserID(e.dao.ReadDB, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "getting user goals by user id: %d", userID)
	}
	user.UserGoals = goals

	profileImage, err := GetProfileImageByUserID(e.dao.ReadDB, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrapf(err, "getting profile image by user id: %d", userID)
	}
	user.ProfileImage = profileImage
	user.PasswordHash = ""

	referralCode, err := GetReferralCodeByUserID(e.dao.ReadDB, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, errors.Wrapf(err, "getting user referral code by user id: %d", userID)
	}
	user.ReferralCode = referralCode

	return user, nil
}

func (e *External) HandleGoalCreate(w http.ResponseWriter, r *http.Request) {
	nG := &UserGoal{}
	// decode the request body into struct and failed if any error occur
	if err := json.NewDecoder(r.Body).Decode(nG); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	nG.UserID = r.Context().Value("user_id").(int)

	if err := CreateGoal(e.dao.DB, nG); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "creating goal"))
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleGoalComplete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	goalID, err := strconv.Atoi(params["id"])
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "parsing goal id"))
		return
	}
	// userID := ctx.Value("user_id").(int)
	if err := CompleteGoal(e.dao.DB, goalID); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "completing goal"))
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleGoalIncomplete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	goalID, err := strconv.Atoi(params["id"])
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "parsing goal id"))
		return
	}

	// userID := ctx.Value("user_id").(int)
	if err := IncompleteGoal(e.dao.DB, goalID); err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "incompleting goal"))
		return
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleGetGoals(w http.ResponseWriter, r *http.Request) {
	l, err := GetGoalsByUserID(e.dao.ReadDB, r.Context().Value("user_id").(int))
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, l)
}

func (e *External) HandleGetAllUserGoalTemplates(w http.ResponseWriter, r *http.Request) {
	t, err := GetAllGoalTemplates(e.dao.ReadDB)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, t)
}

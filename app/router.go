package external

import (
	"net/http"

	"github.com/rs/cors"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func Router(
	e *External,
	log *logrus.Entry,
	allowedOrigins []string,
) (
	http.Handler, *mux.Router, error,
) {
	r := mux.NewRouter()

	// Global
	// Unauthed
	r.Use(mux.MiddlewareFunc(e.DeviceUniqueIDParser))
	r.HandleFunc("/ping", e.HandlePing).Methods(http.MethodGet)

	// route to /api/v0.1/
	a := r.PathPrefix("/api/v0.1").Subrouter()

	// Users
	// Unauthed /user
	userUnAuthed := a.PathPrefix("/user").Subrouter()
	userUnAuthed.
		HandleFunc("", e.HandleSignUp).
		Methods(http.MethodPost)
	userUnAuthed.
		HandleFunc("/login", e.HandleLogin).
		Methods(http.MethodPost)
	userUnAuthed.
		HandleFunc("/token/refresh", e.HandleRefreshToken).
		Methods(http.MethodGet)
	userUnAuthed.
		HandleFunc("/password/forgotten", e.HandleForgotPassword).
		Methods(http.MethodPost)
	userUnAuthed.
		HandleFunc("/password/reset", e.HandleResetPassword).
		Methods(http.MethodPost)

	// Unauthed User Social
	userSocialUnAuthed := userUnAuthed.PathPrefix("/social").Subrouter()
	userSocialUnAuthed.
		HandleFunc("/login", e.HandleSocialLogin).
		Methods(http.MethodPost).
		Queries(
			"access_token", "{access_token}",
			"access_secret", "{access_secret}",
			"social_network", "{social_network}",
		)
	userSocialUnAuthed.
		HandleFunc("/signup", e.HandleSocialSignUp).
		Methods(http.MethodPost).
		Queries(
			"access_token", "{access_token}",
			"access_secret", "{access_secret}",
			"social_network", "{social_network}",
		)

	// Authed: /user
	userAuthed := userUnAuthed.NewRoute().Subrouter()
	userAuthed.Use(mux.MiddlewareFunc(e.JWTAuthentication))
	userAuthed.
		HandleFunc("/self", e.HandleUserUpdate).
		Methods(http.MethodPut)
	userAuthed.
		HandleFunc("/self", e.HandleGetSelf).
		Methods(http.MethodGet)
	userAuthed.
		HandleFunc("/self/learning_progress", e.HandleGetLearningProgress).
		Methods(http.MethodGet)
	userAuthed.
		HandleFunc("/self/quizzes/gradings", e.HandleGetQuizGradings).
		Methods(http.MethodGet)
	userAuthed.
		HandleFunc("/self/password", e.HandlePasswordChange).
		Methods(http.MethodPut)
	userAuthed.
		HandleFunc("/self/goals", e.HandleGoalCreate).
		Methods(http.MethodPost)
	userAuthed.
		HandleFunc("/self/goals", e.HandleGetGoals).
		Methods(http.MethodGet)
	userAuthed.
		HandleFunc("/self/goals/{id:[0-9]+}/complete", e.HandleGoalComplete).
		Methods(http.MethodPut)
	userAuthed.
		HandleFunc("/self/goals/{id:[0-9]+}/incomplete", e.HandleGoalIncomplete).
		Methods(http.MethodPut)
	userAuthed.
		HandleFunc("/goals/templates", e.HandleGetAllUserGoalTemplates).
		Methods(http.MethodGet)

	// Files
	filesUnAuthed := a.PathPrefix("/files").Subrouter()
	filesAuthed := filesUnAuthed.NewRoute().Subrouter()
	filesAuthed.Use(mux.MiddlewareFunc(e.JWTAuthentication))
	// Authed
	filesAuthed.
		HandleFunc("", e.HandleFileUpload).
		Methods(http.MethodPost)

	// Modules
	modulesUnAuthed := a.PathPrefix("/modules").Subrouter()
	// Authed
	modulesAuthed := modulesUnAuthed.NewRoute().Subrouter()
	modulesAuthed.Use(mux.MiddlewareFunc(e.JWTAuthentication))
	modulesAuthed.
		HandleFunc("", e.HandleGetAllModules).
		Methods(http.MethodGet)
	modulesAuthed.
		HandleFunc("/participants", e.HandleGetAllModuleParticipants).
		Methods(http.MethodGet)
	modulesAuthed.
		HandleFunc("/search", e.HandleSearchModules).
		Methods(http.MethodGet).
		Queries("query", "{query}")
	modulesAuthed.
		HandleFunc("/record_progress", e.HandleRecordModuleProgress).
		Methods(http.MethodPost)
	modulesAuthed.
		HandleFunc("/grade", e.HandleGradeQuiz).
		Methods(http.MethodPost)

	// Waitlist
	// Unauthed /waitlist
	waitlistUnAuthed := a.PathPrefix("/waitlist").Subrouter()
	waitlistUnAuthed.
		HandleFunc("", e.HandleGetWaitlist).
		Methods(http.MethodGet)
	waitlistUnAuthed.
		HandleFunc("/user", e.HandleWaitlistUserAdd).
		Methods(http.MethodPost)

	// Mailer
	emailsUnAuthed := a.PathPrefix("/email").Subrouter()
	emailsAuthed := emailsUnAuthed.NewRoute().Subrouter()
	emailsAuthed.Use(mux.MiddlewareFunc(e.JWTAuthentication), mux.MiddlewareFunc(e.AdminAuthentication))
	emailsAuthed.
		HandleFunc("/preview", e.HandlePreviewEmail).
		Methods(http.MethodGet).
		Queries("type", "{type}")

	// Leads
	leadUnAuthed := a.PathPrefix("/leads").Subrouter()
	leadUnAuthedModules := leadUnAuthed.PathPrefix("/modules").Subrouter()
	leadUnAuthedModules.
		HandleFunc("", e.HandleGetExploreModules).
		Methods(http.MethodGet)
	leadUnAuthedModules.
		HandleFunc("/record_progress", e.HandleRecordLeadModuleProgress).
		Methods(http.MethodPost)
	leadUnAuthedModules.
		HandleFunc("/learning_progress", e.HandleGetLeadLearningProgress).
		Methods(http.MethodGet)

	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPut,
			http.MethodPost,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{"*"},
	}).Handler(r), r, nil
}

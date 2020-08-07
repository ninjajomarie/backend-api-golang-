package external

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type External struct {
	dao      *PostgresDAO
	log      *logrus.Entry
	facebook Facebook
	twitter  Twitter
	Router   *mux.Router
}

func New(
	log *logrus.Entry,
	dao *PostgresDAO,
	facebook Facebook,
	twitter Twitter,
) *External {
	return &External{
		dao:      dao,
		log:      log,
		facebook: facebook,
		twitter:  twitter,
	}
}

func (e *External) returnJSON(w http.ResponseWriter, a interface{}) {
	json, err := json.Marshal(
		struct {
			Message interface{} `json:"message,omitempty"`
		}{
			Message: a,
		},
	)
	if err != nil {
		json = []byte("json marshal error")
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(json); err != nil {
		e.log.WithError(err).Error("writing json")
	}
}

func getMessage(err error) string {
	if err == nil {
		return "unknown error"
	}
	return err.Error()
}

func (e *External) writeError(w http.ResponseWriter, r *http.Request, statusCode int, err error) {
	e.log.WithError(err).WithField("status_code", statusCode).Error("response")
	w.Header().Set("Content-Type", "application/json")

	var requestID string
	md, ok := metadata.FromOutgoingContext(r.Context())
	if ok {
		requestIDs := md.Get("request_id")
		if len(requestIDs) > 0 {
			requestID = requestIDs[0]
		}
	}

	s, ok := status.FromError(err)
	if ok {
		switch s.Code() {
		case codes.InvalidArgument:
			statusCode = http.StatusBadRequest
		case codes.AlreadyExists:
			statusCode = http.StatusConflict
		case codes.NotFound:
			statusCode = http.StatusNotFound
		case codes.Unimplemented:
			statusCode = http.StatusNotImplemented
		case codes.Internal:
			statusCode = http.StatusInternalServerError
		}
	}
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(
		struct {
			RequestID  string `json:"request_id,omitempty"`
			StatusCode int    `json:"code,omitempty"`
			Message    string `json:"message,omitempty"`
		}{
			RequestID:  requestID,
			StatusCode: statusCode,
			Message:    getMessage(err),
		},
	); err != nil {
		e.log.WithError(err).Error("writing error")
	}
}

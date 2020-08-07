package external

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func (e *External) HandlePreviewEmail(w http.ResponseWriter, r *http.Request) {
	t := EmailType(strings.ToUpper(mux.Vars(r)["type"]))
	if t == "" {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid email type: %s", t))
		return
	}

	userID := r.Context().Value("user_id").(int)
	if userID <= 0 {
		e.writeError(w, r, http.StatusForbidden, fmt.Errorf("missing user_id in context"))
		return
	}

	user, err := GetUserByID(e.dao.ReadDB, userID)
	if err != nil {
		e.writeError(w, r, http.StatusForbidden, errors.Wrapf(err, "unknown user"))
		return
	}

	m := NewMailer(e.log)
	switch t {
	case EmailType_Waitlist:
		code := "FAKE-123456"
		if err := m.SendWaitlistEmail(context.Background(), user.Email, code); err != nil {
			e.writeError(w, r, http.StatusInternalServerError, err)
			return
		}
	default:
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("no way to handle preview for email template %s", t))
		return
	}

	e.returnJSON(w, "email sent to your email address")
}

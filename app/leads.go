package external

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

func (e *External) HandleGetExploreModules(w http.ResponseWriter, r *http.Request) {
	m, err := GetModulesByIDs(e.dao.ReadDB, []int{1, 2, 3})
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "getting all modules"))
		return
	}

	mWD, err := e.injectModuleDetails(m)
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "adding module details"))
		return
	}

	e.returnJSON(w, mWD)
}

func (e *External) HandleRecordLeadModuleProgress(w http.ResponseWriter, r *http.Request) {
	p := &LearningProgress{}
	if err := json.NewDecoder(r.Body).Decode(p); err != nil {
		e.writeError(w, r, http.StatusBadRequest, fmt.Errorf("invalid request"))
		return
	}
	p.DeviceUniqueID = sql.NullString{
		String: r.Context().Value("device_unique_id").(string),
		Valid:  true,
	}

	if err := RecordModuleProgress(e.dao.DB, p); err != nil {
		// e.writeError(w, r, http.StatusInternalServerError, errors.Wrap(err, "recording module progress"))
		// return

		// just log the error
		e.log.WithError(err).Error("recording lead module progress")
	}

	e.returnJSON(w, nil)
}

func (e *External) HandleGetLeadLearningProgress(w http.ResponseWriter, r *http.Request) {
	l, err := GetLearningProgressesByDeviceUniqueID(e.dao.ReadDB, r.Context().Value("device_unique_id").(string))
	if err != nil {
		e.writeError(w, r, http.StatusInternalServerError, err)
		return
	}

	e.returnJSON(w, l)
}

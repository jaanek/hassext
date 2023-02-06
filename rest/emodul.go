package rest

import (
	"net/http"

	"github.com/jaanek/hassext/emodul"
)

func HandleBoilerFireUp(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := rest.em.BoilerFireUp()
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{})
	}
}

func HandleBoilerDamping(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := rest.em.BoilerDamping()
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{})
	}
}

func HandleWorkingMode(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req = struct {
			Mode string `json:"mode"`
		}{}
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}

		// parse the mode
		var mode emodul.WorkingMode
		switch req.Mode {
		case "house_heating":
			mode = emodul.HOUSE_HEATING
		case "parallel_pumps":
			mode = emodul.PARALLEL_PUMPS
		case "summer_mode":
			mode = emodul.SUMMER_MODE
		default:
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}

		// set specified working mode
		err = rest.em.SetWorkingMode(mode)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{})
	}
}

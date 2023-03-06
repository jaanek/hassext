package rest

import (
	"fmt"
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

func HandleSetBufferTargetTemps(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req = struct {
			TopTemp    int `json:"temp-top"`
			BottomTemp int `json:"temp-bottom"`
		}{}
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}

		// validate input
		if req.TopTemp < 10 || req.TopTemp > 70 {
			HttpError(rest.lo, w, r, http.StatusBadRequest, fmt.Errorf("Invalid top temp value: %v", req.TopTemp))
			return
		}
		if req.BottomTemp < 10 || req.BottomTemp > 70 {
			HttpError(rest.lo, w, r, http.StatusBadRequest, fmt.Errorf("Invalid bottom temp value: %v", req.BottomTemp))
			return
		}

		// set temperature targets
		err = rest.em.SetBufferTargetTemp(emodul.BUFFER_BOTTOM, uint(req.BottomTemp))
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		err = rest.em.SetBufferTargetTemp(emodul.BUFFER_TOP, uint(req.TopTemp))
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
			Mode       string `json:"mode"`
			IgnoreWhen string `json:"ignore_when"`
		}{}
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}

		// check if we need to ignore the request if boiler is in certain mode
		switch req.IgnoreWhen {
		case "summer_mode":

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

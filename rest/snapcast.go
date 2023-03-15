package rest

import (
	"fmt"
	"net/http"
)

func HandleSnapcastClientMute(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req = struct {
			ClientId string `json:"client-id"`
			Mute     bool   `json:"mute"`
		}{}
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}
		rest.lo.Info("Snapcast set client mute", "request", req)

		// set snapcast client mute/unmute
		client, err := rest.sc.ClientGetStatus(req.ClientId)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, fmt.Errorf("Snapcast get client error: %w", err))
			return
		}
		err = rest.sc.ClientMute(client, req.Mute)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{})
	}
}

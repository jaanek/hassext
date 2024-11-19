package rest

import (
	"fmt"
	"net/http"
	"strings"
)

func HandleSnapcastClientMute(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req = struct {
			ClientId string `json:"client-id"`
			Mute     string `json:"mute"`
		}{}
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, err)
			return
		}
		// because req.Mute comes in as with value "True" - uppercase, quick fix is to just take it in as string and check that
		var mute = false
		switch strings.ToLower(req.Mute) {
		case "true":
			mute = true
		}
		rest.lo.Info("Snapcast set client mute", "request", req)

		// set snapcast client mute/unmute
		client, err := rest.sc.ClientGetStatus(req.ClientId)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, fmt.Errorf("Snapcast get client error: %w", err))
			return
		}
		err = rest.sc.ClientMute(client, mute)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, err)
			return
		}
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{})
	}
}

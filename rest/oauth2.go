package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/bmf-san/goblin"
	"github.com/jaanek/go-oauth2"
)

var services = map[string]oauth2.Service{
	"microsoft": {
		"client_id":          "",
		"client_secret":      "",
		"authType":           "pkce", // secret, pkce
		"authorize_endpoint": "",
		"redirect_uri":       "",
		"scope":              "User.Read",
		"prompt":             "login",
		"token_endpoint":     "",
		"post_type":          "form", // json
		"refresh_allowed":    "true",
	},
}

func HandleOAuthLink(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceKey := goblin.GetParam(r.Context(), "service")
		service := services[serviceKey]
		if service == nil {
			HttpError(rest.lo, w, r, http.StatusBadRequest, errors.New(fmt.Sprintf("Unknown service key: %v", serviceKey)))
			return
		}
		authLink, state := oauth2.AuthLink(serviceKey, service, service["authType"])
		oauth2.SetState(state.Key, state)
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{
			"link": authLink,
		})
	}
}

func HandleOAuthResponse(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, err)
			return
		}
		code := m.Get("code")
		state := oauth2.GetState(m.Get("state"))
		if state == nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, errors.New("State not found"))
			return
		}
		service := services[state.Service]
		if service == nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, errors.New(fmt.Sprintf("Unknown service key in state: %v", state.Service)))
			return
		}
		exchangeToken, err := oauth2.ExchangeToken(state, service, code, "https://auth.coldborecapital.com")
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, err)
			return
		}
		rest.lo.Debug("exchange", "token", exchangeToken)

		// parse the token and put into jwt cookie
		type ExchangeToken struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			Scope        string `json:"scope"`
			TokenType    string `json:"token_type"`
		}
		exchangeData := &ExchangeToken{}
		err = json.Unmarshal([]byte(exchangeToken), &exchangeData)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, err)
			return
		}

		// read it
		resp, err := get("https://graph.microsoft.com/v1.0/me", exchangeData.AccessToken)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, err)
			return
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return
		}
		if resp.StatusCode != 200 {
			err = errors.New(string(body))
			return
		}
		rest.lo.Debug("My data", "body", string(body))

		type MyData struct {
			DisplayName       string `json:"displayName"`
			GivenName         string `json:"givenName"`
			Surname           string `json:"surname"`
			UserPrincipalName string `json:"userPrincipalName"`
			Id                string `json:"id"`
			JobTitle          string `json:"jobTitle"`
			MobilePhone       string `json:"mobilePhone"`
		}
		myData := &MyData{}
		err = json.Unmarshal(body, &myData)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusUnauthorized, err)
			return
		}

		// create cookie
		cookie, _, err := CreateJwtCookie(myData.DisplayName, myData.UserPrincipalName, rest.jwtSecret)
		if err != nil {
			HttpError(rest.lo, w, r, http.StatusInternalServerError, errors.New(fmt.Sprintf("%v", err)))
			return
		}
		http.SetCookie(w, cookie)

		// HttpJson(backend.Log, w, r, http.StatusOK, map[string]interface{}{
		// 	"token": token,
		// })
		http.Redirect(w, r, "/", 302)
	}
}

func get(url string, accessToken string) (resp *http.Response, err error) {
	var client = &http.Client{
		Timeout: time.Second * 10,
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	// req.Header.Set("Origin", httpOrigin)
	return client.Do(req)
}

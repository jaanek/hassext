package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bmf-san/goblin"
	"github.com/golang-jwt/jwt"
	jwtmiddleware "github.com/jaanek/go-jwt-middleware"
	"github.com/jaanek/go-oauth2"
	"github.com/urfave/negroni"
	"github.com/zerodha/logf"
)

func NewRouter(rest *Rest) http.Handler {
	jwtMiddleware := jwtmiddleware.UseJwtMiddleware(
		func(w http.ResponseWriter, req *http.Request, err error) {
			HttpError(rest.lo, w, req, http.StatusUnauthorized, err)
		},
		func(token *jwt.Token) (interface{}, error) {
			// NB! verify that we are dealing with the same signing method used when singning jwt token
			// https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(rest.jwtSecret), nil
		},
		JwtUserKey,
		func() jwt.Claims {
			return JwtClaims{}
		},
	)
	oauth2.InitOauth2()

	// router
	r := goblin.NewRouter()
	r.Methods(http.MethodPost).Handler("/login", HandleLoginUP(rest))
	r.Methods(http.MethodPost).Handler("/logout", HandleLogoutUP(rest))
	r.Methods(http.MethodGet).Handler("/oauth/link/:service", HandleOAuthLink(rest))
	r.Methods(http.MethodGet).Handler("/oauth/response", HandleOAuthResponse(rest))
	r.Methods(http.MethodGet).Handler("/test", handlePing(rest))
	r.Methods(http.MethodGet).Use(jwtMiddleware).Handler("/ping", handleTest(rest))
	// emodul
	r.Methods(http.MethodPost).Handler("/boiler-fireup", HandleBoilerFireUp(rest))
	r.Methods(http.MethodPost).Handler("/boiler-damping", HandleBoilerDamping(rest))
	r.Methods(http.MethodPost).Handler("/working-mode", HandleWorkingMode(rest))
	r.Methods(http.MethodPost).Handler("/set-buffer-target-temps", HandleSetBufferTargetTemps(rest))
	r.Methods(http.MethodPost).Handler("/snapcast-client-mute", HandleSnapcastClientMute(rest))

	// common middleware's
	// n := negroni.Classic() // Includes some default middlewares
	n := negroni.New()
	n.Use(negroni.NewRecovery())
	// n.Use(negroni.NewStatic(http.Dir("public")))
	n.UseHandler(r)
	return n
}

func handlePing(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{
			"ping": "pong",
		})
	}
}

func handleTest(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := r.Context().Value(JwtUserKey).(*jwt.Token)
		HttpJson(rest.lo, w, r, http.StatusOK, map[string]interface{}{
			"token": token.Raw,
		})
	}
}

func HttpBind(input io.Reader, v interface{}) error {
	return json.NewDecoder(input).Decode(v)
}

func HttpJson(lo logf.Logger, w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	lo.Debug("[RESPONSE]", "url", r.URL, "body", data)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}

func HttpError(lo logf.Logger, w http.ResponseWriter, r *http.Request, code int, error error) {
	lo.Error("[ERROR]", "url", r.URL, "error", error)
	http.Error(w, error.Error(), code)
}

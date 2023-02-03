package rest

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt"
	"github.com/jaanek/hassext/model"
	"golang.org/x/crypto/bcrypt"
)

type JwtClaims struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	// Oauth2Token string `json:"oauth2token"`
	jwt.StandardClaims
}

func HandleLoginUP(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get the request data
		type loginRequest struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		var req loginRequest
		err := HttpBind(r.Body, &req)
		if err != nil {
			HttpError(rest.Lo, w, r, http.StatusBadRequest, err)
			return
		}
		_ = strings.TrimSpace(req.Username)
		password := strings.TrimSpace(req.Password)

		// Check auth info against database
		loginUP := &model.LoginUP{}
		// if err := backend.DB.Get(loginUP, fmt.Sprintf("select * from %s where username = $1 limit 1", loginUP.TableName()), username); err != nil {
		// 	HttpError(backend.Log, w, r, http.StatusUnauthorized, errors.New(fmt.Sprintf("No username found: %v! %v", username, err)))
		// 	return
		// }
		// Comparing the password with the hash
		if err := bcrypt.CompareHashAndPassword([]byte(loginUP.Password), []byte(password)); err != nil {
			HttpError(rest.Lo, w, r, http.StatusUnauthorized, errors.New(fmt.Sprintf("Password mismatch! %v", err)))
			return
		}

		// create jwt user
		cookie, token, err := CreateJwtCookie(loginUP.Name, loginUP.Email, rest.JwtSecret)
		if err != nil {
			HttpError(rest.Lo, w, r, http.StatusInternalServerError, errors.New(fmt.Sprintf("%v", err)))
			return
		}
		http.SetCookie(w, cookie)

		// send token
		HttpJson(rest.Lo, w, r, http.StatusOK, map[string]interface{}{
			"token": token,
		})
	}
}

func HandleLogoutUP(rest *Rest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// delete auth cookie
		cookie := new(http.Cookie)
		cookie.Name = AuthCookieName
		cookie.Value = ""
		cookie.MaxAge = 0 // deletes the cookie
		http.SetCookie(w, cookie)
		HttpJson(rest.Lo, w, r, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}

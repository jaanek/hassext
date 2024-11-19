package rest

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt"
)

const AuthCookieName = "jwt-auth"

func CreateJwtCookie(name, email, secret string) (*http.Cookie, string, error) {
	expiresAt := time.Now().Add(time.Hour * 24)
	jwtUser := &JwtClaims{
		name,
		email,
		jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
		},
	}
	// Create token with authenticated user data
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtUser)

	// generate encoded token and send it as reponse
	t, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, "", err
	}

	// write auth cookie
	cookie := new(http.Cookie)
	cookie.Name = AuthCookieName
	cookie.Value = t
	cookie.Expires = expiresAt
	cookie.Secure = true
	cookie.Path = "/"
	return cookie, t, nil
}

// cookie helpers
func setCookie(w http.ResponseWriter, cookieName string, token string, secure bool, sameSite http.SameSite) {
	tok64 := base64.StdEncoding.EncodeToString([]byte(token))
	cookie := http.Cookie{
		Name:     cookieName,
		Value:    tok64,
		HttpOnly: true,   // make it so that browser's javascript cannot read/see the cookie
		Secure:   secure, // use true for production
		Path:     "/",
		SameSite: sameSite, // http.SameSiteLaxMode,
	}
	http.SetCookie(w, &cookie)
	return
}
func getCookie(r *http.Request, cookieName string) (token string, err error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return
	}
	tokb, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return
	}
	token = string(tokb)
	return
}

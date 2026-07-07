package auth

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

func SetReturnToCookie(w http.ResponseWriter, url string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "return_to",
		Value:    url,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   300,
		SameSite: http.SameSiteLaxMode,
	})
}

func DropReturnToCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "return_to",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // <0 writes "Max-Age: 0" → delete now
		HttpOnly: true,
		Secure:   true,
	})
}

func SetSessionCookie(w http.ResponseWriter, session Session, rawToken SessionToken) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   int(time.Until(session.ExpiresAt).Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
}

func DropSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
	})
}

func SetUserNameCookie(w http.ResponseWriter, name string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "user_name",
		Value:    strings.ReplaceAll(url.QueryEscape(name), "+", "%20"),
		Path:     "/",
		Secure:   true,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		SameSite: http.SameSiteLaxMode,
	})
}

func DropUserNameCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user_name",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

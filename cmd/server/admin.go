package main

import (
	"net/http"
	"os"
	"time"
)

const adminSessionCookie = "fitnation_admin"

var adminTokens = map[string]time.Time{}

func (a *App) adminLogin(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.currentAdmin(r) {
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}
		a.render(w, "admin_login.html", nil)

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			badRequest(w)
			return
		}

		username := r.FormValue("username")
		password := r.FormValue("password")

		adminUser := os.Getenv("ADMIN_USERNAME")
		adminPass := os.Getenv("ADMIN_PASSWORD")
		if adminUser == "" {
			adminUser = "admin"
		}
		if adminPass == "" {
			adminPass = "admin1234"
		}

		if username != adminUser || password != adminPass {
			a.render(w, "admin_login.html", map[string]any{
				"Error": "Identifiants incorrects.",
			})
			return
		}

		token, err := randomToken(32)
		if err != nil {
			serverError(w, err)
			return
		}

		adminTokens[token] = time.Now().Add(2 * time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:     adminSessionCookie,
			Value:    token,
			Path:     "/",
			Expires:  time.Now().Add(2 * time.Hour),
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})

		http.Redirect(w, r, "/admin", http.StatusSeeOther)

	default:
		methodNotAllowed(w)
	}
}

func (a *App) adminLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}
	cookie, err := r.Cookie(adminSessionCookie)
	if err == nil {
		delete(adminTokens, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookie,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (a *App) currentAdmin(r *http.Request) bool {
	cookie, err := r.Cookie(adminSessionCookie)
	if err != nil || cookie.Value == "" {
		return false
	}
	expiry, ok := adminTokens[cookie.Value]
	if !ok || time.Now().After(expiry) {
		delete(adminTokens, cookie.Value)
		return false
	}
	return true
}

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !a.currentAdmin(r) {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return false
	}
	return true
}

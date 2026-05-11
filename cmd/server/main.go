package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"fitnation/internal/database"
	"fitnation/internal/models"
)

const sessionCookieName = "fitnation_session"

type App struct {
	store *database.Store
}

func loadEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		os.Setenv(strings.TrimSpace(key), strings.TrimSpace(val))
	}
}

func main() {
	loadEnv(".env")

	dbPath := os.Getenv("FITNATION_DB")
	if dbPath == "" {
		dbPath = "fitnation.db"
	}

	store, err := database.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	app := &App{store: store}
	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.HandleFunc("/", app.home)
	mux.HandleFunc("/login", app.login)
	mux.HandleFunc("/logout", app.logout)
	mux.HandleFunc("/logout/", app.logout)
	mux.HandleFunc("/register", app.register)
	mux.HandleFunc("/forgot-password", app.forgotPassword)
	mux.HandleFunc("/forgot-password/", app.forgotPassword)
	mux.HandleFunc("/reset-password", app.resetPassword)
	mux.HandleFunc("/reset-password/", app.resetPassword)
	mux.HandleFunc("/profile", app.profile)
	mux.HandleFunc("/profile/update", app.profileUpdate)
	mux.HandleFunc("/profile/delete", app.profileDelete)
	mux.HandleFunc("/network", app.network)
	mux.HandleFunc("/posts/new", app.newPost)
	mux.HandleFunc("/posts/create", app.createPost)
	mux.HandleFunc("/posts/", app.postsRouter)
	mux.HandleFunc("/api/posts/", app.likePost)
	mux.HandleFunc("/api/comments/create", app.createComment)
	mux.HandleFunc("/api/comments/", app.commentsRouter)
	mux.HandleFunc("/admin", app.admin)
	mux.HandleFunc("/users/", app.userProfile)

	log.Println("FITNATION lancé sur http://localhost:8000")
	log.Fatal(http.ListenAndServe(":8000", mux))
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	user := a.currentUser(r)
	userID := 0
	if user != nil {
		userID = user.ID
	}

	sort := r.URL.Query().Get("sort")
	period := r.URL.Query().Get("date")
	minLikesStr := r.URL.Query().Get("likes")
	minLikes := 0
	if minLikesStr != "" {
		if v, err := strconv.Atoi(minLikesStr); err == nil && v >= 0 {
			minLikes = v
		}
	}

	posts, err := a.store.ListPosts(userID, sort, minLikes, period)
	if err != nil {
		serverError(w, err)
		return
	}

	users, err := a.store.ListUsers()
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "home.html", map[string]any{
		"CurrentUser": user,
		"Posts":       posts,
		"Users":       users,
		"Sort":        sort,
	})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.currentUser(r) != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		a.render(w, "login.html", nil)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			badRequest(w)
			return
		}

		identifier := strings.TrimSpace(r.FormValue("username"))
		if identifier == "" {
			identifier = strings.TrimSpace(r.FormValue("email"))
		}

		user, err := a.store.UserByEmailOrUsername(identifier)
		if err != nil {
			serverError(w, err)
			return
		}
		if user == nil || !checkPassword(r.FormValue("password"), user.PasswordHash) {
			http.Error(w, "Identifiants incorrects", http.StatusUnauthorized)
			return
		}

		if err := a.startSession(w, user.ID); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		_ = a.store.DeleteSession(cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if a.currentUser(r) != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		a.render(w, "register.html", nil)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			badRequest(w)
			return
		}

		fullName := strings.TrimSpace(r.FormValue("full_name"))
		email := strings.TrimSpace(r.FormValue("email"))
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		confirm := r.FormValue("password_confirm")

		if fullName == "" || email == "" || username == "" || password == "" || password != confirm {
			http.Error(w, "Formulaire invalide", http.StatusBadRequest)
			return
		}

		passwordHash, err := hashPassword(password)
		if err != nil {
			serverError(w, err)
			return
		}

		userID, err := a.store.CreateUser(fullName, username, email, passwordHash)
		if err != nil {
			http.Error(w, "Email ou pseudo déjà utilisé", http.StatusBadRequest)
			return
		}

		if err := a.startSession(w, userID); err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) forgotPassword(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.render(w, "forgot_password.html", nil)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			badRequest(w)
			return
		}

		email := strings.TrimSpace(r.FormValue("email"))
		if email == "" {
			badRequest(w)
			return
		}

		user, err := a.store.UserByEmail(email)
		if err != nil {
			serverError(w, err)
			return
		}

		resetURL := ""
		emailSent := false

		if user != nil {
			token, err := randomToken(32)
			if err != nil {
				serverError(w, err)
				return
			}

			expiresAt := time.Now().Add(1 * time.Hour)
			if err := a.store.CreatePasswordReset(token, user.ID, expiresAt); err != nil {
				serverError(w, err)
				return
			}

			link := resetLink(r, token)
			resetURL = link
			emailSent = sendResetEmail(user.Email, link)
		}

		data := map[string]any{
			"Message": "Si cet email existe, un lien de réinitialisation a été envoyé.",
		}
		if resetURL != "" && !emailSent {
			data["DevLink"] = resetURL
		}

		a.render(w, "forgot_password.html", data)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) resetPassword(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		token := strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			http.NotFound(w, r)
			return
		}

		user, err := a.store.UserByPasswordReset(token)
		if err != nil {
			serverError(w, err)
			return
		}
		if user == nil {
			http.Error(w, "Lien invalide ou expiré", http.StatusBadRequest)
			return
		}

		a.render(w, "reset_password.html", map[string]any{"Token": token})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			badRequest(w)
			return
		}

		token := strings.TrimSpace(r.FormValue("token"))
		password := r.FormValue("password")
		confirm := r.FormValue("password_confirm")
		if token == "" || password == "" || password != confirm {
			http.Error(w, "Formulaire invalide", http.StatusBadRequest)
			return
		}

		user, err := a.store.UserByPasswordReset(token)
		if err != nil {
			serverError(w, err)
			return
		}
		if user == nil {
			http.Error(w, "Lien invalide ou expiré", http.StatusBadRequest)
			return
		}

		passwordHash, err := hashPassword(password)
		if err != nil {
			serverError(w, err)
			return
		}

		if err := a.store.UpdatePassword(user.ID, passwordHash); err != nil {
			serverError(w, err)
			return
		}
		if err := a.store.DeletePasswordReset(token); err != nil {
			serverError(w, err)
			return
		}

		http.Redirect(w, r, "/login", http.StatusSeeOther)
	default:
		methodNotAllowed(w)
	}
}

func (a *App) profile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	stats, err := a.store.StatsForUser(user.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	posts, err := a.store.UserPosts(user.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "profile.html", map[string]any{
		"CurrentUser": user,
		"User":        user,
		"Stats":       stats,
		"Posts":       posts,
	})
}

func (a *App) profileUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		badRequest(w)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	if username == "" || email == "" {
		http.Error(w, "Pseudo et email obligatoires", http.StatusBadRequest)
		return
	}

	passwordHash := ""
	if password != "" {
		hash, err := hashPassword(password)
		if err != nil {
			serverError(w, err)
			return
		}
		passwordHash = hash
	}

	if err := a.store.UpdateUser(user.ID, username, email, passwordHash); err != nil {
		http.Error(w, "Impossible de mettre à jour le profil", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (a *App) profileDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	if err := a.store.DeleteUser(user.ID); err != nil {
		serverError(w, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/register", http.StatusSeeOther)
}

func (a *App) network(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	users, err := a.store.ListUsers()
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "network.html", map[string]any{
		"CurrentUser": a.currentUser(r),
		"Users":       users,
	})
}

func (a *App) newPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	a.render(w, "new_post.html", map[string]any{"CurrentUser": user})
}

func (a *App) createPost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	imageURL := strings.TrimSpace(r.FormValue("image_url"))
	tags := cleanTags(r.FormValue("tags"))
	if title == "" || content == "" {
		http.Error(w, "Titre et contenu obligatoires", http.StatusBadRequest)
		return
	}

	postID, err := a.store.CreatePost(user.ID, title, content, imageURL, tags)
	if err != nil {
		serverError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusSeeOther)
}

func (a *App) postsRouter(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/posts/"))
	if len(parts) == 0 {
		http.NotFound(w, r)
		return
	}

	postID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if len(parts) == 1 && r.Method == http.MethodGet {
		a.showPost(w, r, postID)
		return
	}
	if len(parts) == 2 && parts[1] == "edit" && r.Method == http.MethodGet {
		a.editPost(w, r, postID)
		return
	}
	if len(parts) == 2 && parts[1] == "update" && r.Method == http.MethodPost {
		a.updatePost(w, r, postID)
		return
	}
	if len(parts) == 2 && parts[1] == "delete" && r.Method == http.MethodPost {
		a.deletePost(w, r, postID)
		return
	}

	http.NotFound(w, r)
}

func (a *App) showPost(w http.ResponseWriter, r *http.Request, postID int) {
	user := a.currentUser(r)
	userID := 0
	if user != nil {
		userID = user.ID
	}

	post, err := a.store.PostByID(postID, userID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	comments, err := a.store.CommentsByPost(postID)
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "post_detail.html", map[string]any{
		"CurrentUser": user,
		"Post":        post,
		"Comments":    comments,
	})
}

func (a *App) editPost(w http.ResponseWriter, r *http.Request, postID int) {
	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	post, err := a.store.PostByID(postID, user.ID)
	if err != nil || post.UserID != user.ID {
		http.NotFound(w, r)
		return
	}

	a.render(w, "edit_post.html", map[string]any{
		"CurrentUser": user,
		"Post":        post,
	})
}

func (a *App) updatePost(w http.ResponseWriter, r *http.Request, postID int) {
	user := a.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	content := strings.TrimSpace(r.FormValue("content"))
	imageURL := strings.TrimSpace(r.FormValue("image_url"))
	tags := cleanTags(r.FormValue("tags"))
	if title == "" || content == "" {
		http.Error(w, "Titre et contenu obligatoires", http.StatusBadRequest)
		return
	}

	if err := a.store.UpdatePost(postID, user.ID, title, content, imageURL, tags); err != nil {
		serverError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusSeeOther)
}

func (a *App) deletePost(w http.ResponseWriter, r *http.Request, postID int) {
	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	if err := a.store.DeletePost(postID, user.ID); err != nil {
		serverError(w, err)
		return
	}

	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

func (a *App) likePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	user := a.currentUser(r)
	if user == nil {
		http.Error(w, "Connexion requise", http.StatusUnauthorized)
		return
	}

	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/posts/"))
	if len(parts) != 2 || parts[1] != "like" {
		http.NotFound(w, r)
		return
	}

	postID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	liked, count, err := a.store.ToggleLike(postID, user.ID)
	if err != nil {
		serverError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"liked":       liked,
		"likes_count": count,
	})
}

func (a *App) createComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		badRequest(w)
		return
	}

	postID, err := strconv.Atoi(r.FormValue("post_id"))
	if err != nil {
		badRequest(w)
		return
	}

	content := strings.TrimSpace(r.FormValue("content"))
	if content == "" {
		http.Error(w, "Commentaire vide", http.StatusBadRequest)
		return
	}

	if err := a.store.CreateComment(postID, user.ID, content); err != nil {
		serverError(w, err)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/posts/%d", postID), http.StatusSeeOther)
}

func (a *App) commentsRouter(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(strings.TrimPrefix(r.URL.Path, "/api/comments/"))
	if len(parts) != 2 || parts[1] != "delete" || r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	commentID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if err := a.store.DeleteComment(commentID, user.ID); err != nil {
		serverError(w, err)
		return
	}

	redirectBack(w, r, "/")
}

func (a *App) admin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	user := a.requireUser(w, r)
	if user == nil {
		return
	}

	users, err := a.store.ListUsers()
	if err != nil {
		serverError(w, err)
		return
	}

	posts, err := a.store.ListPosts(user.ID, "", 0, "")
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "admin.html", map[string]any{
		"CurrentUser": user,
		"Users":       users,
		"Posts":       posts,
	})
}

func (a *App) userProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	userID, err := strconv.Atoi(idStr)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	member, err := a.store.UserByID(userID)
	if err != nil || member == nil {
		http.NotFound(w, r)
		return
	}

	posts, err := a.store.UserPosts(userID)
	if err != nil {
		serverError(w, err)
		return
	}

	stats, err := a.store.StatsForUser(userID)
	if err != nil {
		serverError(w, err)
		return
	}

	a.render(w, "user_profile.html", map[string]any{
		"CurrentUser": a.currentUser(r),
		"Member":      member,
		"Posts":       posts,
		"Stats":       stats,
	})
}

func (a *App) currentUser(r *http.Request) *models.User {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	user, err := a.store.UserBySession(cookie.Value)
	if err != nil {
		return nil
	}
	return user
}

func (a *App) requireUser(w http.ResponseWriter, r *http.Request) *models.User {
	user := a.currentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil
	}
	return user
}

func (a *App) startSession(w http.ResponseWriter, userID int) error {
	token, err := randomToken(32)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := a.store.CreateSession(token, userID, expiresAt); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (a *App) render(w http.ResponseWriter, page string, data any) {
	tmpl, err := template.ParseFiles("web/templates/layout.html", "web/templates/"+page)
	if err != nil {
		serverError(w, err)
		return
	}

	if data == nil {
		data = map[string]any{}
	}

	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		serverError(w, err)
	}
}

func hashPassword(password string) (string, error) {
	salt, err := randomBytes(16)
	if err != nil {
		return "", err
	}

	iterations := 120000
	hash := pbkdf2SHA256([]byte(password), salt, iterations, 32)
	return fmt.Sprintf("pbkdf2_sha256$%d$%s$%s", iterations, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}

func checkPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha256" {
		return false
	}

	iterations, err := strconv.Atoi(parts[1])
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}

	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}

	actual := pbkdf2SHA256([]byte(password), salt, iterations, len(expected))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func pbkdf2SHA256(password, salt []byte, iterations, keyLen int) []byte {
	hashLen := sha256.Size
	blocks := (keyLen + hashLen - 1) / hashLen
	key := make([]byte, 0, blocks*hashLen)

	for block := 1; block <= blocks; block++ {
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
		u := mac.Sum(nil)
		t := make([]byte, len(u))
		copy(t, u)

		for i := 1; i < iterations; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}

		key = append(key, t...)
	}

	return key[:keyLen]
}

func randomToken(size int) (string, error) {
	b, err := randomBytes(size)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func randomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	return b, err
}

func splitPath(path string) []string {
	var parts []string
	for _, part := range strings.Split(path, "/") {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func cleanTags(value string) string {
	seen := make(map[string]bool)
	var tags []string

	for _, tag := range strings.Split(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		tags = append(tags, tag)
	}

	return strings.Join(tags, ",")
}

func resetLink(r *http.Request, token string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/reset-password?token=%s", scheme, r.Host, token)
}

func sendResetEmail(to, link string) bool {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	password := os.Getenv("SMTP_PASSWORD")
	from := os.Getenv("SMTP_FROM")

	if host == "" || port == "" || user == "" || password == "" || from == "" {
		log.Println("Lien de réinitialisation pour", to+":", link)
		return false
	}

	auth := smtp.PlainAuth("", user, password, host)
	message := "" +
		"To: " + to + "\r\n" +
		"Subject: Réinitialisation du mot de passe FITNATION\r\n" +
		"\r\n" +
		"Bonjour,\r\n\r\n" +
		"Pour réinitialiser ton mot de passe, clique sur ce lien :\r\n" +
		link + "\r\n\r\n" +
		"Ce lien expire dans 1 heure.\r\n"

	if err := smtp.SendMail(host+":"+port, auth, from, []string{to}, []byte(message)); err != nil {
		log.Println("Erreur envoi email:", err)
		log.Println("Lien de réinitialisation pour", to+":", link)
		return false
	}

	return true
}

func redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	target := r.Referer()
	if target == "" {
		target = fallback
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func methodNotAllowed(w http.ResponseWriter) {
	http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
}

func badRequest(w http.ResponseWriter) {
	http.Error(w, "Requête invalide", http.StatusBadRequest)
}

func serverError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, "Erreur serveur", http.StatusInternalServerError)
}
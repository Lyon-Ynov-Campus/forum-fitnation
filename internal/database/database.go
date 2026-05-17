package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"fitnation/internal/models"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		db.Close()
		return nil, err
	}

	store := &Store{DB: db}
	if err := store.Migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.DB.Close()
}

func (s *Store) Migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			full_name TEXT NOT NULL,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			avatar_url TEXT NOT NULL DEFAULT '/static/images/avatar.svg',
			bio TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			image_url TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS post_likes (
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (post_id, user_id),
			FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS password_resets (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}

	for _, query := range queries {
		if _, err := s.DB.Exec(query); err != nil {
			return err
		}
	}

	alterQueries := []string{
		`ALTER TABLE posts ADD COLUMN image_url TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE posts ADD COLUMN tags TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN banned INTEGER NOT NULL DEFAULT 0`,
	}

	for _, query := range alterQueries {
		_, _ = s.DB.Exec(query)
	}

	return nil
}

func (s *Store) CreateUser(fullName, username, email, passwordHash string) (int, error) {
	result, err := s.DB.Exec(
		`INSERT INTO users (full_name, username, email, password_hash) VALUES (?, ?, ?, ?)`,
		fullName, username, email, passwordHash,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func (s *Store) UserByEmailOrUsername(identifier string) (*models.User, error) {
	row := s.DB.QueryRow(
		`SELECT id, full_name, username, email, password_hash, avatar_url, bio, created_at, banned
		FROM users WHERE email = ? OR username = ?`,
		identifier, identifier,
	)
	return scanUser(row)
}

func (s *Store) UserByEmail(email string) (*models.User, error) {
	row := s.DB.QueryRow(
		`SELECT id, full_name, username, email, password_hash, avatar_url, bio, created_at, banned
		FROM users WHERE email = ?`,
		email,
	)
	return scanUser(row)
}

func (s *Store) UserByID(id int) (*models.User, error) {
	row := s.DB.QueryRow(
		`SELECT id, full_name, username, email, password_hash, avatar_url, bio, created_at, banned
		FROM users WHERE id = ?`,
		id,
	)
	return scanUser(row)
}

func (s *Store) UpdateUser(id int, username, email, passwordHash string) error {
	if passwordHash == "" {
		_, err := s.DB.Exec(`UPDATE users SET username = ?, email = ? WHERE id = ?`, username, email, id)
		return err
	}
	_, err := s.DB.Exec(
		`UPDATE users SET username = ?, email = ?, password_hash = ? WHERE id = ?`,
		username, email, passwordHash, id,
	)
	return err
}

func (s *Store) UpdatePassword(userID int, passwordHash string) error {
	_, err := s.DB.Exec(`UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	return err
}

func (s *Store) DeleteUser(id int) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

func (s *Store) ListUsers() ([]models.User, error) {
	rows, err := s.DB.Query(
		`SELECT u.id, u.full_name, u.username, u.email, u.avatar_url, u.bio, COUNT(p.id) AS posts_count, u.banned
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id
		GROUP BY u.id
		ORDER BY u.username ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		var banned int
		if err := rows.Scan(&user.ID, &user.FullName, &user.Username, &user.Email, &user.AvatarURL, &user.Bio, &user.PostsCount, &banned); err != nil {
			return nil, err
		}
		user.Banned = banned == 1
		user.Username = displayUsername(user.Username)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) CreateSession(token string, userID int, expiresAt time.Time) error {
	_, err := s.DB.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) UserBySession(token string) (*models.User, error) {
	row := s.DB.QueryRow(
		`SELECT u.id, u.full_name, u.username, u.email, u.password_hash, u.avatar_url, u.bio, u.created_at, u.banned
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ? AND s.expires_at > ? AND u.banned = 0`,
		token, time.Now().UTC().Format(time.RFC3339),
	)
	return scanUser(row)
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

func (s *Store) CreatePasswordReset(token string, userID int, expiresAt time.Time) error {
	_, err := s.DB.Exec(`DELETE FROM password_resets WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	_, err = s.DB.Exec(
		`INSERT INTO password_resets (token, user_id, expires_at) VALUES (?, ?, ?)`,
		token, userID, expiresAt.UTC().Format(time.RFC3339),
	)
	return err
}

func (s *Store) UserByPasswordReset(token string) (*models.User, error) {
	row := s.DB.QueryRow(
		`SELECT u.id, u.full_name, u.username, u.email, u.password_hash, u.avatar_url, u.bio, u.created_at, u.banned
		FROM password_resets pr
		JOIN users u ON u.id = pr.user_id
		WHERE pr.token = ? AND pr.expires_at > ?`,
		token, time.Now().UTC().Format(time.RFC3339),
	)
	return scanUser(row)
}

func (s *Store) DeletePasswordReset(token string) error {
	_, err := s.DB.Exec(`DELETE FROM password_resets WHERE token = ?`, token)
	return err
}

func (s *Store) ListPosts(currentUserID int, sort string, minLikes int, minComments int, period string, q string) ([]models.Post, error) {
	orderBy := "p.created_at DESC"
	switch sort {
	case "likes":
		orderBy = "likes_count DESC"
	case "comments":
		orderBy = "comments_count DESC"
	}

	periodFilter := ""
	switch period {
	case "today":
		periodFilter = "AND date(p.created_at) = date('now')"
	case "week":
		periodFilter = "AND p.created_at >= datetime('now', '-7 days')"
	case "month":
		periodFilter = "AND p.created_at >= datetime('now', '-30 days')"
	}

	args := []any{currentUserID}

	titleFilter := ""
	if q != "" {
		titleFilter = "AND lower(p.title) LIKE ?"
		args = append(args, "%"+strings.ToLower(q)+"%")
	}

	args = append(args, minLikes, minComments)

	query := fmt.Sprintf(`
		SELECT p.id, p.user_id, u.username, p.title, p.content, p.image_url, p.tags, p.created_at,
			COUNT(DISTINCT pl.user_id) AS likes_count,
			COUNT(DISTINCT c.id) AS comments_count,
			MAX(CASE WHEN pl.user_id = ? THEN 1 ELSE 0 END) AS liked
		FROM posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id
		LEFT JOIN comments c ON c.post_id = p.id
		WHERE 1=1 %s %s
		GROUP BY p.id
		HAVING likes_count >= ? AND comments_count >= ?
		ORDER BY %s`, periodFilter, titleFilter, orderBy)

	rows, err := s.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := scanPostRows(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *Store) UserPosts(userID int) ([]models.Post, error) {
	rows, err := s.DB.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.image_url, p.tags, p.created_at,
			COUNT(DISTINCT pl.user_id) AS likes_count,
			COUNT(DISTINCT c.id) AS comments_count,
			0 AS liked
		FROM posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id
		LEFT JOIN comments c ON c.post_id = p.id
		WHERE p.user_id = ?
		GROUP BY p.id
		ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := scanPostRows(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

func (s *Store) PostByID(id, currentUserID int) (*models.Post, error) {
	row := s.DB.QueryRow(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.image_url, p.tags, p.created_at,
			COUNT(DISTINCT pl.user_id) AS likes_count,
			COUNT(DISTINCT c.id) AS comments_count,
			MAX(CASE WHEN pl.user_id = ? THEN 1 ELSE 0 END) AS liked
		FROM posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id
		LEFT JOIN comments c ON c.post_id = p.id
		WHERE p.id = ?
		GROUP BY p.id`,
		currentUserID, id,
	)

	var post models.Post
	var liked int
	var tags string
	if err := row.Scan(&post.ID, &post.UserID, &post.Author, &post.Title, &post.Content, &post.ImageURL, &tags, &post.CreatedAt, &post.LikesCount, &post.CommentsCount, &liked); err != nil {
		return nil, err
	}
	post.Author = displayUsername(post.Author)
	post.Tags = splitTags(tags)
	post.Liked = liked == 1
	return &post, nil
}

func (s *Store) CreatePost(userID int, title, content, imageURL, tags string) (int, error) {
	result, err := s.DB.Exec(
		`INSERT INTO posts (user_id, title, content, image_url, tags) VALUES (?, ?, ?, ?, ?)`,
		userID, title, content, imageURL, tags,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	return int(id), err
}

func (s *Store) UpdatePost(id, userID int, title, content, imageURL, tags string) error {
	_, err := s.DB.Exec(
		`UPDATE posts SET title = ?, content = ?, image_url = ?, tags = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND user_id = ?`,
		title, content, imageURL, tags, id, userID,
	)
	return err
}

func (s *Store) DeletePost(id, userID int) error {
	_, err := s.DB.Exec(`DELETE FROM posts WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) ToggleLike(postID, userID int) (bool, int, error) {
	var exists int
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID).Scan(&exists)
	if err != nil {
		return false, 0, err
	}

	liked := exists == 0
	if liked {
		_, err = s.DB.Exec(`INSERT INTO post_likes (post_id, user_id) VALUES (?, ?)`, postID, userID)
	} else {
		_, err = s.DB.Exec(`DELETE FROM post_likes WHERE post_id = ? AND user_id = ?`, postID, userID)
	}
	if err != nil {
		return false, 0, err
	}

	var count int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM post_likes WHERE post_id = ?`, postID).Scan(&count); err != nil {
		return false, 0, err
	}
	return liked, count, nil
}

func (s *Store) CommentsByPost(postID int) ([]models.Comment, error) {
	rows, err := s.DB.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at
		FROM comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.post_id = ?
		ORDER BY c.created_at ASC`,
		postID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Author, &comment.Content, &comment.CreatedAt); err != nil {
			return nil, err
		}
		comment.Author = displayUsername(comment.Author)
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (s *Store) CreateComment(postID, userID int, content string) error {
	_, err := s.DB.Exec(`INSERT INTO comments (post_id, user_id, content) VALUES (?, ?, ?)`, postID, userID, content)
	return err
}

func (s *Store) DeleteComment(id, userID int) error {
	_, err := s.DB.Exec(`DELETE FROM comments WHERE id = ? AND user_id = ?`, id, userID)
	return err
}

func (s *Store) StatsForUser(userID int) (models.Stats, error) {
	var stats models.Stats
	err := s.DB.QueryRow(`SELECT COUNT(*) FROM posts WHERE user_id = ?`, userID).Scan(&stats.PostsCount)
	if err != nil {
		return stats, err
	}
	err = s.DB.QueryRow(`SELECT COUNT(*) FROM comments WHERE user_id = ?`, userID).Scan(&stats.CommentsCount)
	if err != nil {
		return stats, err
	}
	err = s.DB.QueryRow(
		`SELECT COUNT(*) FROM post_likes pl JOIN posts p ON p.id = pl.post_id WHERE p.user_id = ?`,
		userID,
	).Scan(&stats.LikesReceived)
	return stats, err
}

func scanUser(row *sql.Row) (*models.User, error) {
	var user models.User
	var banned int
	err := row.Scan(&user.ID, &user.FullName, &user.Username, &user.Email, &user.PasswordHash, &user.AvatarURL, &user.Bio, &user.CreatedAt, &banned)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	user.Banned = banned == 1
	user.Username = displayUsername(user.Username)
	return &user, nil
}

func scanPostRows(rows *sql.Rows) (models.Post, error) {
	var post models.Post
	var tags string
	var liked int
	err := rows.Scan(&post.ID, &post.UserID, &post.Author, &post.Title, &post.Content, &post.ImageURL, &tags, &post.CreatedAt, &post.LikesCount, &post.CommentsCount, &liked)
	post.Author = displayUsername(post.Author)
	post.Tags = splitTags(tags)
	post.Liked = liked == 1
	return post, err
}

func displayUsername(username string) string {
	username = strings.TrimSpace(username)
	if username == "" {
		return "membre"
	}
	if strings.Contains(username, "@") {
		parts := strings.Split(username, "@")
		if strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}
	return username
}

func splitTags(value string) []string {
	var tags []string
	for _, tag := range strings.Split(value, ",") {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			tags = append(tags, tag)
		}
	}
	return tags
}

// UpdateAvatar met à jour l'URL de l'avatar d'un utilisateur
func (s *Store) UpdateAvatar(userID int, avatarURL string) error {
	_, err := s.DB.Exec(`UPDATE users SET avatar_url = ? WHERE id = ?`, avatarURL, userID)
	return err
}

// UpdateComment modifie le contenu d'un commentaire (uniquement par son auteur)
func (s *Store) UpdateComment(id, userID int, content string) error {
	_, err := s.DB.Exec(
		`UPDATE comments SET content = ? WHERE id = ? AND user_id = ?`,
		content, id, userID,
	)
	return err
}

// SearchPostsByTitle retourne les posts dont le titre contient la query
func (s *Store) SearchPostsByTitle(query string, currentUserID int) ([]models.Post, error) {
	likeQuery := "%" + strings.ToLower(query) + "%"
	rows, err := s.DB.Query(
		`SELECT p.id, p.user_id, u.username, p.title, p.content, p.image_url, p.tags, p.created_at,
			COUNT(DISTINCT pl.user_id) AS likes_count,
			COUNT(DISTINCT c.id) AS comments_count,
			MAX(CASE WHEN pl.user_id = ? THEN 1 ELSE 0 END) AS liked
		FROM posts p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id
		LEFT JOIN comments c ON c.post_id = p.id
		WHERE lower(p.title) LIKE ?
		GROUP BY p.id
		ORDER BY p.created_at DESC`,
		currentUserID, likeQuery,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		post, err := scanPostRows(rows)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, rows.Err()
}

// AdminDeleteUser supprime un compte sans vérification de propriété
func (s *Store) AdminDeleteUser(id int) error {
	_, err := s.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// AdminToggleBan bascule le statut banni d'un utilisateur
func (s *Store) AdminToggleBan(id int) error {
	_, err := s.DB.Exec(`UPDATE users SET banned = CASE WHEN banned = 1 THEN 0 ELSE 1 END WHERE id = ?`, id)
	return err
}

// AdminDeletePost supprime un post sans vérification de propriété
func (s *Store) AdminDeletePost(id int) error {
	_, err := s.DB.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

// UserComments retourne tous les commentaires d'un utilisateur avec le titre du post associé
func (s *Store) UserComments(userID int) ([]models.Comment, error) {
	rows, err := s.DB.Query(
		`SELECT c.id, c.post_id, c.user_id, u.username, c.content, c.created_at, p.title
		FROM comments c
		JOIN users u ON u.id = c.user_id
		JOIN posts p ON p.id = c.post_id
		WHERE c.user_id = ?
		ORDER BY c.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		if err := rows.Scan(&comment.ID, &comment.PostID, &comment.UserID, &comment.Author, &comment.Content, &comment.CreatedAt, &comment.PostTitle); err != nil {
			return nil, err
		}
		comment.Author = displayUsername(comment.Author)
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

// ListUsersEnriched retourne les utilisateurs avec leurs stats complètes
func (s *Store) ListUsersEnriched() ([]models.UserEnriched, error) {
	rows, err := s.DB.Query(
		`SELECT u.id, u.full_name, u.username, u.email, u.avatar_url, u.bio,
			COUNT(DISTINCT p.id) AS posts_count,
			COUNT(DISTINCT c.id) AS comments_count,
			COUNT(DISTINCT pl.post_id) AS likes_received,
			u.created_at
		FROM users u
		LEFT JOIN posts p ON p.user_id = u.id
		LEFT JOIN comments c ON c.user_id = u.id
		LEFT JOIN post_likes pl ON pl.post_id IN (SELECT id FROM posts WHERE user_id = u.id)
		GROUP BY u.id
		ORDER BY posts_count DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserEnriched
	for rows.Next() {
		var u models.UserEnriched
		if err := rows.Scan(&u.ID, &u.FullName, &u.Username, &u.Email, &u.AvatarURL, &u.Bio,
			&u.PostsCount, &u.CommentsCount, &u.LikesReceived, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Username = displayUsername(u.Username)
		users = append(users, u)
	}
	return users, rows.Err()
}
package models

type User struct {
	ID           int
	FullName     string
	Username     string
	Email        string
	PasswordHash string
	AvatarURL    string
	Bio          string
	PostsCount   int
	CreatedAt    string
}

type Post struct {
	ID            int
	UserID        int
	Author        string
	Title         string
	Content       string
	ImageURL      string
	Tags          []string
	CreatedAt     string
	LikesCount    int
	CommentsCount int
	Liked         bool
}

type Comment struct {
	ID        int
	PostID    int
	UserID    int
	Author    string
	Content   string
	CreatedAt string
}

type Stats struct {
	PostsCount    int
	CommentsCount int
	LikesReceived int
}

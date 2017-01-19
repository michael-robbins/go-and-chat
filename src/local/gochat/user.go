package gochat

import (
	"math/rand"
	"time"
)

const (
	TOKEN_LENGTH = 12
	TOKEN_LETTERS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

type User struct {
	username string
	token string
	token_expiry time.Time
	salt string
	password_sha256 string
}

func (user *User) GenerateToken() {
	// Seed the RNG
	rand.Seed(time.Now().UnixNano())

	// Generate the byte array and fill it
	token := make([]byte, TOKEN_LENGTH)

	for i := range token {
		token[i] = TOKEN_LETTERS[rand.Intn(len(TOKEN_LETTERS))]
	}

	// Return the token (in string form) and the expiry for the token
	user.token = string(token)
	user.token_expiry = time.Now()
}

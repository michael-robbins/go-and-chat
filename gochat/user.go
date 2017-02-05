package gochat

import (
	"encoding/gob"
	"math/rand"
	"time"
)

const (
	TOKEN_LENGTH  = 12
	TOKEN_LETTERS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type User struct {
	Id              int    `db:"id"`
	Username        string `db:"username"`
	Salt            string `db:"salt"`
	Password_sha256 string `db:"password_sha256"`
	Deleted         bool `db:"deleted"`
	token           string
	token_expiry    time.Time
	encoder			*gob.Encoder
}

func (user *User) GetToken() string {
	if user.token == "" {
		user.generateToken()
	}

	return user.token
}

func (user *User) generateToken() {
	// Seed the RNG
	rand.Seed(time.Now().UnixNano())

	// Generate the byte array and fill it
	token := make([]byte, TOKEN_LENGTH)

	for i := range token {
		token[i] = TOKEN_LETTERS[rand.Intn(len(TOKEN_LETTERS))]
	}

	// Return the Token (in string form) and the expiry for the Token
	user.token = string(token)
	user.token_expiry = time.Now()
}

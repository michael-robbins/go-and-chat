package gochat

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	SALT_BYTES = 64
)

const (
	CREATE_USER_SQL = "INSERT INTO users (username, salt, password_sha256, deleted) VALUES (?, ?, ?, ?)"
	UPDATE_USERNAME_SQL = "UPDATE users SET username=? WHERE username=?"
	UPDATE_PASSWORD_SQL = "UPDATE users SET username=? WHERE username=?"
	DELETE_USER_SQL = "UPDATE users SET deleted=true WHERE username=?"
	GET_USER_SQL = "SELECT * FROM users WHERE username=?"
	USER_SCHEMA = `
	CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT,
		salt TEXT,
		password_sha256	TEXT,
		deleted BOOLEAN
	)`
)

type UserManager struct {
	storage		*StorageManager
	user_cache	map[string]*User
}

func NewUserManager(storage *StorageManager) *UserManager {
	return &UserManager{storage: storage}
}

func (manager *UserManager) InitialiseUserManager()     {}

func (manager *UserManager) PersistUser(user *User) (bool, error) {
	return true, nil
}

func (manager *UserManager) GetUser(username string) (*User, error) {
	// If the user has already been extracted from storage, just return them
	if user, ok := manager.user_cache[username]; ok {
		return user, nil
	}

	// Otherwise extract the user from storage, putting them into the cache as well
	var err error
	var user *User

	// TODO: Get user from storage manager

	if err != nil {
		return nil, err
	}

	manager.user_cache[user.Username] = user
	return user, nil
}

func (manager *UserManager) CreateUser(username string, password string) (*User, error) {
	// Generate a salt
	salt := make([]byte, SALT_BYTES)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("There was an error registering the user.")
	}

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	salted_hash := sha256.Sum256(append(salt, password_hash[:]...))

	user := User{
		Username:        username,
		salt:            hex.EncodeToString(salt),
		password_sha256: hex.EncodeToString(salted_hash[:]),
	}

	manager.PersistUser(&user)

	return &user, nil
}

func (manager *UserManager) AuthenticateUser(username string, password_sha256 string) (*User, error) {
	user, err := manager.GetUser(username)
	if err != nil {
		return nil, err
	}

	client_hash, err := hex.DecodeString(password_sha256)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users password hash.")
	}

	server_salt, err := hex.DecodeString(user.salt)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users server salt.")
	}

	server_hash, err := hex.DecodeString(user.password_sha256)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users server password hash.")
	}

	client_side_hash := sha256.Sum256(append(server_salt, client_hash...))
	server_side_hash := sha256.Sum256(append(server_salt, server_hash...))

	if client_side_hash == server_side_hash {
		return user, nil
	} else {
		return nil, errors.New("Invalid password")
	}

	return user, nil
}

func (manager *UserManager) TokenIsValid(token string) (bool, error) {
	// We can safely assert here that if the Token does not belong to a user in the cache, then the Token is invalid

	for _, user := range manager.user_cache {
		if user.token == token {
			if user.token_expiry.After(time.Now().Add(time.Hour * -24)) {
				// Token is valid and Token has not expired yet, this is a valid request
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	// User was not found or their Token doesn't exist
	return false, nil
}

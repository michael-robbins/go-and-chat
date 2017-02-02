package gochat

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"time"

	log "github.com/Sirupsen/logrus"
)

const (
	SALT_BYTES = 64
)

const (
	CREATE_USER_SQL     = "INSERT INTO users (username, salt, password_sha256, Deleted) VALUES (?, ?, ?, ?)"
	UPDATE_PASSWORD_SQL = "UPDATE users SET password_sha256=? WHERE username=?"
	DELETE_USER_SQL     = "UPDATE users SET deleted=true WHERE username=?"
	GET_USER_SQL        = "SELECT * FROM users WHERE username=? AND deleted=false"
	USER_SCHEMA         = `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		salt TEXT,
		password_sha256	TEXT,
		deleted BOOLEAN
	)`
)

type UserManager struct {
	storage     *StorageManager
	logger	    *log.Entry
	user_cache  map[string]*User
	token_cache map[string]*User
}

func NewUserManager(storage *StorageManager, logger *log.Entry) (*UserManager, error) {
	// Create the users table if it doesn't already exist
	_, err := storage.db.Exec(USER_SCHEMA)
	if err != nil {
		return &UserManager{}, err
	}

	return &UserManager{storage: storage, logger: logger}, nil
}

func (manager *UserManager) GetUser(username string) (*User, error) {
	// If the user has already been extracted from storage, just return them
	if user, ok := manager.user_cache[username]; ok {
		return user, nil
	}

	// Otherwise extract the user from storage, putting them into the cache as well
	var user *User

	if err := manager.storage.db.Get(user, GET_USER_SQL, username); err != nil {
		return &User{}, err
	}

	manager.user_cache[user.Username] = user

	// Generate a token for the user
	manager.token_cache[user.GetToken()] = user
	return user, nil
}

func hashPassword(password string) (string, string, error) {
	// Generate a salt
	salt := make([]byte, SALT_BYTES)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return "", "", err
	}

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	salted_hash := sha256.Sum256(append(salt, password_hash[:]...))

	return hex.EncodeToString(salt), hex.EncodeToString(salted_hash[:]), nil
}

func (manager *UserManager) CreateUser(username string, password string) (*User, error) {
	// Hash the password, generating a new salt as well
	salt, password, err := hashPassword(password)
	if err != nil {
		return &User{}, err
	}

	// Create the user
	if err = manager.storage.ExecOneRow(CREATE_USER_SQL, []interface{}{username, salt, password, false}); err != nil {
		return &User{}, err
	}

	return manager.GetUser(username)
}

func (manager *UserManager) AuthenticateUser(username string, password_sha256 string) (*User, error) {
	user, err := manager.GetUser(username)
	if err != nil {
		return &User{}, err
	}

	client_hash, err := hex.DecodeString(password_sha256)
	if err != nil {
		return &User{}, errors.New("Error decoding users password hash.")
	}

	server_salt, err := hex.DecodeString(user.salt)
	if err != nil {
		return &User{}, errors.New("Error decoding users server salt.")
	}

	server_hash, err := hex.DecodeString(user.password_sha256)
	if err != nil {
		return &User{}, errors.New("Error decoding users server password hash.")
	}

	client_side_hash := sha256.Sum256(append(server_salt, client_hash...))
	server_side_hash := sha256.Sum256(append(server_salt, server_hash...))

	if client_side_hash == server_side_hash {
		return user, nil
	} else {
		return &User{}, errors.New("Invalid password")
	}
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

func (manager *UserManager) UpdatePassword(username string, password string) error {
	// Hash the password, generating a new salt as well
	salt, password, err := hashPassword(password)
	if err != nil {
		return err
	}

	// Update the password of the user
	args := []interface{}{username, salt, password, false}
	if err = manager.storage.ExecOneRow(UPDATE_PASSWORD_SQL, args); err != nil {
		return err
	}

	// Delete them from the cache
	delete(manager.user_cache, username)

	// Fetch the updated user
	user, err := manager.GetUser(username)
	if err != nil {
		return err
	}

	// Add them back into the cache
	manager.user_cache[user.Username] = user

	return nil
}

func (manager *UserManager) DeleteUser(username string) error {
	// Mark the user as deleted
	if err := manager.storage.ExecOneRow(DELETE_USER_SQL, []interface{}{username}); err != nil {
		return err
	}

	// Remove the user from the cache
	delete(manager.user_cache, username)

	return nil
}

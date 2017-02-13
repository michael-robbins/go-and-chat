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
	UPDATE_PASSWORD_SQL = "UPDATE users SET salt=?, password_sha256=? WHERE username=?"
	DELETE_USER_SQL     = "UPDATE users SET deleted=true WHERE username=?"
	GET_USER_SQL        = "SELECT * FROM users WHERE username=? AND deleted=?"
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
	logger      *log.Entry
	user_cache  map[string]*ServerUser
	token_cache map[string]*ServerUser
}

func NewUserManager(storage *StorageManager, logger *log.Entry) (*UserManager, error) {
	// Create the users table if it doesn't already exist
	_, err := storage.db.Exec(USER_SCHEMA)
	if err != nil {
		return &UserManager{}, err
	}

	manager := UserManager{
		storage:     storage,
		logger:      logger,
		user_cache:  make(map[string]*ServerUser),
		token_cache: make(map[string]*ServerUser),
	}

	return &manager, nil
}

func (manager *UserManager) GetUser(username string) (*ServerUser, error) {
	// If the user has already been extracted from storage, just return them
	if user, ok := manager.user_cache[username]; ok {
		return user, nil
	}

	// Otherwise extract the user from storage, putting them into the cache as well
	var dbUser User

	sql := manager.storage.db.Rebind(GET_USER_SQL)
	if err := manager.storage.db.Get(&dbUser, sql, username, false); err != nil {
		manager.logger.Debug("Failed to get the user from the DB")
		manager.logger.Error(err)
		return &ServerUser{}, err
	}

	user := ServerUser{User: &dbUser}

	// Put the user in the cache
	manager.user_cache[dbUser.Username] = &user

	// Generate a token for the user and put it in the token cache
	manager.token_cache[user.GetToken()] = &user
	return &user, nil
}

func hashPassword(password string) (string, string, error) {
	// Generate a salt
	salt := make([]byte, SALT_BYTES)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return "", "", err
	}

	// Prepend the salt and hash again
	salted_hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt), hex.EncodeToString(salted_hash[:]), nil
}

func (manager *UserManager) CreateUser(username string, password string) error {
	// Hash the password, generating a new salt as well
	salt, salted_hash, err := hashPassword(password)
	if err != nil {
		return err
	}

	// Create the user
	sql := manager.storage.db.Rebind(CREATE_USER_SQL)
	return manager.storage.ExecOneRow(manager.storage.db.Exec(sql, username, salt, salted_hash, false))
}

func (manager *UserManager) AuthenticateUser(username string, password_sha256 string) (*ServerUser, error) {
	user, err := manager.GetUser(username)
	if err != nil {
		manager.logger.Error(err)
		return &ServerUser{}, errors.New("That user does not exist!")
	}

	server_salt, err := hex.DecodeString(user.User.Salt)
	if err != nil {
		return &ServerUser{}, errors.New("Error decoding users server salt.")
	}

	client_hash_bytes := sha256.Sum256(append(server_salt, []byte(password_sha256)...))
	client_hash_string := hex.EncodeToString(client_hash_bytes[:])

	if client_hash_string == user.User.Password_sha256 {
		return user, nil
	} else {
		return &ServerUser{}, errors.New("Invalid password!")
	}
}

func (manager *UserManager) TokenIsValid(token string) (bool, error) {
	// We can safely assert here that if the Token does not belong to a user in the cache, then the Token is invalid
	for _, user := range manager.user_cache {
		if user.token == token {
			if user.tokenExpiry.After(time.Now().Add(time.Hour * -24)) {
				// Token is valid and Token has not expired yet, this is a valid request
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	// ServerUser was not found or their Token doesn't exist
	return false, nil
}

func (manager *UserManager) UpdatePassword(username string, password string) error {
	// Hash the password, generating a new salt as well
	salt, password, err := hashPassword(password)
	if err != nil {
		return err
	}

	// Update the password of the user
	sql := manager.storage.db.Rebind(UPDATE_PASSWORD_SQL)
	if err := manager.storage.ExecOneRow(manager.storage.db.Exec(sql, salt, password, username)); err != nil {
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
	manager.user_cache[user.User.Username] = user

	return nil
}

func (manager *UserManager) DeleteUser(username string) error {
	// Remove the user from the cache
	delete(manager.user_cache, username)

	// Mark the user as deleted
	sql := manager.storage.db.Rebind(DELETE_USER_SQL)
	return manager.storage.ExecOneRow(manager.storage.db.Exec(sql, username))
}

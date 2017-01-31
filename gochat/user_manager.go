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
	UPDATE_PASSWORD_SQL = "UPDATE users SET username=? WHERE username=?"
	DELETE_USER_SQL = "UPDATE users SET deleted=true WHERE username=?"
	GET_USER_SQL = "SELECT salt, password_sha256 FROM users WHERE username=? AND deleted=false"
	USER_SCHEMA = `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		salt TEXT,
		password_sha256	TEXT,
		deleted BOOLEAN
	)`
)

type UserManager struct {
	storage		*StorageManager
	user_cache	map[string]*User
}

func NewUserManager(storage *StorageManager) (*UserManager, error) {
	// Create the users table if it doesn't already exist
	result, err := storage.db.Exec(USER_SCHEMA)
	if err != nil {
		return &UserManager{}, err
	}
	fmt.Println("Attempted to create users table!")
	fmt.Println(result)

	return &UserManager{storage: storage}, nil
}

func (manager *UserManager) PersistUser(user *User) (bool, error) {

	return true, nil
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

	user := User{
		Username:        username,
		salt:            salt,
		password_sha256: password,
	}

	result, err := manager.storage.db.Exec(CREATE_USER_SQL, user.Username, user.salt, user.password_sha256, false)
	if err != nil {
		return &User{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return &User{}, err
	}

	if affected != 1 {
		return &User{}, errors.New("We did not insert the user? We affected " + string(affected) + " rows")
	}

	// Add them into the cache as well
	manager.user_cache[user.Username] = &user

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

func (manager *UserManager) UpdatePassword(username string, password string) (*User, error) {
	// Hash the password, generating a new salt as well
	salt, password, err := hashPassword(password)
	if err != nil {
		return &User{}, err
	}

	user := User{
		Username:        username,
		salt:            salt,
		password_sha256: password,
	}

	result, err := manager.storage.db.Exec(UPDATE_PASSWORD_SQL, user.Username, user.salt, user.password_sha256, false)
	if err != nil {
		return &User{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return &User{}, err
	}

	if affected != 1 {
		return &User{}, errors.New("We did not update the user? We affected " + string(affected) + " rows")
	}

	// Add them into the cache as well
	manager.user_cache[user.Username] = &user

	return &user, nil
}

func (manager *UserManager) DeleteUser(username string) error {
	result, err := manager.storage.db.Exec(DELETE_USER_SQL, username)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if affected != 1 {
		return errors.New("We did not delete the user? We affected " + string(affected) + " rows")
	}

	// Remove them from the cache
	delete(manager.user_cache, username)

	return nil
}
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

type STORAGE_STRATEGY string

const (
	FILE     = STORAGE_STRATEGY("FILE")
	DATABASE = STORAGE_STRATEGY("DATABASE")
)

const (
	SALT_BYTES = 64
)

type UserManager struct {
	strategy   STORAGE_STRATEGY
	user_cache map[string]*User
}

func NewUserManager() *UserManager {
	return &UserManager{}
}

func (manager *UserManager) InitialiseFileStorage()     {}
func (manager *UserManager) InitialiseDatabaseStorage() {}

func (manager *UserManager) PersistUserToFileStorage(user *User) (bool, error) {
	return true, nil
}

func (manager *UserManager) PersistUserToDatabaseStorage(user *User) (bool, error) {
	return true, nil
}

func (manager *UserManager) PersistUser(user *User) (bool, error) {
	switch manager.strategy {
	case FILE:
		return manager.PersistUserToFileStorage(user)
	case DATABASE:
		return manager.PersistUserToDatabaseStorage(user)
	default:
		return false, errors.New("Unable to determine storage stratergy")
	}
}

func (manager *UserManager) GetUserFromFileStorage(username string) (*User, error) {
	return nil, nil
}

func (manager *UserManager) GetUserFromDatabaseStorage(username string) (*User, error) {
	return nil, nil
}

func (manager *UserManager) GetUser(username string) (*User, error) {
	// If the user has already been extracted from storage, just return them
	if user, ok := manager.user_cache[username]; ok {
		return user, nil
	}

	// Otherwise extract the user from storage, putting them into the cache as well
	var err error
	var user *User

	switch manager.strategy {
	case DATABASE:
		user, err = manager.GetUserFromDatabaseStorage(username)
	case FILE:
		user, err = manager.GetUserFromFileStorage(username)
	default:
		return nil, errors.New("Unable to determine user manager storage stratergy?")
	}

	if err != nil {
		return nil, err
	}

	manager.user_cache[user.username] = user
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
		username:        username,
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

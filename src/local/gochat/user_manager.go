package gochat

import (
	"errors"
	"time"
)

type STORAGE_STRATEGY string

const (
	FILE     = STORAGE_STRATEGY("FILE")
	DATABASE = STORAGE_STRATEGY("DATABASE")
)

type UserManager struct {
	strategy STORAGE_STRATEGY
	user_cache map[string]*User
}

func NewUserManager() *UserManager {
	return &UserManager{}
}

func (manager *UserManager) InitialiseFileStorage() {}
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
		return nil, nil
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

func (manager *UserManager) TokenIsValid(token string) (bool, error) {
	// We can safely assert here that if the token does not belong to a user in the cache, then the token is invalid

	for _, user := range manager.user_cache {
		if user.token == token {
			if user.token_expiry.After(time.Now().Add(time.Hour * -24)) {
				// Token is valid and token has not expired yet, this is a valid request
				return true, nil
			} else {
				return false, nil
			}
		}
	}

	// User was not found or their token doesn't exist
	return false, nil
}
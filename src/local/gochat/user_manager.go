package gochat

type STORAGE_STRATEGY string

const (
	FILE     = STORAGE_STRATEGY("FILE")
	DATABASE = STORAGE_STRATEGY("DATABASE")
)

type UserManager struct {
	strategy STORAGE_STRATEGY
}

func NewUserManager() *UserManager {
	return &UserManager{}
}

func (manager *UserManager) InitialiseFileStorage() {}
func (manager *UserManager) InitialiseDatabaseStorage() {}

func (manager *UserManager) CreateUserFromFileStorage(user User) (bool, error) {
	return nil, nil
}

func (manager *UserManager) CreateUserFromDatabaseStorage(user User) (bool, error) {
	return nil, nil
}

func (manager *UserManager) CreateUser(user User) (bool, error) {
	switch manager.strategy {
	case FILE:
		return manager.CreateUserFromFileStorage(user)
	case DATABASE:
		return manager.CreateUserFromDatabaseStorage(user)
	default:
		return nil, nil
	}
}

func (manager *UserManager) GetUserFromFileStorage(username string) (User, error) {
	return nil, nil
}

func (manager *UserManager) GetUserFromDatabaseStorage(username string) (User, error) {
	return nil, nil
}

func (manager *UserManager) GetUser(username string) (User, error) {
	switch manager.strategy {
	case DATABASE:
		return manager.GetUserFromDatabaseStorage(username)
	case FILE:
		return manager.GetUserFromFileStorage(username)
	default:
		return nil, nil
	}
}
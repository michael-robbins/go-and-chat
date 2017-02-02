package gochat

import (
	"errors"
	"fmt"
)

const (
	CREATE_ROOM_SQL = "INSERT INTO rooms (name, capacity, closed) VALUES (?, ?, ?)"
	DELETE_ROOM_SQL = "DELETE FROM rooms WHERE name=?"
	GET_ROOM_SQL    = "SELECT * FROM rooms WHERE name=?"
	ROOM_SCHEMA     = `
	CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		capacity INTEGER,
		closed BOOLEAN
	)`

	CREATE_ROOM_USER_SQL  = "INSERT INTO room_users (room_id, user_id) VALUES (?, ?)"
	DELETE_ROOM_USER_SQL  = "DELETE FROM room_users WHERE room_id=? AND user_id=?"
	DELETE_ROOMS_USER_SQL = "DELETE FROM room_users WHERE user_id=?"
	DELETE_ROOM_USERS_SQL = "DELETE FROM room_users WHERE room_id=?"
	GET_ROOM_USERS_SQL    = "SELECT u.* FROM users AS u JOIN room_users AS ru ON u.user_id = ru.user_id WHERE ru.room_id=?"
	GET_USER_ROOMS_SQL    = "SELECT room_id FROM room_users WHERE user_id=?"
	ROOM_USERS_SCHEMA     = `
	CREATE TABLE IF NOT EXISTS room_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id INTEGER,
		user_id INTEGER
	)`
)

type RoomManager struct {
	storage    *StorageManager
	room_cache map[string]*Room
}

func NewRoomManager(storage *StorageManager) (*RoomManager, error) {
	// Create the rooms table if it doesn't already exist
	result, err := storage.db.Exec(ROOM_SCHEMA)
	if err != nil {
		return &RoomManager{}, err
	}
	fmt.Println("Attempted to create rooms table!")
	fmt.Println(result)

	// Create the room_users table if it doesn't already exist
	result, err = storage.db.Exec(ROOM_USERS_SCHEMA)
	if err != nil {
		return &RoomManager{}, err
	}
	fmt.Println("Attempted to create room_users table!")
	fmt.Println(result)

	return &RoomManager{storage: storage}, nil
}

func (manager *RoomManager) GetRoom(name string) (*Room, error) {
	// If the Room has already been extracted from storage, just return them
	if room, ok := manager.room_cache[name]; ok {
		if room.Closed {
			return nil, errors.New("Room is closed.")
		}

		return room, nil
	}

	// Otherwise extract the Room from storage, putting it into the cache as well
	var room *Room

	if err := manager.storage.db.Get(room, GET_ROOM_SQL, name); err != nil {
		return &Room{}, err
	}

	// Get the users that are in the room
	users := []User{}
	if err := manager.storage.db.Select(users, GET_ROOM_USERS_SQL, room.Id); err != nil {
		return &Room{}, err
	}

	for _, user := range users {
		manager.AddUserToRoom(room, &user)
	}

	// Add the Room to the cache regardless of if it's closed or not
	manager.room_cache[name] = room

	if room.Closed {
		return nil, errors.New("Room is closed.")
	}

	return room, nil
}

func (manager *RoomManager) CreateRoom(name string, capacity int) (*Room, error) {
	if err := manager.storage.ExecOneRow(CREATE_ROOM_SQL, []interface{}{name, capacity, false}); err != nil {
		return &Room{}, err
	}

	room, err := manager.GetRoom(name)
	if err != nil {
		return &Room{}, err
	}

	// Add them into the cache as well
	manager.room_cache[room.Name] = room

	return room, nil
}

func (manager *RoomManager) CloseRoom(name string) (*Room, error) {
	room, err := manager.GetRoom(name)
	if err != nil {
		return &Room{}, err
	}

	// Mark the room as deleted
	if err = manager.storage.ExecOneRow(DELETE_ROOM_SQL, []interface{}{name}); err != nil {
		return &Room{}, err
	}

	// Remove all users from the room
	if err = manager.storage.ExecZeroOrMoreRows(DELETE_ROOM_USERS_SQL, []interface{}{room.Id}); err != nil {
		return &Room{}, err
	}

	// Remove the room from the cache
	delete(manager.room_cache, name)

	return room, nil
}

func (manager *RoomManager) AddUserToRoom(room *Room, user *User) error {
	room.AddUser(user)

	if err := manager.storage.ExecOneRow(CREATE_ROOM_USER_SQL, []interface{}{room.Id, user.Id}); err != nil {
		return err
	}

	return nil
}

func (manager *RoomManager) RemoveUserFromRoom(room *Room, user *User) error {
	room.RemoveUser(user)

	if err := manager.storage.ExecOneRow(DELETE_ROOM_USER_SQL, []interface{}{room.Id, user.Id}); err != nil {
		return err
	}

	return nil
}

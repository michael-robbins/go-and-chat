package gochat

import (
	"errors"
	"fmt"
)

const (
	CREATE_ROOM_SQL = "INSERT INTO rooms (name, capacity, closed) VALUES (?, ?, ?)"
	UPDATE_ROOM_SQL = "UPDATE rooms SET name=? WHERE name=?"
	DELETE_ROOM_SQL = "DELETE FROM rooms WHERE name=?"
	GET_ROOM_SQL = "SELECT * FROM rooms WHERE name=?"
	ROOM_SCHEMA = `
	CREATE TABLE IF NOT EXISTS rooms (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		capacity INTEGER,
		closed BOOLEAN
	)`

	CREATE_ROOM_USER_SQL = "INSERT INTO room_users (room_id, user_id) VALUES (?, ?)"
	DELETE_ROOM_USER_SQL = "DELETE FROM room_users WHERE room_id=? AND user_id=?"
	GET_ROOM_USERS_SQL = "SELECT user_id FROM room_users WHERE room_id=?"
	GET_USER_ROOMS_SQL = "SELECT room_id FROM room_users WHERE user_id=?"
	ROOM_USERS_SCHEMA = `
	CREATE TABLE IF NOT EXISTS room_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		room_id INTEGER,
		user_id INTEGER
	)`

)

type RoomManager struct {
	storage		*StorageManager
	room_cache	map[string]*Room
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

func (manager *RoomManager) PersistRoom(room *Room) (bool, error) {
	return true, nil
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
	var err error
	var room *Room

	// TODO: Get room from storage manager

	if err != nil {
		return nil, err
	}

	// Add the Room to the cache regardless of if it's closed or not
	manager.room_cache[name] = room

	if room.Closed {
		return nil, errors.New("Room is closed.")
	}

	return room, nil
}

func (manager *RoomManager) CreateRoom(name string, capacity int) (*Room, error) {
	room := Room{
		Name:     name,
		Capacity: capacity,
	}

	// Ensure the Room is stored and not just in memory
	manager.PersistRoom(&room)

	return &room, nil
}

func (manager *RoomManager) CloseRoom(name string) (bool, error) {
	if room, ok := manager.room_cache[name]; ok {
		// Mark the Room as closed
		room.Closed = true
		manager.PersistRoom(room)

		return true, nil
	}

	return false, errors.New("Room doesn't exist.")
}

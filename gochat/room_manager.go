package gochat

import (
	"errors"
	"fmt"
)

const (
	CREATE_ROOM_SQL = "INSERT INTO rooms (name, capacity, closed) VALUES (?, ?, ?)"
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
	DELETE_ROOMS_USER_SQL = "DELETE FROM room_users WHERE user_id=?"
	DELETE_ROOM_USERS_SQL = "DELETE FROM room_users WHERE room_id=?"
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

	// TODO: Get room from storage manager
	if err := manager.storage.db.Get(room, GET_ROOM_SQL, name); err != nil {
		return &Room{}, err
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

	result, err := manager.storage.db.Exec(CREATE_ROOM_SQL, room.Name, room.Capacity, false)
	if err != nil {
		return &Room{}, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return &Room{}, err
	}

	if affected != 1 {
		return &Room{}, errors.New("We did not create the Room? We affected " + string(affected) + " rows")
	}

	// Add them into the cache as well
	manager.room_cache[room.Name] = &room

	return &room, nil
}

func (manager *RoomManager) CloseRoom(name string) (*Room, error) {
	room, err := manager.GetRoom(name)
	if err != nil {
		return &Room{}, err
	}

	// Mark the room as deleted
	err = manager.storage.Exec(
		DELETE_ROOM_SQL,
		func(affected int64) bool {return affected == 1},
		[]interface{}{name})

	if err != nil {
		return &Room{}, err
	}

	// Remove the room from the cache
	delete(manager.room_cache, name)

	return room, nil
}

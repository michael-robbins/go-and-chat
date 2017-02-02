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

	// Remove the room from the cache
	delete(manager.room_cache, name)

	return room, nil
}
package gochat

import (
	"errors"

	log "github.com/Sirupsen/logrus"
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
	logger	   *log.Entry
	room_cache map[string]*Room
}

func NewRoomManager(storage *StorageManager, logger *log.Entry) (*RoomManager, error) {
	// Create the rooms table if it doesn't already exist
	_, err := storage.db.Exec(ROOM_SCHEMA)
	if err != nil {
		return &RoomManager{}, err
	}

	return &RoomManager{storage: storage, logger: logger}, nil
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
	sql := manager.storage.db.Rebind(CREATE_ROOM_SQL)
	if err := manager.storage.ExecOneRow(manager.storage.db.Exec(sql, name, capacity, false)); err != nil {
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
	// Attempt to get the room first, failing if it doesn't exist
	room, err := manager.GetRoom(name)
	if err != nil {
		return &Room{}, err
	}

	// Remove the room from the cache
	delete(manager.room_cache, name)

	// Mark the room as deleted
	sql := manager.storage.db.Rebind(DELETE_ROOM_SQL)
	return room, manager.storage.ExecOneRow(manager.storage.db.Exec(sql, name))
}

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
	roomCache map[string]*ServerRoom
}

func NewRoomManager(storage *StorageManager, logger *log.Entry) (*RoomManager, error) {
	// Create the rooms table if it doesn't already exist
	_, err := storage.db.Exec(ROOM_SCHEMA)
	if err != nil {
		return &RoomManager{}, err
	}

	manager := RoomManager{
		storage: storage,
		logger: logger,
		roomCache: make(map[string]*ServerRoom),
	}

	return &manager, nil
}

func (manager *RoomManager) GetRoom(name string) (*ServerRoom, error) {
	// If the Room has already been extracted from storage, just return them
	if room, ok := manager.roomCache[name]; ok {
		if room.Room.Closed {
			return nil, errors.New("Room is closed.")
		}

		return room, nil
	}

	// Otherwise extract the Room from storage, putting it into the cache as well
	var dbRoom Room

	if err := manager.storage.db.Get(&dbRoom, GET_ROOM_SQL, name); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return &ServerRoom{}, errors.New("Room doesn't exist")
		}

		manager.logger.Error(err)
		return &ServerRoom{}, err
	}

	room := ServerRoom{Room: &dbRoom}

	// Add the Room to the cache regardless of if it's closed or not
	manager.roomCache[name] = &room

	if room.Room.Closed {
		return &ServerRoom{}, errors.New("Room is closed.")
	}

	return &room, nil
}

func (manager *RoomManager) GetRoomNames() []string {
	rooms := make([]string, len(manager.roomCache))

	// key is the name of the room here
	for key := range manager.roomCache {
		rooms = append(rooms, key)
	}

	if len(rooms) == 0 {
		rooms = append(rooms, "There are no rooms!")
	}

	return rooms
}

func (manager *RoomManager) CreateRoom(name string, capacity int) (*ServerRoom, error) {
	sql := manager.storage.db.Rebind(CREATE_ROOM_SQL)
	if err := manager.storage.ExecOneRow(manager.storage.db.Exec(sql, name, capacity, false)); err != nil {
		manager.logger.Debug("Failed to run CREATE_ROOM_SQL")
		return &ServerRoom{}, err
	}

	room, err := manager.GetRoom(name)
	if err != nil {
		manager.logger.Debug("Failed to get room after creation")
		manager.logger.Error(err)
		return &ServerRoom{}, err
	}

	// Add them into the cache as well
	manager.roomCache[room.Room.Name] = room

	return room, nil
}

func (manager *RoomManager) CloseRoom(name string) (*ServerRoom, error) {
	// Attempt to get the room first, failing if it doesn't exist
	room, err := manager.GetRoom(name)
	if err != nil {
		return &ServerRoom{}, err
	}

	// Remove the room from the cache
	delete(manager.roomCache, name)

	// Mark the room as closed on the object
	room.Room.Closed = true

	// Mark the room as closed in the db
	sql := manager.storage.db.Rebind(DELETE_ROOM_SQL)
	return room, manager.storage.ExecOneRow(manager.storage.db.Exec(sql, name))
}

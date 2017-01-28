package gochat

import (
	"errors"
)

const (
	ROOM_UNKNOWN_STORAGE_STRATEGY = "Unable to determine room manager storage stratergy?"
	ROOM_DOES_NOT_EXIST_MESSAGE   = "Room doesn't exist."
	ROOM_CLOSED_MESSAGE           = "Room is closed."
)

type RoomManager struct {
	strategy   STORAGE_STRATEGY
	room_cache map[string]*ChatRoom
}

func NewRoomManager() *RoomManager {
	return &RoomManager{}
}

func (manager *RoomManager) InitialiseFileStorage()     {}
func (manager *RoomManager) InitialiseDatabaseStorage() {}

func (manager *RoomManager) PersistRoomToFileStorage(room *ChatRoom) (bool, error) {
	return true, nil
}

func (manager *RoomManager) PersistRoomToDatabaseStorage(room *ChatRoom) (bool, error) {
	return true, nil
}

func (manager *RoomManager) PersistRoom(room *ChatRoom) (bool, error) {
	switch manager.strategy {
	case FILE:
		return manager.PersistRoomToFileStorage(room)
	case DATABASE:
		return manager.PersistRoomToDatabaseStorage(room)
	default:
		return false, errors.New("Unable to determine storage stratergy")
	}
}

func (manager *RoomManager) GetRoomFromFileStorage(room_name string) (*ChatRoom, error) {
	return nil, nil
}

func (manager *RoomManager) GetRoomFromDatabaseStorage(room_name string) (*ChatRoom, error) {
	return nil, nil
}

func (manager *RoomManager) GetRoom(name string) (*ChatRoom, error) {
	// If the room has already been extracted from storage, just return them
	if room, ok := manager.room_cache[name]; ok {
		if room.closed {
			return nil, errors.New(ROOM_CLOSED_MESSAGE)
		}

		return room, nil
	}

	// Otherwise extract the room from storage, putting it into the cache as well
	var err error
	var room *ChatRoom

	switch manager.strategy {
	case DATABASE:
		room, err = manager.GetRoomFromDatabaseStorage(name)
	case FILE:
		room, err = manager.GetRoomFromFileStorage(name)
	default:
		return nil, errors.New(ROOM_UNKNOWN_STORAGE_STRATEGY)
	}

	if err != nil {
		return nil, err
	}

	// Add the room to the cache regardless of if it's closed or not
	manager.room_cache[name] = room

	if room.closed {
		return nil, errors.New(ROOM_CLOSED_MESSAGE)
	}

	return room, nil
}

func (manager *RoomManager) CreateRoom(name string, capacity int) (*ChatRoom, error) {
	room := ChatRoom{
		name:     name,
		capacity: capacity,
	}

	// Ensure the room is stored and not just in memory
	manager.PersistRoom(&room)

	return &room, nil
}

func (manager *RoomManager) CloseRoom(name string) (bool, error) {
	if room, ok := manager.room_cache[name]; ok {
		// Mark the room as closed
		room.closed = true
		manager.PersistRoom(room)

		return true, nil
	}

	return false, errors.New(ROOM_DOES_NOT_EXIST_MESSAGE)
}

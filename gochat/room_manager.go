package gochat

import (
	"errors"
)

type RoomManager struct {
	config		ServerConfig
	room_cache map[string]*ChatRoom
}

func NewRoomManager(config ServerConfig) *RoomManager {
	return &RoomManager{config: config}
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
	switch manager.config.Method {
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
	// If the Room has already been extracted from storage, just return them
	if room, ok := manager.room_cache[name]; ok {
		if room.closed {
			return nil, errors.New("Room is closed.")
		}

		return room, nil
	}

	// Otherwise extract the Room from storage, putting it into the cache as well
	var err error
	var room *ChatRoom

	switch manager.config.Method {
	case DATABASE:
		room, err = manager.GetRoomFromDatabaseStorage(name)
	case FILE:
		room, err = manager.GetRoomFromFileStorage(name)
	default:
		return nil, errors.New("Unable to determine Room manager storage stratergy?")
	}

	if err != nil {
		return nil, err
	}

	// Add the Room to the cache regardless of if it's closed or not
	manager.room_cache[name] = room

	if room.closed {
		return nil, errors.New("Room is closed.")
	}

	return room, nil
}

func (manager *RoomManager) CreateRoom(name string, capacity int) (*ChatRoom, error) {
	room := ChatRoom{
		name:     name,
		capacity: capacity,
	}

	// Ensure the Room is stored and not just in memory
	manager.PersistRoom(&room)

	return &room, nil
}

func (manager *RoomManager) CloseRoom(name string) (bool, error) {
	if room, ok := manager.room_cache[name]; ok {
		// Mark the Room as closed
		room.closed = true
		manager.PersistRoom(room)

		return true, nil
	}

	return false, errors.New("Room doesn't exist.")
}

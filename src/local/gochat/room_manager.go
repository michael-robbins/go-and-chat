package gochat

import (
	"errors"
)

type RoomManager struct {
	strategy STORAGE_STRATEGY
	room_cache map[string]*ChatRoom
}

func NewRoomManager() *RoomManager {
	return &RoomManager{}
}

func (manager *RoomManager) InitialiseFileStorage() {}
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
		return nil, nil
	}
}

func (manager *RoomManager) GetRoomFromFileStorage(room_name string) (*ChatRoom, error) {
	return nil, nil
}

func (manager *RoomManager) GetRoomFromDatabaseStorage(room_name string) (*ChatRoom, error) {
	return nil, nil
}

func (manager *RoomManager) GetRoom(room_name string) (*ChatRoom, error) {
	// If the room has already been extracted from storage, just return them
	if room, ok := manager.room_cache[room_name]; ok {
		return room, nil
	}

	// Otherwise extract the room from storage, putting it into the cache as well
	var err error
	var room *ChatRoom

	switch manager.strategy {
	case DATABASE:
		room, err = manager.GetRoomFromDatabaseStorage(room_name)
	case FILE:
		room, err = manager.GetRoomFromFileStorage(room_name)
	default:
		return nil, errors.New("Unable to determine room manager storage stratergy?")
	}

	if err != nil {
		return nil, err
	}

	manager.room_cache[room_name] = room
	return room, nil
}
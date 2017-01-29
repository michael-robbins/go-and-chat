package gochat

import (
	"errors"
)

type RoomManager struct {
	storage		*StorageManager
	room_cache	map[string]*ChatRoom
}

func NewRoomManager(storage *StorageManager) *RoomManager {
	return &RoomManager{storage: storage}
}

func (manager *RoomManager) InitialiseRoomManager() {}

func (manager *RoomManager) PersistRoom(room *ChatRoom) (bool, error) {
	return true, nil
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

	// TODO: Get room from storage manager

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

package gochat

import (
	"errors"
	"fmt"
)

type Room struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	Capacity int    `db:"capacity"`
	Closed   bool   `db:"closed"`
}

type ServerRoom struct {
	Room  *Room
	users []*ServerUser
}

func (room *ServerRoom) String() string {
	if room.Room == nil {
		return "Unconfigured Room"
	}

	return fmt.Sprintf("%s", room.Room.Name)
}

func (room *ServerRoom) AddUser(user *ServerUser) error {
	room.users = append(room.users, user)
	return nil
}

func removeUserFromList(user *ServerUser, array []*ServerUser) ([]*ServerUser, error) {
	index := -1
	for i, room_user := range array {
		if room_user == user {
			index = i
		}
	}

	if index == -1 {
		return nil, errors.New("ServerUser does not exist in this list")
	}

	return append(array[:index], array[index+1:]...), nil
}

func (room *ServerRoom) RemoveUser(user *ServerUser) error {
	array, err := removeUserFromList(user, room.users)
	if err != nil {
		return err
	}

	room.users = array

	return nil
}

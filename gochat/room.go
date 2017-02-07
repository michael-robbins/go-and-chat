package gochat

import (
	"errors"
	"fmt"
)

type Room struct {
	Id       int    `db:"id"`
	Name     string `db:"name"`
	Capacity int  `db:"capacity"`
	Closed   bool `db:"closed"`
}

type ServerRoom struct {
	Room	*Room
	users	[]*ServerUser
}

func (room *ServerRoom) String() string {
	if room.Room == nil {
		return "Unconfigured Room"
	}

	return fmt.Sprintf("%s", room.Room.Name)
}


func (room *ServerRoom) AddUser(user *ServerUser) {
	room.users = append(room.users, user)
}

func removeUserFromList(user *ServerUser, array []*ServerUser) error {
	index := -1
	for i, room_user := range array {
		if room_user == user {
			index = i
		}
	}

	if index == -1 {
		return errors.New("ServerUser does not exist in this list")
	}

	array = append(array[:index], array[index+1:]...)

	return nil
}

func (room *ServerRoom) RemoveUser(user *ServerUser) error {
	if err := removeUserFromList(user, room.users); err != nil {
		return err
	}

	// Capacity is only for normal users, super users do not count towards the Capacity of a Room
	room.Room.Capacity = room.Room.Capacity - 1

	return nil
}

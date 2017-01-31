package gochat

import (
	"errors"
)

type Room struct {
	Name       string	`db:"name"`
	users      []*User
	superUsers []*User
	Capacity   int		`db:"capacity"`
	Closed     bool		`db:"closed"`
}

func (room *Room) AddUser(user *User, isSuperUser bool) error {
	if isSuperUser {
		// We skip the Capacity check as the user is a super user
		room.users = append(room.users, user)
		room.superUsers = append(room.superUsers, user)
	} else {
		if len(room.users) > room.Capacity {
			return errors.New("Room is at Capacity!")
		}

		room.users = append(room.users, user)
	}

	return nil
}

func removeUserFromList(user *User, array []*User) error {
	index := -1
	for i, room_user := range array {
		if room_user == user {
			index = i
		}
	}

	if index == -1 {
		return errors.New("User does not exist in this list")
	}

	array = append(array[:index], array[index+1:]...)

	return nil
}

func (room *Room) RemoveUser(user *User) error {
	if err := removeUserFromList(user, room.users); err != nil {
		return err
	}

    // Capacity is only for normal users, super users do not count towards the Capacity of a Room
	room.Capacity = room.Capacity - 1

	if err := removeUserFromList(user, room.superUsers); err != nil {
		return err
	}

	return nil
}

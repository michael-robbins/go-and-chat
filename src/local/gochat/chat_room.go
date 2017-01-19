package gochat

import "errors"

type ChatRoom struct {
	name string
	users []User
	superUsers []User
	capacity int
}

func (room *ChatRoom) AddUser(user User, isSuperUser bool) {
	room.users = append(room.users, user)

	if isSuperUser {
		room.superUsers = append(room.superUsers, user)
	}
}

func removeUserFromList(user User, array []User) error {
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

func (room *ChatRoom) RemoveUser(user User) error {
	for _, list := range [][]User{room.users, room.superUsers} {
		if err := removeUserFromList(user, list); err != nil {
			return err
		}
	}

	return nil
}

func (room *ChatRoom) SendMessage(message string) {}
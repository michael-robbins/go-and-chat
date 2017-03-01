package gochat

import (
	"fmt"
	"time"
)

type RoomMessage struct {
	Username  string `db:"username"`
	Message   string `db:"message"`
	Timestamp int64  `db:"epoch_timestamp"`
}

type ServerRoomMessage struct {
	RoomMessage *RoomMessage
	Room        *ServerRoom
	User        *ServerUser
	Timestamp   time.Time
}

func (message *ServerRoomMessage) String() string {
	if message.RoomMessage == nil {
		return "Unconfigured RoomMessage"
	}

	return fmt.Sprintf("[%s] %s", message.User, message.RoomMessage.Message)
}

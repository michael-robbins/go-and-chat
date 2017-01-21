package gochat

import (
	"encoding/gob"
	"net"
)

type COMMAND string

const (
	AUTHENTICATE    = COMMAND("AUTHENTICATE")
	TOKEN           = COMMAND("TOKEN")
	LIST_ROOMS		= COMMAND("LIST_ROOMS")
	JOIN_ROOM		= COMMAND("JOIN_ROOM")
	LEAVE_ROOM		= COMMAND("LEAVE_ROOM")
	CREATE_ROOM		= COMMAND("CREATE_ROOM")
	CLOSE_ROOM		= COMMAND("CLOSE_ROOM")
	SEND_MSG    = COMMAND("SEND_MSG")
	RECV_MSG    = COMMAND("RECV_MSG")
)

type Message struct {
	command COMMAND
	contents interface{}
}

type AuthenticateMessage struct {
	username string
	password_hash string
}

type TokenMessage struct {
	username string
	token string
}

type TextMessage struct {
	username string
	room string
	text string
}

type SendTextMessage struct {
	token string
	message TextMessage
}

type RecvTextMessage struct {
	message TextMessage
}

type ListRoomsMessage struct {
	rooms []string
}

type JoinRoomMessage struct {
	username string
	room string
	isSuperUser bool
	token string
}

type LeaveRoomMessage struct {
	username string
	room string
	token string
}

type CreateRoomMessage struct {
	room string
	capacity int
	token string
}

type CloseRoomMessage struct {
	room string
	token string
}

func SendRemoteCommand(connection net.Conn, message Message) error {
	encoder := gob.NewEncoder(connection)
	return encoder.Encode(message)
}

func BuildMessage(message_type COMMAND, contents interface{}) Message {
	return Message{command: message_type, contents: contents}
}

package gochat

import (
	"encoding/gob"
	"net"
)

type COMMAND string

const (
	REGISTER	 = COMMAND("Register")
	AUTHENTICATE = COMMAND("Authenticate")
	TOKEN        = COMMAND("Token")
	LIST_ROOMS   = COMMAND("List Rooms")
	JOIN_ROOM    = COMMAND("Join Room")
	LEAVE_ROOM   = COMMAND("Leave Room")
	CREATE_ROOM  = COMMAND("Create Room")
	CLOSE_ROOM   = COMMAND("Close Room")
	SEND_MSG     = COMMAND("Send Message")
	RECV_MSG     = COMMAND("Receive Message")
)

type Message struct {
	Command  COMMAND
	Contents interface{}
}

type RegisterMessage struct {
	Username	 string
	PasswordHash string
}

type AuthenticateMessage struct {
	Username     string
	PasswordHash string
}

type TokenMessage struct {
	Username string
	Token    string
}

type TextMessage struct {
	Username string
	Room     string
	Text     string
}

type SendTextMessage struct {
	Token   string
	Message TextMessage
}

type RecvTextMessage struct {
	Message TextMessage
}

type ListRoomsMessage struct {
	Rooms []string
}

type JoinRoomMessage struct {
	Username    string
	Room        string
	IsSuperUser bool
	Token       string
}

type LeaveRoomMessage struct {
	Username string
	Room     string
	Token    string
}

type CreateRoomMessage struct {
	Room     string
	Capacity int
	Token    string
}

type CloseRoomMessage struct {
	Room  string
	Token string
}

func RegisterStructs() {
	// Register all the various subtypes of messages so gob can encode/decode them correctly
	gob.Register(RegisterMessage{})
	gob.Register(AuthenticateMessage{})
	gob.Register(TokenMessage{})
	gob.Register(TextMessage{})
	gob.Register(SendTextMessage{})
	gob.Register(RecvTextMessage{})
	gob.Register(ListRoomsMessage{})
	gob.Register(JoinRoomMessage{})
	gob.Register(LeaveRoomMessage{})
	gob.Register(CreateRoomMessage{})
	gob.Register(CloseRoomMessage{})
}

func SendRemoteCommand(connection net.Conn, message Message) error {
	encoder := gob.NewEncoder(connection)
	return encoder.Encode(message)
}

func BuildMessage(message_type COMMAND, contents interface{}) Message {
	return Message{Command: message_type, Contents: contents}
}

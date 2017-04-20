package gochat

import (
	"encoding/gob"
	"time"
)

type COMMAND string

const (
	REGISTER     = COMMAND("Register")
	AUTHENTICATE = COMMAND("Authenticate")
	TOKEN        = COMMAND("Token")
	LIST_ROOMS   = COMMAND("List Rooms")
	JOIN_ROOM    = COMMAND("Join Room")
	LEAVE_ROOM   = COMMAND("Leave Room")
	CREATE_ROOM  = COMMAND("Create Room")
	CLOSE_ROOM   = COMMAND("Close Room")
	SEND_MSG     = COMMAND("Send Message")
	RECV_MSG     = COMMAND("Receive Message")
	POP_MSGS     = COMMAND("Populate Messages")
)

type STATUS string

const (
	SUCCESS = STATUS("Request was successfull")
	FAILURE = STATUS("Request failed")
)

type Message struct {
	Command  COMMAND
	Contents interface{}
}

type RegisterMessage struct {
	Username     string
	PasswordHash string
}

type AuthenticateMessage struct {
	Username     string
	PasswordHash string
}

type TokenMessage struct {
	Username string
	Token    string
	Message  string
}

type TextMessage struct {
	Username string
	Room     string
	Text     string
	Time     time.Time
}

type SendTextMessage struct {
	Token   string
	Message TextMessage
}

type RecvTextMessage struct {
	Message TextMessage
}

type PopulateMessages struct {
	Room      string
	Messages  []TextMessage
	TimeSince int
	Limit     int
	Token     string
}

type ListRoomsMessage struct {
	Rooms []string
}

type JoinRoomMessage struct {
	Username    string
	Room        string
	IsSuperUser bool
	Token       string
	Status      STATUS
	Message     string
}

type LeaveRoomMessage struct {
	Username string
	Room     string
	Token    string
	Status   STATUS
	Message  string
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
	gob.Register(PopulateMessages{})
}

func SendRemoteCommand(encoder *gob.Encoder, message Message) error {
	return encoder.Encode(message)
}

func BuildMessage(message_type COMMAND, contents interface{}) Message {
	return Message{Command: message_type, Contents: contents}
}

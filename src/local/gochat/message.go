package gochat

import (
	"encoding/json"
	"encoding/gob"
	"net"
)

type COMMAND string

const (
	AUTHENTICATE    = COMMAND("AUTHENTICATE")
	TOKEN           = COMMAND("TOKEN")
	SEND_MESSAGE    = COMMAND("SEND_MESSAGE")
	RECV_MESSAGE    = COMMAND("RECV_MESSAGE")
)

type Message struct {
	command COMMAND
	contents string
}

func SendRemoteCommand(connection net.Conn, message Message) error {
	encoder := gob.NewEncoder(connection)
	return encoder.Encode(message)
}

func BuildMessage(message_type COMMAND, contents interface{}) Message {
	return Message{command: message_type, contents: string(json.Marshal(contents))}
}
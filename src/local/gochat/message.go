package gochat

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

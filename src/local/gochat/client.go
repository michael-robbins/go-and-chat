package gochat

import (
	"encoding/gob"
	"encoding/hex"
	"crypto/sha256"
	"errors"
	"bufio"
	"fmt"
	"net"
	"os"
)

type ChatClient struct {
	conn net.Conn
	token string
}

func NewChatClient() (*ChatClient, error) {
	return &ChatClient{}, nil
}

func (client *ChatClient) Connect(connection_string string) error {
	// Attempt to connect to the server returning the connection status
	conn, err := net.Dial("tcp", connection_string)
	if err != nil {
		return errors.New("Unable to connect to the server")
	}

	client.conn = conn

	return nil
}

func (client *ChatClient) Authenticate(username string, password string) error {
	// Hash the password
	password_hash := hex.EncodeToString([]byte(sha256.Sum256([]byte(password))))

	// Send off the authentication attempt, the response will be handled elsewhere
	return SendRemoteCommand(client.conn,
		BuildMessage(AUTHENTICATE, map[string]string{"username": username, "password": password_hash}))
}

func (client *ChatClient) ListenToServer() error {
	decoder := gob.NewDecoder(client.conn)

	for {
		message := Message{}
		decoder.Decode(&message)

		if err := client.HandleMessage(message); err != nil {
			fmt.Println("Error handling the message, the error was:")
			fmt.Println(err)
		}

		fmt.Println("Breaking out of ListenToServer loop")
		break
	}

	return nil
}

func (client *ChatClient) ListenToUser() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter Message: ")
		message, _ := reader.ReadString('\n')

		if err := client.SendMessage(message); err != nil {
			fmt.Println(err)
		}
	}
}

func (client *ChatClient) HandleMessage(message Message) error {
	// Interpret message
	switch message.command {
		case AUTHENTICATE:
			// This message isn't a client command
			break
		case TOKEN:
			// Response to an authentication attempt
			client.token = message.contents
		case SEND_MESSAGE:
			// This message isn't a client command
			break
		case RECV_MESSAGE:
			if err := client.DisplayMessage(message.contents); err != nil {
				return err
			}
		default:
			// Unknown message command
			break
	}

	return nil
}

func (client *ChatClient) DisplayMessage(blob string) error {
	fmt.Println("Displaying Client Message:")
	fmt.Println(blob)

	return nil
}

func (client *ChatClient) SendMessage(content string) error {
	if client.token == "" {
		return errors.New("Unable to send message as we have not authenticated yet!")
	}

	return SendRemoteCommand(client.conn,
		BuildMessage(SEND_MESSAGE, map[string]string{"token": client.token, "message": content}))
}

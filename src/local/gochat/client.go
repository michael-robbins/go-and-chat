package gochat

import (
	"encoding/json"
	"encoding/gob"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"net"
	"io"
	"bufio"
	"os"
)

const (
	SALT_BYTES = 64
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
	// Generate a salt
	salt := make([]byte, SALT_BYTES)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		return err
	}

	// Hash the password with the salt
	combined := string(salt) + password
	hash := sha1.Sum([]byte(combined))

	// Authenticate message contents
	contents := map[string]string{
		"username": username,
		"salt": string(salt),
		"password": string(hash),
	}

	// Send off the authentication attempt, the response will be handled elsewhere
	message := Message{command: AUTHENTICATE, contents: string(json.Marshal(contents))}

	encoder := gob.NewEncoder(client.conn)
	encoder.Encode(message)

	return nil
}

func (client *ChatClient) ListenToServer() error {
	decoder := gob.NewDecoder(client.conn)

	for {
		message := Message{}
		decoder.Decode(&message)

		if err = client.HandleMessage(message); err != nil {
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

		if err = client.SendMessage(message); err != nil {
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
			if err = client.DisplayMessage(message.contents); err != nil {
				return err
			}
		default:
			// Unknown message command
			break
	}

	return nil
}

func (client *ChatClient) SendServerCommand(message Message) error {
	encoder := gob.NewEncoder(client.conn)
	encoder.Encode(message)

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

	// Build message contents
	contents := map[string]string{
		"token": client.token,
		"message": content,
	}

	command := Message{command: SEND_MESSAGE, contents: string(json.Marshal(contents))}

	if err := client.SendServerCommand(command); err != nil {
		return err
	}

	return nil
}

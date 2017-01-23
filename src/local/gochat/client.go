package gochat

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"bufio"
	"fmt"
	"net"
	"os"
)

type ChatClient struct {
	conn net.Conn
	username string
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
	// Save the username
	client.username = username

	// Hash the password
	password_hash := hex.EncodeToString([]byte(sha256.Sum256([]byte(password))))

	// Send off the authentication attempt, the response will be handled elsewhere
	return SendRemoteCommand(client.conn,
		BuildMessage(AUTHENTICATE, AuthenticateMessage{username: username, password_hash: password_hash}))
}

func (client *ChatClient) ListenToUser() error {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter Message: ")
		message, _ := reader.ReadString('\n')

		fmt.Print("Enter Room: ")
		room, _ := reader.ReadString('\n')

		if err := client.SendMessage(message, room); err != nil {
			fmt.Println(err)
		}
	}
}

func (client *ChatClient) JoinRoom(room string) error {
	if client.token == "" {
		return errors.New("Unable to join a room as we have not authenticated yet!")
	}

	message := BuildMessage(JOIN_ROOM,
		JoinRoomMessage{
			username: client.username,
			room: room,
			isSuperUser: false,
			token: client.token,
		})

	return SendRemoteCommand(client.conn, message)
}

func (client *ChatClient) LeaveRoom(room string) error {
	if client.token == "" {
		return errors.New("Unable to leave a room as we have not authenticated yet!")
	}

	message := BuildMessage(LEAVE_ROOM,
		LeaveRoomMessage{
			username: client.username,
			room: room,
			token: client.token,
		})

	return SendRemoteCommand(client.conn, message)
}

func (client *ChatClient) CreateRoom(room string, capacity int) error {
	if client.token == "" {
		return errors.New("Unable to create a room as we have not authenticated yet!")
	}

	message := BuildMessage(CREATE_ROOM,
		CreateRoomMessage{
			room: room,
			capacity: capacity,
			token: client.token,
		})

	return SendRemoteCommand(client.conn, message)
}

func (client *ChatClient) CloseRoom(room string) error {
	if client.token == "" {
		return errors.New("Unable to create a room as we have not authenticated yet!")
	}

	message := BuildMessage(CLOSE_ROOM,
		CloseRoomMessage{
			room: room,
			token: client.token,
		})

	return SendRemoteCommand(client.conn, message)
}

func (client *ChatClient) SendMessage(content string, room string) error {
	if client.token == "" {
		return errors.New("Unable to send message as we have not authenticated yet!")
	}

	message := BuildMessage(SEND_MSG,
		SendTextMessage{
			token: client.token,
			message: TextMessage{username: client.username, text: content, room: room},
		})

	return SendRemoteCommand(client.conn, message)
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

func (client *ChatClient) HandleMessage(message Message) error {
	// Interpret message
	switch message.command {
	case TOKEN:
		contents := message.contents.(TokenMessage)
		client.token = contents.token
	case RECV_MSG:
		contents := message.contents.(RecvTextMessage)
		client.DisplayTextMessage(contents.message)
	case LIST_ROOMS:
		contents := message.contents.(ListRoomsMessage)
		client.DisplayRoomListingMessage(contents)
	default:
		// Unknown message command
		return errors.New("Unable to determine incoming message type from server")
	}

	return nil
}

func (client *ChatClient) DisplayTextMessage(message TextMessage) {
	fmt.Println("[" + message.room + "] " + message.username + ":", message.text)
}

func (client *ChatClient) DisplayRoomListingMessage(message ListRoomsMessage) {
	fmt.Println("Room Listing:")
	for i, room := range message.rooms {
		fmt.Println(string(i) + ":", room)
	}
}

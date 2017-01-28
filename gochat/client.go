package gochat

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
)

type ChatClient struct {
	conn     net.Conn
	username string
	token    string
}

func NewChatClient() (*ChatClient, error) {
	return &ChatClient{}, nil
}

func (client *ChatClient) Connection() (net.Conn, error) {
	if client.conn != nil {
		return client.conn, nil
	}

	return nil, errors.New("Not connected")
}

func (client *ChatClient) Connect(connection_string string) (net.Conn, error) {
	// Attempt to connect to the server returning the connection status
	conn, err := net.Dial("tcp", connection_string)
	if err != nil {
		return nil, err
	}

	client.conn = conn

	return client.conn, nil
}

func (client *ChatClient) Authenticate(username string, password string) error {
	// Save the username
	client.username = username

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	password_hash_hex := hex.EncodeToString(password_hash[:])

	// Send off the authentication attempt, the response will be handled elsewhere
	return SendRemoteCommand(client.conn,
		BuildMessage(AUTHENTICATE, AuthenticateMessage{username: username, password_hash: password_hash_hex}))
}

func (client *ChatClient) ListenToUser(message_channel chan<- Message, exit chan<- int) error {
	client_commands := []COMMAND{LIST_ROOMS, JOIN_ROOM, CREATE_ROOM, CLOSE_ROOM}

UserMenuLoop:
	for {
		number := getClientCommandsOption(client_commands)

		// The user has indicated to quit
		if number == -1 {
			exit <- 1
		}

		command := client_commands[number]

		// Populate the room_name if required
		var room_name string

		switch command {
		case JOIN_ROOM, LEAVE_ROOM, CREATE_ROOM, CLOSE_ROOM:
			room_name := getRoomName()
			if room_name == "" {
				// The user has indicated to return to the main menu
				continue UserMenuLoop
			}
		}

		// Populate the room_capacity if required
		var room_capacity int

		switch command {
		case CREATE_ROOM:
			room_capacity = getRoomCapacity()
			if room_capacity == -1 {
				// The user has indicated to return to the main menu
				continue UserMenuLoop
			}
		}

		// Generate the message to send
		var message Message
		var err error

		switch command {
		case LIST_ROOMS:
			message, err = client.BuildListRoomsMessage()
		case JOIN_ROOM:
			message, err = client.BuildJoinRoomMessage(room_name)
		case LEAVE_ROOM:
			message, err = client.BuildLeaveRoomMessage(room_name)
		case CREATE_ROOM:
			message, err = client.BuildCreateRoomMessage(room_name, room_capacity)
		case CLOSE_ROOM:
			message, err = client.BuildCloseRoomMessage(room_name)
		}

		// Send the message into the queue or print out the error and continue the main loop
		if err != nil {
			fmt.Println(err)
		} else {
			message_channel <- message
		}
	}
}

func (client *ChatClient) BuildListRoomsMessage() (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to list any rooms we have not authenticated yet!")
	}

	return BuildMessage(LIST_ROOMS, ListRoomsMessage{}), nil
}

func (client *ChatClient) BuildJoinRoomMessage(room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to join a room as we have not authenticated yet!")
	}

	return BuildMessage(JOIN_ROOM,
		JoinRoomMessage{
			username:    client.username,
			room:        room,
			isSuperUser: false,
			token:       client.token,
		}), nil
}

func (client *ChatClient) BuildLeaveRoomMessage(room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to leave a room as we have not authenticated yet!")
	}

	return BuildMessage(LEAVE_ROOM,
		LeaveRoomMessage{
			username: client.username,
			room:     room,
			token:    client.token,
		}), nil
}

func (client *ChatClient) BuildCreateRoomMessage(room string, capacity int) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to create a room as we have not authenticated yet!")
	}

	return BuildMessage(CREATE_ROOM,
		CreateRoomMessage{
			room:     room,
			capacity: capacity,
			token:    client.token,
		}), nil
}

func (client *ChatClient) BuildCloseRoomMessage(room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to close a room as we have not authenticated yet!")
	}

	return BuildMessage(CLOSE_ROOM,
		CloseRoomMessage{
			room:  room,
			token: client.token,
		}), nil
}

func (client *ChatClient) BuildSendMessageMessage(content string, room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to send message as we have not authenticated yet!")
	}

	return BuildMessage(SEND_MSG,
		SendTextMessage{
			token:   client.token,
			message: TextMessage{username: client.username, text: content, room: room},
		}), nil
}

func (client *ChatClient) ListenToServer(notify chan<- Message) error {
	decoder := gob.NewDecoder(client.conn)

	for {
		message := Message{}
		decoder.Decode(&message)
		notify <- message
	}

	return nil
}

func (client *ChatClient) HandleServerMessage(message Message) error {
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
	fmt.Println("["+message.room+"] "+message.username+":", message.text)
}

func (client *ChatClient) DisplayRoomListingMessage(message ListRoomsMessage) {
	fmt.Println("Room Listing:")
	for i, room := range message.rooms {
		fmt.Println(string(i)+":", room)
	}
}

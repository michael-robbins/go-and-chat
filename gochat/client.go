package gochat

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	log "github.com/Sirupsen/logrus"
)

type ChatClient struct {
	conn     net.Conn
	logger   *log.Entry
	username string
	token    string
}

func NewChatClient(logger *log.Entry) (*ChatClient, error) {
	return &ChatClient{logger: logger}, nil
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

func (client *ChatClient) Register(username string, password string) error {
	return SendRemoteCommand(client.conn,
		BuildMessage())
}

func (client *ChatClient) Authenticate(username string, password string) error {
	// Save the Username
	client.username = username

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	password_hash_hex := hex.EncodeToString(password_hash[:])

	// Send off the authentication attempt, the response will be handled elsewhere
	client.logger.Debug("Sending auth request to the server")
	return SendRemoteCommand(client.conn,
		BuildMessage(AUTHENTICATE, AuthenticateMessage{Username: username, PasswordHash: password_hash_hex}))
}

func (client *ChatClient) ListenToUser(message_channel chan<- Message, exit chan<- int) error {
	client_commands := []COMMAND{LIST_ROOMS, JOIN_ROOM, CREATE_ROOM, CLOSE_ROOM}

UserMenuLoop:
	for {
		number := getClientCommandsOption(client_commands)

		// The user has indicated to quit
		if number == -1 {
			// Fire an exit event to the event loop
			exit <- 1

			// Return to stop this go thread
			return nil
		}

		// We offset by one as the user is only given the choices of 1 -> len(client_commands)
		command := client_commands[number-1]

		// Ensure that any commands that require authentication have a Token
		switch command {
		case LIST_ROOMS, JOIN_ROOM, CREATE_ROOM, CLOSE_ROOM:
			if client.token == "" {
				fmt.Println("Unable to do that, as we have not authenticated yet!")
				continue UserMenuLoop
			}
		}

		// Populate the room_name if required
		var roomName string

		switch command {
		case JOIN_ROOM, LEAVE_ROOM, CREATE_ROOM, CLOSE_ROOM:
			roomName := getRoomName()
			if roomName == "" {
				// The user has indicated to return to the main menu
				continue UserMenuLoop
			}
		}

		// Populate the room_capacity if required
		var roomCapacity int

		switch command {
		case CREATE_ROOM:
			roomCapacity = getRoomCapacity()
			if roomCapacity == -1 {
				// The user has indicated to return to the main menu
				continue UserMenuLoop
			}
		}

		// Generate the Message to send
		var message Message
		var err error

		switch command {
		case LIST_ROOMS:
			message, err = client.BuildListRoomsMessage()
		case JOIN_ROOM:
			message, err = client.BuildJoinRoomMessage(roomName)

		case CREATE_ROOM:
			message, err = client.BuildCreateRoomMessage(roomName, roomCapacity)
		case CLOSE_ROOM:
			message, err = client.BuildCloseRoomMessage(roomName)
		}

		// Send the Message into the queue or print out the error and continue the main loop
		if err != nil {
			fmt.Println(err)
		} else {
			message_channel <- message
		}

		// The user now thinks they're in a room, so we enter 'room' mode and poll them for messages to send
		if command == JOIN_ROOM {
			// Keep looping asking for messages to send until they quit
			for {
				textMessage := getTextMessage()

				if textMessage == "" {
					// User has indicated to leave the room
					message, err = client.BuildLeaveRoomMessage(roomName)
					if err != nil {
						fmt.Println(err)
					} else {
						// Send the leave room request
						message_channel<- message
					}

					break
				}

				// User has indicated to leave the room
				message, err = client.BuildSendMessageMessage(textMessage, roomName)
				if err != nil {
					fmt.Println(err)
					break
				} else {
					message_channel<- message
				}

				// Continue the loop asking for another text message to send
			}
		}
	}
}

func (client *ChatClient) BuildListRoomsMessage() (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to list any Rooms we have not authenticated yet!")
	}

	return BuildMessage(LIST_ROOMS, ListRoomsMessage{}), nil
}

func (client *ChatClient) BuildJoinRoomMessage(room string) (Message, error) {
	return BuildMessage(JOIN_ROOM,
		JoinRoomMessage{
			Username:    client.username,
			Room:        room,
			IsSuperUser: false,
			Token:       client.token,
		}), nil
}

func (client *ChatClient) BuildLeaveRoomMessage(room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to leave a Room as we have not authenticated yet!")
	}

	return BuildMessage(LEAVE_ROOM,
		LeaveRoomMessage{
			Username: client.username,
			Room:     room,
			Token:    client.token,
		}), nil
}

func (client *ChatClient) BuildCreateRoomMessage(room string, capacity int) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to create a Room as we have not authenticated yet!")
	}

	return BuildMessage(CREATE_ROOM,
		CreateRoomMessage{
			Room:     room,
			Capacity: capacity,
			Token:    client.token,
		}), nil
}

func (client *ChatClient) BuildCloseRoomMessage(room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to close a Room as we have not authenticated yet!")
	}

	return BuildMessage(CLOSE_ROOM,
		CloseRoomMessage{
			Room:  room,
			Token: client.token,
		}), nil
}

func (client *ChatClient) BuildSendMessageMessage(content string, room string) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to send Message as we have not authenticated yet!")
	}

	return BuildMessage(SEND_MSG,
		SendTextMessage{
			Token:   client.token,
			Message: TextMessage{Username: client.username, Text: content, Room: room},
		}), nil
}

func (client *ChatClient) ListenToServer(notify chan<- Message) error {
	decoder := gob.NewDecoder(client.conn)
	var empty_message Message

	for {
		message := Message{}
		decoder.Decode(&message)

		if message == empty_message {
			// If we did not decode anything just sleep for a second and try again
			time.Sleep(time.Second * 1)
			continue
		}

		notify <- message
	}

	return nil
}

func (client *ChatClient) HandleServerMessage(message Message) error {
	// Interpret Message
	switch message.Command {
	case TOKEN:
		contents := message.Contents.(TokenMessage)
		client.token = contents.Token
	case RECV_MSG:
		contents := message.Contents.(RecvTextMessage)
		client.DisplayTextMessage(contents.Message)
	case LIST_ROOMS:
		contents := message.Contents.(ListRoomsMessage)
		client.DisplayRoomListingMessage(contents)
	default:
		// Unknown Message command
		return errors.New("Unable to determine incoming Message type from server.")
	}

	return nil
}

func (client *ChatClient) DisplayTextMessage(message TextMessage) {
	fmt.Println("["+message.Room+"] "+message.Username+":", message.Text)
}

func (client *ChatClient) DisplayRoomListingMessage(message ListRoomsMessage) {
	fmt.Println("Room Listing:")
	for i, room := range message.Rooms {
		fmt.Println(string(i)+":", room)
	}
}

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
	encoder  *gob.Encoder
	decoder  *gob.Decoder
	logger   *log.Entry
	username string
	token    string
}

func NewChatClient(logger *log.Entry) (*ChatClient, error) {
	return &ChatClient{logger: logger}, nil
}

func (client *ChatClient) Connect(connection_string string) error {
	// Attempt to connect to the server returning the connection status
	conn, err := net.Dial("tcp", connection_string)
	if err != nil {
		return err
	}

	client.encoder = gob.NewEncoder(conn)
	client.decoder = gob.NewDecoder(conn)

	return nil
}

func (client *ChatClient) EventLoop(server_messages <-chan Message, client_messages <-chan Message, exit <-chan int) {
EventLoop:
	for {
		select {
		case message := <-server_messages:
			// Handle the server initiated message
			client.logger.Debug("Handling Server Message: " + message.Command)
			if err := client.HandleServerMessage(message); err != nil {
				client.logger.Error(err)
			}
		case message := <-client_messages:
			// Handle the client initiated message
			if err := SendRemoteCommand(client.encoder, message); err != nil {
				client.logger.Error(err)
			} else {
				client.logger.Debug("Successfully sent " + message.Command + " message.")
			}
		case _ = <-exit:
			break EventLoop
		default:
			// Sleep for half a second then check again for any server/client messages or exit decisions
			time.Sleep(time.Millisecond * 500)
		}
	}
}

func (client *ChatClient) Register(username string, password string) error {
	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	password_hash_hex := hex.EncodeToString(password_hash[:])

	client.logger.Debug("Sending registration request to the server")
	return SendRemoteCommand(client.encoder,
		BuildMessage(REGISTER, RegisterMessage{Username: username, PasswordHash: password_hash_hex}))
}

func (client *ChatClient) Authenticate(username string, password string) error {
	// Save the Username
	client.username = username

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	password_hash_hex := hex.EncodeToString(password_hash[:])

	// Send off the authentication attempt, the response will be handled elsewhere
	client.logger.Debug("Sending auth request to the server")
	return SendRemoteCommand(client.encoder,
		BuildMessage(AUTHENTICATE, AuthenticateMessage{Username: username, PasswordHash: password_hash_hex}))
}

func (client *ChatClient) ListenToUser(message_channel chan<- Message) error {
	client_commands := []COMMAND{LIST_ROOMS, JOIN_ROOM, CREATE_ROOM, CLOSE_ROOM}

UserMenuLoop:
	for {
		number := getClientCommandsOption(client_commands)

		// The user has indicated to quit
		if number == -1 {
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
			roomName = getRoomName()
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
			// Send a 'populate' message requesting backfill of messages for this room
			backfill_message, err := client.BuildPopulateMessage(roomName, time.Now())
			message_channel <- backfill_message

			// Keep looping asking for messages to send until they quit
			for {
				textMessage := getTextMessage()

				if textMessage == "" {
					// ServerUser has indicated to leave the room
					message, err = client.BuildLeaveRoomMessage(roomName)
					if err != nil {
						fmt.Println(err)
					} else {
						// Send the leave room request
						message_channel <- message
					}

					break
				}

				// ServerUser has indicated to leave the room
				message, err = client.BuildSendMessageMessage(textMessage, roomName)
				if err != nil {
					fmt.Println(err)
					break
				} else {
					message_channel <- message
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

func (client *ChatClient) BuildPopulateMessage(roomName string, timeSince time.Time) (Message, error) {
	if client.token == "" {
		return Message{}, errors.New("Unable to send populate request as we have not authenticated yet!")
	}

	return BuildMessage(POP_MSGS,
		PopulateMessages{
			Room:      roomName,
			TimeSince: int(timeSince.Unix()),
			Token:     client.token,
		}), nil
}

func (client *ChatClient) ListenToServer(notify chan<- Message, exit <-chan int, auth chan<- bool) error {
	var empty_message Message

ListenLoop:
	for {
		select {
		case _ = <-exit:
			break ListenLoop
		default:
			// This needs to be here for the above channel read to be non-blocking
			fmt.Print("")
		}

		message := Message{}
		client.decoder.Decode(&message)

		if message == empty_message {
			// If we did not decode anything just sleep for a second and try again
			time.Sleep(time.Second * 1)
			continue ListenLoop
		}

		if message.Command == TOKEN {
			tokenMsg := message.Contents.(TokenMessage)
			if tokenMsg.Token != "" {
				auth <- true
			} else {
				auth <- false
			}
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

		if contents.Token != "" {
			client.token = contents.Token
			fmt.Println(contents.Message)
		} else {
			fmt.Println(contents.Message)
		}

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
	for _, room := range message.Rooms {
		fmt.Println("* " + room)
	}
}

package gochat

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"strconv"
	"errors"
	"bufio"
	"fmt"
	"net"
	"os"
	"math"
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
	client_commands := []COMMAND{LIST_ROOMS, JOIN_ROOM, CREATE_ROOM, CLOSE_ROOM}

	MainMenu:
	for {
		number := -1

		// Inner for loop will break when we have a valid choice
		for {
			if number != -1 {
				break
			}

			fmt.Println("Please select an option:")
			for i, command := range client_commands {
				fmt.Println(string(i) + ":", command)
			}

			text := getUserInput("Choice (number): ")
			if text == "quit" || text == "q" {
				return nil
			}

			var err error
			number, err = strconv.Atoi(text)
			if err != nil || number < 1 || number > len(client_commands) {
				fmt.Println("Invalid choice (Only '1' -> '" + string(len(client_commands)) +"').")
			}

			break
		}

		switch client_commands[number] {
		case LIST_ROOMS:
			client.ListRooms()
			fmt.Println("Client: Sent list rooms request!")
		case JOIN_ROOM:
			var room_name string
			for {
				if room_name != "" {
					break
				}

				room_name := getUserInput("Room to join: ")
				if room_name == "quit" || room_name == "q" {
					continue MainMenu
				}
			}

			client.JoinRoom(room_name)
			fmt.Println("Client: Sent join room request!")
		case CREATE_ROOM:
			var room_name string
			for {
				if room_name != "" {
					break
				}

				room_name := getUserInput("Room to create: ")
				if room_name == "quit" || room_name == "q" {
					continue MainMenu
				}
			}

			room_capacity := -1
			for {
				if room_capacity != -1 {
					break
				}

				text := getUserInput("Room to create: ")

				if room_name == "quit" || room_name == "q" {
					continue MainMenu
				}

				var err error
				number, err := strconv.Atoi(text)

				if err != nil || number < 1 {
					fmt.Println("Invalid choice (Only '1' -> 'MAX_INT32'.")
				}

				room_capacity = number
			}

			client.CreateRoom(room_name, room_capacity)
			fmt.Println("Client: Sent create room request!")
		case CLOSE_ROOM:
			var room_name string
			for {
				if room_name != "" {
					break
				}

				room_name := getUserInput("Room to join: ")
				if room_name == "quit" || room_name == "q" {
					continue MainMenu
				}
			}

			client.CloseRoom(room_name)
			fmt.Println("Client: Close room request!")
		}
	}
}

func getUserInput(message string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("'quit' or 'q' will exit.")
	fmt.Println("")
	fmt.Print(message)

	text, _ := reader.ReadString('\n')

	return text
}


func (client *ChatClient) ListRooms() error {
	if client.token == "" {
		return errors.New("Unable to list any rooms we have not authenticated yet!")
	}

	return SendRemoteCommand(client.conn, BuildMessage(LIST_ROOMS, ListRoomsMessage{}))
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

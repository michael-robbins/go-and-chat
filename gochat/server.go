package gochat

import (
	"encoding/gob"
	"errors"
	"fmt"
	"net"
)

const (
	UNABLE_TO_ACCEPT_MESSAGE = "Unable to accept connection correctly."
	INVALID_TOKEN            = "Token is invalid."
)

type ChatServer struct {
	user_manager *UserManager
	room_manager *RoomManager
}

func NewChatServer() (*ChatServer, error) {
	chat_server := ChatServer{
		user_manager: NewUserManager(),
		room_manager: NewRoomManager(),
	}

	return &chat_server, nil
}

func (server ChatServer) Listen(connection_string string) error {
	// Bind to the IP/Port and listen for new incoming connections
	socket, err := net.Listen("tcp", connection_string)
	if err != nil {
		return err
	}

	for {
		connection, err := socket.Accept()
		if err != nil {
			fmt.Println(UNABLE_TO_ACCEPT_MESSAGE)
		}
		go server.HandleIncomingConnection(connection)
	}
}

func (server *ChatServer) HandleIncomingConnection(connection net.Conn) {
	defer connection.Close()

	decoder := gob.NewDecoder(connection)

	message := Message{}
	decoder.Decode(&message)

	reply, err := server.HandleMessage(message)
	if err != nil {
		fmt.Println(err)
		return
	} else if message.Command != "" {
		// Only send a reply if the command is not empty
		encoder := gob.NewEncoder(connection)
		encoder.Encode(reply)
	}
}

func (server *ChatServer) HandleMessage(message Message) (Message, error) {
	// If the message provides a token, ensure it's valid
	passes, err := server.messagePassesTokenTest(message)
	if err != nil {
		return Message{}, err
	}

	if !passes {
		return Message{}, errors.New(INVALID_TOKEN)
	}
	// We assume now that any requests that require a token are valid (authenticated)

	// Get the room if this message contains one
	// This saves us having the same room extraction code for each message type
	room, err := server.getRoomIfRequired(message)
	if err != nil {
		return Message{}, err
	}

	// Get the user if this message has one
	// This saves us having the same user extraction code for each message type
	user, err := server.getUserIfRequired(message)
	if err != nil {
		return Message{}, err
	}

	// Interpret message
	switch message.Command {
	case AUTHENTICATE:
		contents := message.Contents.(AuthenticateMessage)
		user, err := server.user_manager.AuthenticateUser(contents.username, contents.password_hash)
		if err != nil {
			return Message{}, err
		}

		// Respond with the authentication token
		return BuildMessage(TOKEN, TokenMessage{username: user.username, token: user.GetToken()}), nil
	case SEND_MSG:
		contents := message.Contents.(SendTextMessage)
		for _, user := range room.users {
			SendRemoteCommand(user.conn, BuildMessage(RECV_MSG, RecvTextMessage{message: contents.message}))
		}
	case JOIN_ROOM:
		contents := message.Contents.(JoinRoomMessage)
		if err := room.AddUser(user, contents.isSuperUser); err != nil {
			return Message{}, err
		}
	case LEAVE_ROOM:
		if err := room.RemoveUser(user); err != nil {
			return Message{}, err
		}
	case CREATE_ROOM:
		contents := message.Contents.(CreateRoomMessage)
		_, err := server.room_manager.CreateRoom(contents.room, contents.capacity)
		if err != nil {
			return Message{}, err
		}
	case CLOSE_ROOM:
		contents := message.Contents.(CloseRoomMessage)
		if _, err := server.room_manager.CloseRoom(contents.room); err != nil {
			return Message{}, err
		}
	}

	return Message{}, nil
}

func (server *ChatServer) messagePassesTokenTest(message Message) (bool, error) {
	// Ensure any message requiring a token is valid
	var token string

	switch message.Command {
	case SEND_MSG:
		token = message.Contents.(SendTextMessage).token
	case JOIN_ROOM:
		token = message.Contents.(JoinRoomMessage).token
	case LEAVE_ROOM:
		token = message.Contents.(LeaveRoomMessage).token
	case CREATE_ROOM:
		token = message.Contents.(CreateRoomMessage).token
	case CLOSE_ROOM:
		token = message.Contents.(CloseRoomMessage).token
	default:
		return true, nil
	}

	if valid, _ := server.user_manager.TokenIsValid(token); valid {
		return true, nil
	}

	// Token is provided, but is not valid
	return false, nil
}

func (server *ChatServer) getRoomIfRequired(message Message) (*ChatRoom, error) {
	var name string

	switch message.Command {
	case SEND_MSG:
		name = message.Contents.(SendTextMessage).message.room
	case JOIN_ROOM:
		name = message.Contents.(JoinRoomMessage).room
	case LEAVE_ROOM:
		name = message.Contents.(LeaveRoomMessage).room
	case CREATE_ROOM:
		name = message.Contents.(CreateRoomMessage).room
	case CLOSE_ROOM:
		name = message.Contents.(CloseRoomMessage).room
	default:
		return nil, nil
	}

	room, err := server.room_manager.GetRoom(name)
	if err != nil {
		return nil, err
	}

	return room, nil
}

func (server *ChatServer) getUserIfRequired(message Message) (*User, error) {
	var name string

	switch message.Command {
	case SEND_MSG:
		name = message.Contents.(SendTextMessage).message.username
	case JOIN_ROOM:
		name = message.Contents.(JoinRoomMessage).username
	case LEAVE_ROOM:
		name = message.Contents.(LeaveRoomMessage).username
	default:
		return nil, nil
	}

	user, err := server.user_manager.GetUser(name)
	if err != nil {
		return nil, err
	}

	return user, nil
}

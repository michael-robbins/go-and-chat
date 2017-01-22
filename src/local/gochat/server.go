package gochat

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"crypto/rand"
	"errors"
	"net"
	"fmt"
	"io"
)

const (
	SALT_BYTES = 64
)

type ChatServer struct {
	user_manager *UserManager
	room_manager *RoomManager
}

type UserCredentials struct {
	username string `json:"username"`
	password_sha256 string `json:"password"`
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
			fmt.Println("Unable to accept connection correctly.")
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
	} else if message.command == "" {
		encoder := gob.NewEncoder(connection)
		encoder.Encode(reply)
	}
}

func (server *ChatServer) HandleMessage(message Message) (Message, error) {
	// Interpret message
	switch message.command {
	case AUTHENTICATE:
		contents := message.contents.(AuthenticateMessage)
		user_obj, err := server.AuthenticateUser(contents.username, contents.password_hash)
		if err != nil {
			return nil, err
		}

		// Respond with the authentication token
		return BuildMessage(TOKEN, TokenMessage{username: user_obj.username, token: user_obj.GetToken()}), nil
	case SEND_MSG:
		contents := message.contents.(SendTextMessage)

		// Ensure the user's token is valid
		if valid, _ := server.user_manager.TokenIsValid(contents.token); valid {
			server.sendTextMessageToRoom(contents.message)
		}
		return Message{}, nil
	}

	return Message{}, nil
}

func (server *ChatServer) AuthenticateUser(username string, password_sha256 string) (User, error) {
	user_object, err := server.user_manager.GetUser(username)
	if err != nil {
		return nil, err
	}

	client_hash, err := hex.DecodeString(password_sha256)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users password hash.")
	}

	server_salt, err := hex.DecodeString(user_object.salt)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users server salt.")
	}

	server_hash, err := hex.DecodeString(user_object.password_sha256)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("Error decoding users server password hash.")
	}

	client_side_hash := sha256.Sum256(append(server_salt, client_hash...))
	server_side_hash := sha256.Sum256(append(server_salt, server_hash...))

	if client_side_hash == server_side_hash {
		return *user_object, nil
	} else {
		return nil, errors.New("Invalid password")
	}

	return *user_object, nil
}

func (server *ChatServer) registerUser(username string, password string) (User, error) {
	// Generate a salt
	salt := make([]byte, SALT_BYTES)
	_, err := io.ReadFull(rand.Reader, salt)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("There was an error registering the user.")
	}

	// Hash the password
	password_hash := sha256.Sum256([]byte(password))
	salted_hash := sha256.Sum256(append(salt, password_hash...))

	user := User{
		username: username,
		salt: hex.EncodeToString(salt),
		password_sha256: hex.EncodeToString(salted_hash[:]),
	}

	return user, nil
}

func (server *ChatServer) sendTextMessageToRoom(message TextMessage) error {
	room, err := server.room_manager.GetRoom(message.room)
	if err != nil {
		return err
	}

	for _, user := range room.users {
		SendRemoteCommand(user.conn, BuildMessage(RECV_MSG, RecvTextMessage{message: message}))
	}

	return nil
}
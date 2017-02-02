package gochat

import (
	"encoding/gob"
	"errors"
	"io/ioutil"
	"net"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

type ChatServer struct {
	user_manager *UserManager
	room_manager *RoomManager
	logger       *log.Entry
}

type ServerConfig struct {
	Database DatabaseConfig `yaml:"database"`
}

type DatabaseConfig struct {
	Product  string `yaml:"product"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func NewChatServer(logger *log.Entry, config ServerConfig) (*ChatServer, error) {
	storageManager, err := NewStorageManager(config.Database)
	if err != nil {
		return &ChatServer{}, err
	}

	userManager, err := NewUserManager(storageManager)
	if err != nil {
		return &ChatServer{}, err
	}

	roomManager, err := NewRoomManager(storageManager)
	if err != nil {
		return &ChatServer{}, err
	}

	chat_server := ChatServer{
		user_manager: userManager,
		room_manager: roomManager,
		logger:       logger,
	}

	return &chat_server, nil
}

func (server ChatServer) Listen(connection_string string) error {
	// Bind to the IP/Port and listen for new incoming connections
	socket, err := net.Listen("tcp", connection_string)
	if err != nil {
		return err
	}

	server.logger.Info("Listening on " + connection_string)

	for {
		connection, err := socket.Accept()
		if err != nil {
			server.logger.Error("Unable to accept connection correctly.")
		}

		server.logger.Info("Accepted incoming connection")
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
		server.logger.Error(err)
		return
	} else if message.Command != "" {
		// Only send a reply if the command is not empty
		server.logger.Debug("Sending response back to client")
		encoder := gob.NewEncoder(connection)
		encoder.Encode(reply)
	}
}

func (server *ChatServer) HandleMessage(message Message) (Message, error) {
	// If the Message provides a Token, ensure it's valid
	passes, err := server.messagePassesTokenTest(message)
	if err != nil {
		return Message{}, err
	}

	if !passes {
		return Message{}, errors.New("Token is invalid")
	}
	// We assume now that any requests that require a Token are valid (authenticated)

	// Get the Room if this Message contains one
	// This saves us having the same Room extraction code for each Message type
	room, err := server.getRoomIfRequired(message)
	if err != nil {
		return Message{}, err
	}

	// Get the user if this Message has one
	// This saves us having the same user extraction code for each Message type
	user, err := server.getUserIfRequired(message)
	if err != nil {
		return Message{}, err
	}

	// Interpret Message
	switch message.Command {
	case AUTHENTICATE:
		server.logger.Debug("We have a valid auth attempt!")
		contents := message.Contents.(AuthenticateMessage)

		server.logger.Debug("Parsed the contents of the Authentication message")
		user, err := server.user_manager.AuthenticateUser(contents.Username, contents.PasswordHash)
		if err != nil {
			return Message{}, err
		}

		// Respond with the authentication Token
		server.logger.Debug("Successfully authenticated user!")
		return BuildMessage(TOKEN, TokenMessage{Username: user.Username, Token: user.GetToken()}), nil
	case SEND_MSG:
		contents := message.Contents.(SendTextMessage)
		for _, user := range room.users {
			SendRemoteCommand(user.conn, BuildMessage(RECV_MSG, RecvTextMessage{Message: contents.Message}))
		}
	case JOIN_ROOM:
		contents := message.Contents.(JoinRoomMessage)
		if err := room.AddUser(user, contents.IsSuperUser); err != nil {
			return Message{}, err
		}
	case LEAVE_ROOM:
		if err := room.RemoveUser(user); err != nil {
			return Message{}, err
		}
	case CREATE_ROOM:
		contents := message.Contents.(CreateRoomMessage)
		_, err := server.room_manager.CreateRoom(contents.Room, contents.Capacity)
		if err != nil {
			return Message{}, err
		}
	case CLOSE_ROOM:
		contents := message.Contents.(CloseRoomMessage)
		room, err := server.room_manager.CloseRoom(contents.Room)
		if err != nil {
			return Message{}, err
		}

		for _, user := range room.users {
			SendRemoteCommand(user.conn, BuildMessage(RECV_MSG, RecvTextMessage{Message: TextMessage{Username: "SERVER", Room: room.Name, Text: "This room has been closed."}}))
			SendRemoteCommand(user.conn, BuildMessage(LEAVE_ROOM, LeaveRoomMessage{Room: room.Name}))
		}
	}

	return Message{}, nil
}

func (server *ChatServer) messagePassesTokenTest(message Message) (bool, error) {
	// Ensure any Message requiring a Token is valid
	var token string

	switch message.Command {
	case SEND_MSG:
		token = message.Contents.(SendTextMessage).Token
	case JOIN_ROOM:
		token = message.Contents.(JoinRoomMessage).Token
	case LEAVE_ROOM:
		token = message.Contents.(LeaveRoomMessage).Token
	case CREATE_ROOM:
		token = message.Contents.(CreateRoomMessage).Token
	case CLOSE_ROOM:
		token = message.Contents.(CloseRoomMessage).Token
	default:
		return true, nil
	}

	if valid, _ := server.user_manager.TokenIsValid(token); valid {
		return true, nil
	}

	// Token is provided, but is not valid
	return false, nil
}

func (server *ChatServer) getRoomIfRequired(message Message) (*Room, error) {
	var name string

	switch message.Command {
	case SEND_MSG:
		name = message.Contents.(SendTextMessage).Message.Room
	case JOIN_ROOM:
		name = message.Contents.(JoinRoomMessage).Room
	case LEAVE_ROOM:
		name = message.Contents.(LeaveRoomMessage).Room
	case CREATE_ROOM:
		name = message.Contents.(CreateRoomMessage).Room
	case CLOSE_ROOM:
		name = message.Contents.(CloseRoomMessage).Room
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
		name = message.Contents.(SendTextMessage).Message.Username
	case JOIN_ROOM:
		name = message.Contents.(JoinRoomMessage).Username
	case LEAVE_ROOM:
		name = message.Contents.(LeaveRoomMessage).Username
	default:
		return nil, nil
	}

	user, err := server.user_manager.GetUser(name)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func LoadServerConfigurationFile(filename string) (ServerConfig, error) {
	file, _ := filepath.Abs(filename)

	yamlFile, err := ioutil.ReadFile(file)
	if err != nil {
		return ServerConfig{}, err
	}

	var config ServerConfig

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return ServerConfig{}, err
	}

	return config, nil
}

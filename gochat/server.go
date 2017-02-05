package gochat

import (
	"encoding/gob"
	"errors"
	"io/ioutil"
	"net"
	"path/filepath"
	"time"

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
	storageManager, err := NewStorageManager(config.Database, logger)
	if err != nil {
		return &ChatServer{}, err
	}

	userManager, err := NewUserManager(storageManager, logger)
	if err != nil {
		return &ChatServer{}, err
	}

	roomManager, err := NewRoomManager(storageManager, logger)
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
	var empty_message Message
	encoder := gob.NewEncoder(connection)
	decoder := gob.NewDecoder(connection)

	for {
		message := Message{}
		decoder.Decode(&message)

		if message == empty_message {
			time.Sleep(time.Second * 1)
			continue
		}

		reply, err := server.HandleMessage(message, encoder)
		if err != nil {
			server.logger.Error(err)
			return
		} else if message.Command != "" {
			// Only send a reply if the command is not empty
			encoder.Encode(reply)
		}
	}
}

func (server *ChatServer) HandleMessage(message Message, encoder *gob.Encoder) (Message, error) {
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
	user, err := server.getUserIfRequired(message, encoder)
	if err != nil {
		return Message{}, err
	}

	// Interpret Message
	switch message.Command {
	case REGISTER:
		contents := message.Contents.(RegisterMessage)

		var message TextMessage
		if err := server.user_manager.CreateUser(contents.Username, contents.PasswordHash); err != nil {
			server.logger.Error("Sending back failed registration attempt")
			server.logger.Error(err)
			message = TextMessage{Username: "SERVER", Room: "SERVER", Text: "Registration Failed."}
		} else {
			server.logger.Debug("Sending back successfull registration attempt")
			message = TextMessage{Username: "SERVER", Room: "SERVER", Text: "Registration Successfull."}
		}

		return BuildMessage(RECV_MSG, RecvTextMessage{Message: message}), nil
	case AUTHENTICATE:
		contents := message.Contents.(AuthenticateMessage)
		user, err := server.user_manager.AuthenticateUser(contents.Username, contents.PasswordHash)

		var message Message
		if err != nil {
			server.logger.Debug("Sending back failed authentication attempt")
			server.logger.Error(err)
			textMessage := TextMessage{Username: "SERVER", Room: "SERVER", Text: "Authentication Failed: " + err.Error()}
			message = BuildMessage(RECV_MSG, RecvTextMessage{Message: textMessage})
		} else {
			server.logger.Debug("Sending back successful authentication attempt")
			message = BuildMessage(TOKEN, TokenMessage{Username: user.User.Username, Token: user.GetToken()})
		}

		return message, nil
	case SEND_MSG:
		contents := message.Contents.(SendTextMessage)

		for _, user := range room.users {
			SendRemoteCommand(user.encoder, BuildMessage(RECV_MSG, RecvTextMessage{Message: contents.Message}))
		}
	case JOIN_ROOM:
		//contents := message.Contents.(JoinRoomMessage)
		room.AddUser(user)
	case LEAVE_ROOM:
		var textMessage TextMessage
		if err := room.RemoveUser(user); err != nil {
			server.logger.Debug("Failed to remove user " + user.String() + " from room " + room.String())
			server.logger.Error(err)
			textMessage = TextMessage{Username: "SERVER", Room: "SERVER", Text: "Failed to remove you from: " + room.String()}
		} else {
			textMessage = TextMessage{Username: "SERVER", Room: room.String(), Text: "Successfully removed you from: " + room.String()}
		}

		return BuildMessage(RECV_MSG, RecvTextMessage{Message: textMessage}), nil
	case CREATE_ROOM:
		contents := message.Contents.(CreateRoomMessage)
		var textMessage TextMessage

		_, err := server.room_manager.CreateRoom(contents.Room, contents.Capacity)
		if err != nil {
			server.logger.Debug("Failed to create room " + contents.Room)
			server.logger.Error(err)
			textMessage = TextMessage{Username: "SERVER", Room: "SERVER", Text: "Failed to create room: " + room.String()}
		} else {
			textMessage = TextMessage{Username: "SERVER", Room: "SERVER", Text: "Successfully created room: " + room.String()}
		}

		return BuildMessage(RECV_MSG, RecvTextMessage{Message: textMessage}), nil
	case CLOSE_ROOM:
		contents := message.Contents.(CloseRoomMessage)
		room, err := server.room_manager.CloseRoom(contents.Room)
		if err != nil {
			textMessage := TextMessage{Username: "SERVER", Room: "SERVER", Text: "Failed to close room: " + contents.Room}
			return BuildMessage(RECV_MSG, RecvTextMessage{Message: textMessage}), nil
		}

		// Notify all users that the room has been closed
		for _, user := range room.users {
			SendRemoteCommand(user.encoder, BuildMessage(RECV_MSG, RecvTextMessage{Message: TextMessage{Username: "SERVER", Room: room.Name, Text: "This room has been closed."}}))
			SendRemoteCommand(user.encoder, BuildMessage(LEAVE_ROOM, LeaveRoomMessage{Room: room.Name}))
		}

		textMessage := TextMessage{Username: "SERVER", Room: "SERVER", Text: "Successfully closed room: " + room.Name}
		return BuildMessage(RECV_MSG, RecvTextMessage{Message: textMessage}), nil
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

func (server *ChatServer) getUserIfRequired(message Message, encoder *gob.Encoder) (*ServerUser, error) {
	var name string

	switch message.Command {
	case SEND_MSG:
		name = message.Contents.(SendTextMessage).Message.Username
	case JOIN_ROOM:
		name = message.Contents.(JoinRoomMessage).Username
	case LEAVE_ROOM:
		name = message.Contents.(LeaveRoomMessage).Username
	default:
		return &ServerUser{}, nil
	}

	user, err := server.user_manager.GetUser(name)
	if err != nil {
		return nil, err
	}

	// Save the encoder so we can send the user messages later
	user.encoder = encoder

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

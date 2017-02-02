package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/michael-robbins/go-and-chat/gochat"
	log "github.com/Sirupsen/logrus"
)

func printDefaults(usageTitle string, error string) {
	fmt.Fprintln(os.Stderr, usageTitle)
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, error)
}

func main() {
	connection_string := flag.String("server", "", "'hostname:port' connection string to the server")
	verbose := flag.Bool("v", false, "Enables verbose logging")
	debug := flag.Bool("debug", false, "Enables debug logging")
	logFile := flag.String("logfile", "", "Log file location, default to StdErr")
	flag.Parse()

	usageTitle := "Usage of GoChat Client:\n"

	if *connection_string == "" {
		printDefaults(usageTitle, "\nMissing -server hostname:port")
		return
	}

	// Set up logging
	if *debug == true {
		log.SetLevel(log.DebugLevel)
	} else if *verbose == true {
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(log.WarnLevel)
	}

	if *logFile != "" {
		// Attempt to either open or create the log file
		f, err := os.OpenFile(*logFile, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			printDefaults(usageTitle, "Unable to log to the request file, unable to open/create it.")
			return
		}

		log.SetOutput(f)
	}

	logger := log.WithFields(log.Fields{"type": "GoChatClient"})

	// Register all the Message struct subtypes for encoding/decoding
	gochat.RegisterStructs()

	// Create the new chat client instance
	client, _ := gochat.NewChatClient(logger)

	logger.Debug("Attempting to connect to: " + *connection_string)
	connection, err := client.Connect(*connection_string)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("Successfully connected to: " + *connection_string)

	// Spin off a thread to listen for server events
	server_messages := make(chan gochat.Message, 1)
	go client.ListenToServer(server_messages)

	// Ask the user what they want to do
	choices := []string{"Register", "Log In"}
	reader := bufio.NewReader(os.Stdin)
	for {
		choice := gochat.GetStartupChoice(choices)
		if choice == -1 {
			// The user has indicated to quit the program
			return
		}

		fmt.Print("Enter Username: ")
		username, _ := reader.ReadString('\n')

		fmt.Print("Enter Password: ")
		password, _ := reader.ReadString('\n')

		if choice == 1 {
			fmt.Print("Enter Password (again): ")
			password_again, _ := reader.ReadString('\n')

			if password != password_again {
				fmt.Println("Passwords do not match!")
				continue
			}

			if err := client.Register(username, password); err != nil {
				logger.Error(err)
			}

			fmt.Println("Registration request successfull, please raise for response before logging in!")
		} else if choice == 2 {
			// Attempt to authenticate the user
			if err := client.Authenticate(username, password); err != nil {
				logger.Error(err)
				return
			}

			break
		}
	}

	// Spin off a thread to listen for client events
	client_messages := make(chan gochat.Message, 1)
	exit_decision := make(chan int, 1)
	go client.ListenToUser(client_messages, exit_decision)

	// Listen to events on the server & client channels
EventLoop:
	for {
		select {
		case message := <-server_messages:
			// Handle the server initiated message
			if err := client.HandleServerMessage(message); err != nil {
				logger.Error(err)
			}
		case message := <-client_messages:
			// Handle the client initiated message
			if err := gochat.SendRemoteCommand(connection, message); err != nil {
				logger.Error(err)
			}
		case _ = <-exit_decision:
			break EventLoop
		}

		// Sleep for a second then check again for any server/client messages or exit decisions
		time.Sleep(time.Second)
	}

	logger.Info("Quitting!")
}

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

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
	if err := client.Connect(*connection_string); err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("Successfully connected to: " + *connection_string)

	// Spin off a thread to listen for server events
	server_disconnect := make(chan int, 1)
	server_messages := make(chan gochat.Message, 1)
	go client.ListenToServer(server_messages, server_disconnect)

	// Create the channels the client will populate
	client_messages := make(chan gochat.Message, 1)

	// Listen to events on the server & client channels.
	eventloop_exit := make(chan int, 1)
	go client.EventLoop(server_messages, client_messages, eventloop_exit)

	// Ask the user what they want to do
	choices := []string{"Register", "Log In"}
	reader := bufio.NewReader(os.Stdin)
	for {
		choice := gochat.GetStartupChoice(choices)
		if choice == -1 {
			// The user has indicated to quit the program
			fmt.Println("Quitting")
			return
		}

		if choice == 1 {
			fmt.Println("Registering User:")
		} else if choice == 2 {
			fmt.Println("Logging In:")
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

			fmt.Println("Registration request successfull, please wait for response before logging in!")
		} else if choice == 2 {
			// Attempt to authenticate the user
			if err := client.Authenticate(username, password); err != nil {
				logger.Error(err)
				return
			}

			break
		}
	}

	// Enter the main CLI menu
	client.ListenToUser(client_messages)

	// Block and wait for the eventloop and server connection to finish up
	eventloop_exit<- 1
	server_disconnect<- 1

	logger.Info("Quitting!")
}

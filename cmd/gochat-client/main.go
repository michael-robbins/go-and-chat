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
		f, err := os.OpenFile(*logFile, os.O_WRONLY | os.O_CREATE, 0755)
		if err != nil {
			printDefaults(usageTitle, "Unable to log to the request file, unable to open/create it.")
			return
		}

		log.SetOutput(f)
	}

	logger := log.WithFields(log.Fields{
		"type": "GoChatClient",
	})

	// Register all the Message struct subtypes for encoding/decoding
	gochat.RegisterStructs()

	// Create the new client instance
	client, _ := gochat.NewChatClient(logger)

	logger.Debug("Attempting to connect to: " + *connection_string)
	connection, err := client.Connect(*connection_string)
	if err != nil {
		fmt.Println(err)
		return
	}
	logger.Debug("Successfully connected to: " + *connection_string)

	// Attempt to authenticate the user
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	password, _ := reader.ReadString('\n')

	if err := client.Authenticate(username, password); err != nil {
		fmt.Println(err)
	}

	logger.Debug("Successfully sent Authentication request")

	// Spin off a thread to listen for server events
	server_messages := make(chan gochat.Message, 1)
	go client.ListenToServer(server_messages)

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
				fmt.Println(err)
			}
		case message := <-client_messages:
			// Handle the client initiated message
			if err := gochat.SendRemoteCommand(connection, message); err != nil {
				fmt.Println(err)
			}
		case _ = <-exit_decision:
			break EventLoop
		}

		// Sleep for a second then check again for any server/client messages
		time.Sleep(time.Second)
	}

	fmt.Println("Quitting.")
}

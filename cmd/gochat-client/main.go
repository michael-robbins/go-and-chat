package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/michael-robbins/go-and-chat/gochat"
)

func main() {
	connection_string := flag.String("server", "", "'hostname:port' connection string to the server")
	flag.Parse()

	if *connection_string == "" {
		fmt.Fprintln(os.Stderr, "Usage of GoChat Client:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nMissing -server hostname:port")
		return
	}

	client, _ := gochat.NewChatClient()

	connection, err := client.Connect(*connection_string)
	if err != nil {
		fmt.Println(err)
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Enter Username: ")
	username, _ := reader.ReadString('\n')

	fmt.Print("Enter Password: ")
	password, _ := reader.ReadString('\n')

	if err := client.Authenticate(username, password); err != nil {
		fmt.Println(err)
	}

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

package main

import (
	"local/gochat"
	"bufio"
	"flag"
	"fmt"
	"os"
)

func main() {
	connection_string := flag.String("server", "", "'ip:port' connection string to the server")
	flag.Parse()

	client, _ := gochat.NewChatClient()

	if err := client.Connect(*connection_string); err != nil {
		fmt.Println(err)
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
	go client.ListenToServer()

	// Listen to the users input
	client.ListenToUser()
}

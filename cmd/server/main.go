package main

import (
	"fmt"
	"flag"
	"strings"
	"local/gochat"
)

func main() {
	server := flag.String("server", "", "'ip:port' what we will listen on")
	flag.Parse()

	elements := strings.Split(*server, ":")

	if len(elements) != 2 {
		panic("Wrong format for --server")
	}

	chatServer, _ := gochat.NewChatServer(elements[0], int(elements[1]))

	fmt.Println(chatServer)
}

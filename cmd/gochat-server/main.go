package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/michael-robbins/go-and-chat/gochat"
)

func main() {
	server := flag.String("server", "", "'hostname:port' what we will listen on")
	flag.Parse()

	if *server == "" {
		fmt.Fprintln(os.Stderr, "Usage of GoChat Server:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nMissing -server hostname:port")
		return
	}

	chatServer, _ := gochat.NewChatServer()

	if err := chatServer.Listen(*server); err != nil {
		fmt.Println(err)
	}
}

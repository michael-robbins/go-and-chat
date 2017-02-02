package main

import (
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
	server := flag.String("server", "", "'hostname:port' what we will listen on")
	verbose := flag.Bool("v", false, "Enables verbose logging")
	debug := flag.Bool("debug", false, "Enables debug logging")
	logFile := flag.String("logfile", "", "Log file location, default to StdErr")
	configFile := flag.String("config", "", "Configuration file")
	flag.Parse()

	usageTitle := "Usage of GoChat Server:\n"

	if *server == "" {
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
		f, err := os.OpenFile(*logFile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			printDefaults(usageTitle, "Unable to log to the request file, unable to open/create it.")
			return
		}

		// We default to os.Stderr if this isn't called
		log.SetOutput(f)
	}

	logger := log.WithFields(log.Fields{
		"type": "GoChatServer",
	})

	// Parse the configuration file
	config, err := gochat.LoadServerConfigurationFile(*configFile)
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("Loaded configuration file")

	// Register all the Message struct subtypes for encoding/decoding
	gochat.RegisterStructs()

	// Create the server and listen for incoming connections
	chatServer, _ := gochat.NewChatServer(logger, config)

	if err := chatServer.Listen(*server); err != nil {
		logger.Error(err)
	}
}

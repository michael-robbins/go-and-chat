package gochat

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
)

func getUserInput(message string) string {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("'quit' or 'q' will exit.")
	fmt.Println("")
	fmt.Print(message)

	text, _ := reader.ReadString('\n')

	// Strip the newline character
	text = text[:len(text)-1]

	return text
}

func getClientCommandsOption(client_commands []COMMAND) int {
	number := -1

	// Inner for loop will break when we have a valid choice
	for {
		fmt.Println("Please select an option:")
		for i, command := range client_commands {
			fmt.Print(i + 1, " ", "=", " ", command, "\n")
		}

		text := getUserInput("Choice (number): ")
		if text == "quit" || text == "q" {
			return -1
		}

		var err error
		number, err = strconv.Atoi(text)

		if err != nil || number < 1 || number > len(client_commands) {
			// Invalid choice, force the for loop to iterate
			fmt.Print("Invalid choice (Valid options are: 1 -> ", len(client_commands), ").\n")
			continue
		}

		// Passed our validation, break the loop
		break
	}

	return number
}

func getRoomName() string {
	var room_name string

	for {
		if room_name != "" {
			break
		}

		room_name := getUserInput("Room to join: ")
		if room_name == "quit" || room_name == "q" {
			return ""
		}
	}

	return room_name
}

func getRoomCapacity() int {
	room_capacity := -1

	for {
		if room_capacity != -1 {
			break
		}

		text := getUserInput("Room to create: ")

		if text == "quit" || text == "q" {
			return -1
		}

		var err error
		number, err := strconv.Atoi(text)

		if err != nil || number < 1 {
			fmt.Println("Invalid choice (Only '1' -> 'MAX_INT32'.")
		}

		room_capacity = number
	}

	return room_capacity
}

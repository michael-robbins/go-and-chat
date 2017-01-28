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

	return text
}

func getClientCommandsOption(client_commands []COMMAND) int {
	number := -1

	// Inner for loop will break when we have a valid choice
	for {
		if number != -1 {
			break
		}

		fmt.Println("Please select an option:")
		for i, command := range client_commands {
			fmt.Print(i, " ", "=", " ", command, "\n")
		}

		text := getUserInput("Choice (number): ")
		fmt.Println(text)
		if text == "quit" || text == "q" {
			return -1
		}

		var err error
		number, err = strconv.Atoi(text)
		if err != nil || number < 1 || number > len(client_commands) {
			fmt.Println("Invalid choice (Only '1' -> '" + string(len(client_commands)) + "').")
		}

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

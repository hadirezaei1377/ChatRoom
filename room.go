package main

// Room is a server that lets clients chat with each other.

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type client chan<- string

type User struct {
	Username        string
	Password        string
	PrivateMessages chan privateMessage
}

type privateMessage struct {
	Sender   string
	Receiver string
	Message  string
}

var (
	entering        = make(chan client)
	leaving         = make(chan client)
	messages        = make(chan string)
	privateMessages = make(chan privateMessage)
)

func main() {

	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

func broadcaster() {
	clients := make(map[client]bool)
	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				cli <- msg
			}

		case privateMsg := <-privateMessages:
			for cli := range clients {
				if cli.User.Username == privateMsg.Receiver {
					cli <- privateMsg.Sender + " (private): " + privateMsg.Message
				} else if cli.User.Username == privateMsg.Sender {
					cli <- privateMsg.Receiver + " (private): " + privateMsg.Message
				}
			}

		case cli := <-entering:
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		if strings.Contains(msg, "(private)") {
			fmt.Fprintln(conn, "\033[33m"+msg+"\033[0m") // Print private messages in yellow
		} else {
			fmt.Fprintln(conn, msg)
		}
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan string)
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	ch <- "Welcome to the room " + who

	// Prompt the user to log in or register
	io.WriteString(conn, "Please login or register.\n")
	io.WriteString(conn, "Enter 'login' to log in or 'register' to register.\n")

	input := bufio.NewScanner(conn)
	var user User

	for input.Scan() {
		text := input.Text()

		if text == "login" {
			// Prompt for username and password
			io.WriteString(conn, "Enter username: ")
			input.Scan()
			username := input.Text()

			io.WriteString(conn, "Enter password: ")
			input.Scan()
			password := input.Text()

			// Authenticate the user
			if authenticateUser(username, password) {
				user = User{Username: username, Password: password}
				break
			} else {
				io.WriteString(conn, "Invalid username or password. Please try again.\n")
			}

		} else if text == "register" {
			// Prompt for new username and password
			io.WriteString(conn, "Enter new username: ")
			input.Scan()
			username := input.Text()

			io.WriteString(conn, "Enter new password: ")
			input.Scan()
			password := input.Text()

			// Register the user
			registerUser(username, password)
			user = User{Username: username, Password: password}
			break
		} else {
			io.WriteString(conn, "Invalid command. Please enter 'login' or 'register'.\n")
		}
	}

	messages <- who + " joined the room"
	entering <- ch

	go func() {
		for privateMsg := range user.PrivateMessages {
			if privateMsg.Receiver == user.Username {
				ch <- privateMsg.Sender + " (private): " + privateMsg.Message
			}
		}
	}()

	for input.Scan() {
		text := input.Text()
		if strings.HasPrefix(text, "/msg") {
			// Extract receiver username and message content
			parts := strings.SplitN(text, " ", 3)
			if len(parts) != 3 {
				io.WriteString(conn, "Invalid usage. Use /msg <receiver> <message>\n")
				continue
			}

			receiver := parts[1]
			message := parts[2]

			// Send private message
			privateMsg := privateMessage{
				Sender:   who,
				Receiver: receiver,
				Message:  message,
			}
			privateMessages <- privateMsg
		} else {
			messages <- who + ": " + text
		}
	}

	leaving <- ch
	messages <- who + " has left the room"
	conn.Close()
}

var registeredUsers = make(map[string]string)

func authenticateUser(username, password string) bool {
	storedPassword, ok := registeredUsers[username]
	if !ok {
		return false // User not found
	}
	return storedPassword == password
}

func registerUser(username, password string) {
	_, ok := registeredUsers[username]
	if ok {
		fmt.Println("Username already exists")
		return
	}
	registeredUsers[username] = password
	fmt.Println("User registration successful")
}

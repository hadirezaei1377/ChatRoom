package main

// Room is a server that lets clients chat with each other.

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

type client chan<- string

type privateMessage struct {
	Sender   string
	Receiver string
	Message  string
}

type Client struct {
	conn     net.Conn
	username string
}

type ChatMessage struct {
	Sender    string
	Text      string
	Timestamp time.Time // New field to track message timestamp
}

type UserMessage struct {
	ID        int // Unique identifier for the message
	Sender    string
	Content   string
	Timestamp time.Time
}

type Message struct {
	Text     string
	IsFormat bool
}

var (
	entering        = make(chan client)
	leaving         = make(chan client)
	messages        = make(chan string)
	privateMessages = make(chan privateMessage)
	chatmessages    = make(chan ChatMessage)
)

type User struct {
	Username        string
	Password        string
	PrivateMessages chan privateMessage
	IsOnline        bool
	Messages        []UserMessage // New field to store user messages
}

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
				if msg.IsFormat {
					cli <- formatMessage(msg.Text)
				} else {
					cli <- msg.Text
				}
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
			cli.User.IsOnline = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
			cli.User.IsOnline = false
		}
	}
}

func formatMessage(text string) string {

	// Assume Markdown-like syntax for simplicity
	formattedText := text

	// Apply bold formatting: *bold*
	formattedText = strings.ReplaceAll(formattedText, "*bold*", "\033[1m"+formattedText+"\033[0m")

	// Apply italic formatting: _italic_
	formattedText = strings.ReplaceAll(formattedText, "_italic_", "\033[3m"+formattedText+"\033[0m")

	// Apply underline formatting: ~underline~
	formattedText = strings.ReplaceAll(formattedText, "~underline~", "\033[4m"+formattedText+"\033[0m")

	// Apply code formatting: `code`
	formattedText = strings.ReplaceAll(formattedText, "`code`", "\033[7m"+formattedText+"\033[0m")

	// Apply multiline code formatting: ```multiline code```
	formattedText = strings.ReplaceAll(formattedText, "```multiline code```", "\033[7m"+formattedText+"\033[0m")

	return formattedText
}

func clientWriter(conn net.Conn, ch <-chan string, users map[string]User) {
	for msg := range ch {
		if strings.Contains(msg, "(private)") {
			fmt.Fprintln(conn, "\033[33m"+msg+"\033[0m") // Print private messages in yellow
		} else if strings.HasPrefix(msg, "/format ") {
			text := msg[len("/format "):]
			messages <- Message{Text: text, IsFormat: true}
		} else {
			parts := strings.SplitN(msg, ": ", 2)
			if len(parts) == 2 {
				username := parts[0]
				message := parts[1]

				user, ok := users[username]
				if !ok || !user.IsOnline {
					// Display offline user with a red dot
					fmt.Fprintln(conn, "\033[31m●\033[0m "+msg)
				} else {
					// Display online user with a green dot
					fmt.Fprintln(conn, "\033[32m●\033[0m "+msg)
				}
			} else {
				fmt.Fprintln(conn, msg)
			}
		}
	}
}

func handleConn(conn net.Conn, users map[string]User) {
	ch := make(chan string)
	go clientWriter(conn, ch, users)

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
			if authenticatedUser, ok := users[username]; ok && authenticatedUser.Password == password {
				user = authenticatedUser
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
			users[username] = User{Username: username, Password: password}
			user = users[username]
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
		} else if strings.HasPrefix(text, "/edit") || strings.HasPrefix(text, "/delete") {
			parts := strings.SplitN(text, " ", 3)
			if len(parts) != 3 {
				io.WriteString(conn, "Invalid usage. Use /edit <message_id> <new_content> or /delete <message_id>\n")
				continue
			}

			command := parts[0]
			messageIDStr := parts[1]
			newContent := parts[2]

			messageID, err := strconv.Atoi(messageIDStr)
			if err != nil {
				io.WriteString(conn, "Invalid message ID. Please provide a valid integer.\n")
				continue
			}

			userMessages := user.Messages
			found := false

			for i, msg := range userMessages {
				if msg.ID == messageID && msg.Sender == user.Username {
					found = true

					if command == "/edit" {
						// Update the message content
						userMessages[i].Content = newContent
						userMessages[i].Timestamp = time.Now()
						break
					} else if command == "/delete" {
						// Remove the message from the user's messages
						userMessages = append(userMessages[:i], userMessages[i+1:]...)
						break
					}
				}
			}

			if found {
				// Update the user's messages
				users[user.Username].Messages = userMessages

				// Broadcast the updated/deleted message to other clients
				if command == "/edit" {
					messages <- user.Username + " edited message with ID " + messageIDStr
				} else if command == "/delete" {
					messages <- user.Username + " deleted message with ID " + messageIDStr
				}
			} else {
				io.WriteString(conn, "Invalid message ID or permission denied.\n")
			}
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

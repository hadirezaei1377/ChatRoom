package main

// Room is a server that lets clients chat with each other.

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

// We have a type of one-way channel called client.
// an outgoing message channel
type client chan<- string

// In below we define some channels of channels Variably.
// a variable of type client ...client is a channel also
var (
	entering = make(chan client) // a channel of channels
	leaving  = make(chan client) // a channel of channels
	messages = make(chan string) // all incoming client messages
)

// variable messages is a channel if strings for recieving messages

func main() {
	// registering a Listener on localhost
	listener, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}

	// in a goroutine we call broadcaster function, this function waits for clients to send messages.
	// goroutine definition
	go broadcaster()
	// when main function call broadcaster function , waits in this infinite loop for connection
	for {
		// accepting connection done with listener.Accept
		conn, err := listener.Accept()
		// error handling says that if there is not error in this , connect connection to another function called handleConn by connection input.
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

// a function for send any messages to any client
func broadcaster() {
	// a map of all connected clients to server now
	clients := make(map[client]bool)
	// an infinite loop which keeps server alive forever
	for {
		select {
		// in this case ,if recieve a message , read it and send to all client in your map
		case msg := <-messages:

			// If we encounter a problem in sending a message to a client, it means that it receives the message slowly or the process of sending the message fails.
			// All our code will be blocked and other clients will not receive any messages.
			for cli := range clients {
				cli <- msg // What if a reader fails or is slow?
			}

			// in this case we are waiting for a client for regisrering and check it and add to clients list.
			// we should create a channel in this channel.
		case cli := <-entering:
			clients[cli] = true

			// this case using for sign out a client
		case cli := <-leaving:
			delete(clients, cli)
			close(cli)
		}
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	ch <- "Welcome to the room " + who // greetings to the client
	messages <- who + " join the room"
	entering <- ch

	input := bufio.NewScanner(conn)
	for input.Scan() {
		messages <- who + ": " + input.Text()
	}
	// NOTE: ignoring potential errors from input.Err()

	leaving <- ch
	messages <- who + " has left the room"
	conn.Close()
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

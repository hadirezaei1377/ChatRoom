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
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli <- msg // What if a reader fails or is slow?
			}

		case cli := <-entering:
			clients[cli] = true

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

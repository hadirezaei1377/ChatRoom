package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {
	// Dial is use for calling to port for making connection
	// connection must be accepted in room.go
	// based on below if connection be accepted we have no error
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	// define a goroutine that write anything that recieve on connection on stdout
	go func() {
		io.Copy(os.Stdout, conn)
		log.Println("done")
		// when do its duty make a signal on done channel
		done <- struct{}{}
	}()
	mustCopy(conn, os.Stdin)
	conn.Close()
	<-done
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}

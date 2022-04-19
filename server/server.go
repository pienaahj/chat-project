package main

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/pienaahj/grpc-todd/chat-project/client/chatpb"
	"google.golang.org/grpc"
)

// --------------------  design notes  ---------------------------------------------
// you are not allowed to send and recv on more than one gotoutine at a time -- go prpc

// Connection defines a new Connection struct
type Connection struct {
	conn chatpb.Chat_ChatServer  	// the normal connection in grpc generated file
	send chan *chatpb.ChatMessage 	// channel to send the message on
	quit chan struct{} 				// quit signal
}

// NewConnection creates a new connection and starts the start method in another go routine
func NewConnection(conn chatpb.Chat_ChatServer) *Connection {
	c := &Connection{
		conn: conn,
		send: make(chan *chatpb.ChatMessage),
		quit: make(chan struct{}),
	}
	go c.start()
	return c
}

// Close closes a ccnnection
func (c *Connection) Close() error {
	close(c.quit)
	close(c.send)
	return nil
}

// Send sends a msg on a channel
func (c *Connection) Send(msg *chatpb.ChatMessage) {
	defer func () {
		//Ignore any errors about sending on a channel - trying to send on a closed channel
		recover()
	} ()
	// load the message onto the send channel 
	//- if msg channel gets closed, it will panic and run the recover()
	c.send <- msg  
}

// start sets up the monitoring of the messages received and decides what to do with it
func (c *Connection) start() {
	running := true
	for running {
		select {
		case msg := <-c.send: // if there is a message on the message channel send it
			c.conn.Send(msg)  // Ignore the error they just 
		case <-c.quit:
			running = false  // breaks out of loop when Close() closes the quit channel 
		}
	}
}

// GetMessages receives the messages through a send only channel
func (c *Connection) GetMessages(broadcast chan<- *chatpb.ChatMessage) error {
	for {
		msg, err := c.conn.Recv()
		if err == io.EOF {
			c.Close()
			return nil
		} else if err != nil {
			c.Close()
			return err
		}
		// once you have a message, send it to all the connections
		go func(msg *chatpb.ChatMessage) {
			select {
			case broadcast <- msg: // this will block until it is taken off, but does not hamper receiving a message
			case <-c.quit:
			}
		}(msg)
	}
}

// ChatServer create a chat server type
type ChatServer struct {
		chatpb.UnimplementedChatServer // needs this to implement the interface

		broadcast chan *chatpb.ChatMessage
		quit 		chan struct{}
		connections []*Connection
		connLock sync.Mutex
} 

// NewChatServer implements a chat server
func NewChatServer() *ChatServer {
	srv := &ChatServer{
		broadcast: make(chan *chatpb.ChatMessage),
		quit: make(chan struct{}),
	}
	go srv.start()
	return srv
}

// Close closes the chat server
func (c *ChatServer) Close() error {
	close(c.quit)
	return nil
}

// start controls the chat server operations
func (c *ChatServer) start() {
	running := true
	for running {
		select {
		case msg := <-c.broadcast:
			c.connLock.Lock()
			for _, v := range c.connections {
				go v.Send(msg) // not to lock up the complete server
			}
			c.connLock.Unlock()
		case <- c.quit:
			running = false
		}
	}
}

// Chat implements grpc Chat call
func (c *ChatServer) Chat(stream chatpb.Chat_ChatServer) error {
	conn := NewConnection(stream)

	c.connLock.Lock()
	c.connections = append(c.connections, conn)
	c.connLock.Unlock()

	err := conn.GetMessages(c.broadcast)
	c.connLock.Lock()
	for i, v := range c.connections {
		if v == conn {
			c.connections = append(c.connections[:i], c.connections[i+1:]... )
		}
	}
	c.connLock.Unlock()

	return err
}


func main() {
	// listen to port 8080
	lst, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	fmt.Println("Now litening on port 8080...")
	s := grpc.NewServer()

	srv := NewChatServer()
	chatpb.RegisterChatServer(s, srv)
	err = s.Serve(lst)
	if err != nil {
		panic(err)
	}
}
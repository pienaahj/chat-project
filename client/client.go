package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pienaahj/grpc-todd/chat-project/client/chatpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// create a context
	ctx := context.Background()

	//check that the commandline arguments are valid
	if len(os.Args) != 3 {
		fmt.Println("Must have a URL to connect to as first argument, and a username as a second argument")
	}

	// create a connection by dialing it with a context
	conn, err := grpc.DialContext(ctx, os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials().Clone()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// create a new chat client on the connection
	chatClient := chatpb.NewChatClient(conn)

	// define the chat stream
	stream, err := chatClient.Chat(ctx)
	if err != nil {
		panic(err)
	}

	// create go routine to handle bothway streams
	waitc := make(chan struct{})
	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			} else if err != nil {
				panic(err)
			}
			fmt.Println(msg.User + ": " + msg.Message)
		}
	}()

	fmt.Println("Connetcion established, type \"quit\" or use ctrl+c to exit")
	// create a scanner to read command line
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg := scanner.Text()
		if msg == "quit" {
			err := stream.CloseSend()
			if err != nil {
				panic(err)
			}
			break
		}

		// send the currently read message
		err := stream.Send(&chatpb.ChatMessage{
			User: os.Args[2],
			Message: msg,
		})
		if err != nil {
			panic(err)
		}
	}

	// wait for the goroutine to finish
	<-waitc 
}
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var (
	addr   = flag.String("addr", "localhost:8080", "http service address")
	reader = bufio.NewReader(os.Stdin)
)

func readMessages(c *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, msg, err := c.ReadMessage()
		if err != nil {
			log.Printf("Error while reading message: %s", err)
			return
		}
		log.Printf("%s", msg)
	}
}

func writeMessagesUntilExit(c *websocket.Conn) {
	fmt.Println("Connected!\nEnter messages to send to the server, type 'exit' to close the connection:")
	for {
		fmt.Print("> ")
		msg, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Unexpected IO error: %s", err)
			return
		}

		if msg == "exit\n" {
			break
		}

		err = c.WriteMessage(websocket.TextMessage, []byte(msg))
		if err != nil {
			log.Printf("Error while writing message...\nMessage: %s\n, Error:%s", msg, err)
			return
		}
	}
}

func connect(url string, username string) *websocket.Conn {
	c, res, err := websocket.DefaultDialer.Dial(url, http.Header{
		"Username": []string{username},
	})
	if err != nil {
		log.Print("Error while connecting to the server...")
		if res != nil {
			msgBytes, err := io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				log.Fatal("error while reading response body: ", err)
			}
			log.Fatal(string(msgBytes))
		} else {
			log.Fatal("Unexpected error while connecting to server:", err)
		}
	}

	return c
}

func main() {
	flag.Parse()
	fmt.Print("Enter a username, or leave blank to connect as anonymous: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Unexpected IO error: %s", err)
		return
	}
	if username == "\n" {
		username = "anonymous"
	} else {
		username = username[:len(username)-1]
	}

	log.Printf("Connecting to %s, username: %s...", *addr, username)

	u := "ws://" + *addr + "/ws"
	c := connect(u, username)
	defer c.Close()

	done := make(chan struct{})

	go readMessages(c, done)
	writeMessagesUntilExit(c)

	err = c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		log.Println("write close:", err)
		return
	}
	<-done
}

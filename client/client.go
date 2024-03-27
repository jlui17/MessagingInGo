package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gorilla/websocket"
)

var (
	addr = flag.String("addr", "localhost:8080", "http service address")
)

func readMessages(c *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			return
		}
		log.Printf("recv: %s", message)
	}
}

func writeMessagesUntilExit(c *websocket.Conn) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter messages to send to the server, type 'exit' to close the connection:")
	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		if text == "exit\n" {
			break
		}

		err := c.WriteMessage(websocket.TextMessage, []byte(text))
		if err != nil {
			log.Println("write:", err)
			return
		}
	}
}

func main() {
	flag.Parse()
	log.Printf("connecting to %s", *addr)

	u := "ws://" + *addr + "/ws"
	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
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

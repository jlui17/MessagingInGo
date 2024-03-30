package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/jlui17/MessagingInGo/common"
)

var (
	addr = flag.String("addr", "localhost:8080", "http service address")

	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // allow any
		},
	}

	clients   = map[string]*Client{}
	broadcast = make(chan []byte)
	lock      = sync.RWMutex{}
	nAnon     = 0
)

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	username  string
	broadcast chan []byte
}

func (c *Client) sendMessages() {
	defer func() {
		lock.Lock()
		defer lock.Unlock()

		delete(clients, c.username)
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		c.broadcast <- []byte(fmt.Sprintf("%s: %s", c.username, message))
	}
}

func (c *Client) receiveMessages() {
	defer c.conn.Close()

	for {
		message, ok := <-c.send
		if !ok {
			c.conn.WriteMessage(websocket.CloseMessage, []byte{})
			return
		}

		w, err := c.conn.NextWriter(websocket.TextMessage)
		if err != nil {
			log.Printf("Error while getting next writer for %s: %v\nClosing connection...", c.username, err)
			return
		}
		w.Write(message)

		n := len(c.send)
		for i := 0; i < n; i++ {
			w.Write(<-c.send)
		}

		if err := w.Close(); err != nil {
			return
		}
	}
}

func getAndValidateUsername(h http.Header) (string, error) {
	usernameHeader, exists := h["Username"]
	if !exists {
		log.Println("No username provided, closing connection...")
		return "", common.ErrNoUsername
	}
	if len(usernameHeader) != 1 {
		log.Println("Invalid username provided, closing connection...")
		return "", common.ErrInvalidUsername
	}

	un := usernameHeader[0]
	if un == "anonymous" {
		lock.Lock()
		un = fmt.Sprintf("anonymous%d", nAnon)
		nAnon++
		lock.Unlock()
	}

	lock.RLock()
	_, exists = clients[un]
	lock.RUnlock()
	if exists {
		log.Println("Username already exists, closing connection...")
		return "", common.ErrUsernameExists
	}

	return un, nil
}

func acceptConnection(w http.ResponseWriter, r *http.Request) {
	username, err := getAndValidateUsername(r.Header)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}

	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		username:  username,
		broadcast: broadcast,
	}
	clients[username] = client
	go client.receiveMessages()
	go client.sendMessages()
}

func broadcastMessages() {
	for {
		msg := <-broadcast

		for id, client := range clients {
			select {
			case client.send <- msg:
			default:
				close(client.send)
				lock.Lock()
				delete(clients, id)
				lock.Unlock()
			}
		}
	}
}

func main() {
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Welcome to the WebSocket server!"))
	})
	http.HandleFunc("/ws", acceptConnection)
	go broadcastMessages()
	log.Printf("Server started on %s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

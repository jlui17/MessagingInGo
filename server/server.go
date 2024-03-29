package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
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

	clients   = map[uint64]*Client{}
	broadcast = make(chan []byte)
	usernames = map[string]bool{}
	lock      = sync.RWMutex{}
)

type Client struct {
	conn      *websocket.Conn
	send      chan []byte
	username  string
	id        uint64
	broadcast chan []byte
}

func (c *Client) sendMessages() {
	defer func() {
		lock.Lock()
		defer lock.Unlock()

		delete(clients, c.id)
		c.conn.Close()
		delete(usernames, c.username)
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

	lock.RLock()
	defer lock.RUnlock()
	un := usernameHeader[0]
	if _, exists = usernames[un]; un != "anonymous" && exists {
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

	lock.Lock()
	defer lock.Unlock()
	id := rand.Uint64()
	for _, exists := clients[id]; exists; {
		id = rand.Uint64()
	}

	client := &Client{
		conn:      conn,
		send:      make(chan []byte, 256),
		username:  username,
		id:        id,
		broadcast: broadcast,
	}
	clients[id] = client
	usernames[username] = true
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
				delete(usernames, client.username)
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

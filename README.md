# MessagingInGo
A Go websocket client and server for sending and receiving messages between connected clients.

## How to Use
### Server
To start the server from the root directory, run:
```
go run server/server.go
```

### Client
To start a client from the root directory, run:
```
go run client/client.go -t <server_address>
```
- By default, <server_address> is: localhost:8080

## Implementation Details
### Client
The client runs on two event loops...
1. `readMessages()`
   - reads messages from the websocket connection and prints them to the terminal
2. `writeMessagesUntilExit()`
   - reads the next user input and sends it to the server

The client also passes a `Username` header specifying the username to be displayed on it's messages.

### Server
The server represents a connection to a client through the `Client` struct. The client struct is responsible for...
1. reading incoming messages from the websocket.
2. passing the messages to the server's `broadcast` channel.

Clients use the `broadcast` channel to pass messages to the server. The server runs on two event loops...
1. `acceptConnection()`
   - accepts connections sent to the `/ws` endpoint.
   - creates a `Client` and saves it in a map of connections.
2. `broadcastMessages()`
   - reads messages from the `broadcast` channel, formats them, and sends them to every connection. 

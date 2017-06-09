package main

import "log"
import "net/http"
import "github.com/gorilla/websocket"

type room struct {
	forward chan []byte
	join    chan *client
	leave   chan *client
	clients map[*client]bool
}

// newRoom returns instantly usable chat room
func newRoom() *room {
	return &room{
		forward: make(chan []byte),
		join:    make(chan *client),
		leave:   make(chan *client),
		clients: make(map[*client]bool),
	}
}

func (r *room) run() {
	for {
		select {
		case client := <-r.join:
			// login
			r.clients[client] = true
		case client := <-r.leave:
			// logout
			delete(r.clients, client)
			close(client.send)
		case msg := <-r.forward:
			// send message to all clients
			for client := range r.clients {
				select {
				case client.send <- msg:
					// send message
				default:
					// failed to send message
					delete(r.clients, client)
					close(client.send)
				}
			}
		}
	}
}

const socketBufferSize = 1024
const messageBufferSize = 256

var upgrader = &websocket.Upgrader{ReadBufferSize: socketBufferSize, WriteBufferSize: socketBufferSize}

func (r *room) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	socket, err := upgrader.Upgrade(w, req, nil)
	if err != nil {
		log.Fatal("ServeHTTP:", err)
		return
	}
	client := &client{
		socket: socket,
		send:   make(chan []byte, messageBufferSize),
		room:   r,
	}
	r.join <- client
	defer func() { r.leave <- client }()
	go client.write()
	client.read()
}

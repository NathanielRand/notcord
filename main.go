package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

type Message struct {
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "message", "join", "leave", or "user_count"
	UserCount int       `json:"user_count,omitempty"`
}

type Client struct {
	conn     *websocket.Conn
	username string
	send     chan Message
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) run() {
	log.Printf(">>> HUB: run() started, listening for events...")

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			clientCount := len(h.clients)
			h.mu.Unlock()
			log.Printf(">>> HUB REGISTER: Client %s registered. Total clients: %d", client.username, clientCount)

			// Send join and user_count messages in a goroutine to avoid blocking
			go func(username string, count int) {
				// Notify others that someone joined
				joinMsg := Message{
					Username:  username,
					Content:   "joined the chat",
					Timestamp: time.Now(),
					Type:      "join",
				}
				log.Printf(">>> HUB: Sending join message for %s", username)
				h.broadcast <- joinMsg

				// Send updated user count to all clients
				countMsg := Message{
					Type:      "user_count",
					UserCount: count,
					Timestamp: time.Now(),
				}
				log.Printf(">>> HUB: Sending user_count message (count: %d)", count)
				h.broadcast <- countMsg
			}(client.username, clientCount)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				clientCount := len(h.clients)
				log.Printf(">>> HUB UNREGISTER: Client %s unregistered. Total clients: %d", client.username, clientCount)
				close(client.send)

				h.mu.Unlock()

				// Send leave and user_count messages in a goroutine
				go func(username string, count int) {
					// Notify that someone left
					leaveMsg := Message{
						Username:  username,
						Content:   "left the chat",
						Timestamp: time.Now(),
						Type:      "leave",
					}
					log.Printf(">>> HUB: Sending leave message for %s", username)
					h.broadcast <- leaveMsg

					// Send updated user count
					countMsg := Message{
						Type:      "user_count",
						UserCount: count,
						Timestamp: time.Now(),
					}
					log.Printf(">>> HUB: Sending user_count message (count: %d)", count)
					h.broadcast <- countMsg
				}(client.username, clientCount)
			} else {
				h.mu.Unlock()
			}

		case message := <-h.broadcast:
			h.mu.RLock()
			clientCount := len(h.clients)
			h.mu.RUnlock()

			log.Printf(">>> HUB BROADCAST START: Message type=%s from=%s to %d clients",
				message.Type, message.Username, clientCount)

			if message.Type == "message" {
				log.Printf(">>> HUB BROADCAST: Message content: '%s'", message.Content)
			}

			h.mu.RLock()
			successCount := 0
			failCount := 0

			for client := range h.clients {
				select {
				case client.send <- message:
					successCount++
					if message.Type != "user_count" {
						log.Printf(">>> HUB: ✓ Queued message for client %s", client.username)
					}
				default:
					failCount++
					log.Printf(">>> HUB: ✗ Failed to queue message for %s (channel full/closed)", client.username)
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

			log.Printf(">>> HUB BROADCAST COMPLETE: Success=%d, Failed=%d", successCount, failCount)
		}
	}
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		log.Printf(">>> CLIENT %s: readPump ending, unregistering client", c.username)
		hub.unregister <- c
		c.conn.Close()
	}()

	// Set initial read deadline - 5 minutes
	c.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		log.Printf(">>> CLIENT %s: Received pong, extending deadline", c.username)
		c.conn.SetReadDeadline(time.Now().Add(300 * time.Second))
		return nil
	})

	log.Printf(">>> CLIENT %s: readPump started, waiting for messages...", c.username)

	for {
		log.Printf(">>> CLIENT %s: Waiting for next message...", c.username)
		_, messageBytes, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf(">>> CLIENT %s: WebSocket error: %v", c.username, err)
			} else {
				log.Printf(">>> CLIENT %s: Connection closed: %v", c.username, err)
			}
			break
		}

		log.Printf(">>> CLIENT %s: ✓ RAW MESSAGE RECEIVED: %s", c.username, string(messageBytes))

		// Reset read deadline on each message
		c.conn.SetReadDeadline(time.Now().Add(300 * time.Second))

		var msg Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Printf(">>> CLIENT %s: ✗ Error unmarshaling message: %v", c.username, err)
			continue
		}

		msg.Username = c.username
		msg.Timestamp = time.Now()
		msg.Type = "message"

		log.Printf(">>> CLIENT %s: ✓ Parsed message - Content: '%s', queuing for broadcast", c.username, msg.Content)
		hub.broadcast <- msg
		log.Printf(">>> CLIENT %s: ✓ Message queued to broadcast channel", c.username)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
		log.Printf(">>> CLIENT %s: writePump ended", c.username)
	}()

	log.Printf(">>> CLIENT %s: writePump started, waiting to send messages...", c.username)

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				log.Printf(">>> CLIENT %s: Send channel closed, sending close message", c.username)
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			log.Printf(">>> CLIENT %s: Got message from send channel - Type: %s, From: %s, Content: '%s'",
				c.username, message.Type, message.Username, message.Content)

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf(">>> CLIENT %s: ✗ Error writing JSON to WebSocket: %v", c.username, err)
				return
			}

			log.Printf(">>> CLIENT %s: ✓ Successfully sent message to WebSocket", c.username)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf(">>> CLIENT %s: Ping failed, closing connection", c.username)
				return
			}
		}
	}
}

func handleWebSocket(hub *Hub, w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		username = "Anonymous"
	}

	log.Printf(">>> WEBSOCKET: New connection request from: %s", username)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf(">>> WEBSOCKET: Upgrade error for %s: %v", username, err)
		return
	}

	log.Printf(">>> WEBSOCKET: Successfully upgraded connection for %s", username)

	client := &Client{
		conn:     conn,
		username: username,
		send:     make(chan Message, 256),
	}

	log.Printf(">>> WEBSOCKET: Registering client %s with hub", username)
	hub.register <- client
	log.Printf(">>> WEBSOCKET: Client %s registration queued", username)

	log.Printf(">>> WEBSOCKET: Starting writePump goroutine for %s", username)
	go client.writePump()

	log.Printf(">>> WEBSOCKET: Starting readPump goroutine for %s", username)
	go client.readPump(hub)

	log.Printf(">>> WEBSOCKET: Both goroutines started for %s", username)
}

func main() {
	log.Println(">>> MAIN: Creating hub...")
	hub := newHub()

	log.Println(">>> MAIN: Starting hub goroutine...")
	go hub.run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(hub, w, r)
	})

	http.Handle("/", http.FileServer(http.Dir("./static")))

	log.Println("Server starting on :8080")
	log.Println("Access the chat at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	ErrInvalidRoomId = fmt.Errorf("invalid room_id or no room id provided")
)

// MessageType represents different types of WebSocket messages
type MessageType string

const (
	TypeJoin  MessageType = "join"
	TypeCode  MessageType = "code"
	TypeChat  MessageType = "chat"
	TypeSync  MessageType = "sync"
	TypeError MessageType = "error"
)

// WebSocketMessage represents the structure of messages
type WebSocketMessage struct {
	Type    MessageType `json:"type"`
	RoomID  string      `json:"room_id"`
	Content interface{} `json:"content"`
	UserID  string      `json:"user_id"`
}

// Room represents a WebSocket room with a maximum of 2 participants
type Room struct {
	ID         string
	Clients    map[*Client]bool
	Broadcast  chan *WebSocketMessage
	Register   chan *Client
	Unregister chan *Client
	CodeState  string // Current code state
	Problem    string // Current problem
	CreatedAt  time.Time
	mu         sync.RWMutex
}

// Client represents a connected user
type Client struct {
	Conn     *websocket.Conn
	Room     *Room
	UserID   string
	Role     string // "Author" or "Collaborator"
	SendChan chan *WebSocketMessage
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// RoomManager manages all active rooms with cleanup
type RoomManager struct {
	Rooms    map[string]*Room
	mu       sync.RWMutex
	maxRooms int
}

var roomManager = &RoomManager{
	Rooms:    make(map[string]*Room),
	maxRooms: 100, // Adjust based on your server capacity
}

// CreateRoom creates a new room with improved initialization
func CreateRoom(roomID string) *Room {
	room := &Room{
		ID:         roomID,
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan *WebSocketMessage, 100), // Buffered channel
		Register:   make(chan *Client, 2),
		Unregister: make(chan *Client, 2),
		CreatedAt:  time.Now(),
	}
	go room.Run()
	return room
}

// Run handles the room's WebSocket operations with improved error handling
func (r *Room) Run() {
	ticker := time.NewTicker(30 * time.Second) // Periodic cleanup
	defer ticker.Stop()

	for {
		select {
		case client := <-r.Register:
			r.mu.Lock()
			if len(r.Clients) < 2 {
				r.Clients[client] = true
				// Send current state to new client
				syncMsg := &WebSocketMessage{
					Type:    TypeSync,
					Content: r.CodeState,
					RoomID:  r.ID,
				}
				client.SendChan <- syncMsg
			} else {
				client.SendChan <- &WebSocketMessage{
					Type:    TypeError,
					Content: "Room is full",
				}
				close(client.SendChan)
			}
			r.mu.Unlock()

		case client := <-r.Unregister:
			r.mu.Lock()
			if _, ok := r.Clients[client]; ok {
				delete(r.Clients, client)
				close(client.SendChan)
			}
			r.mu.Unlock()

		case message := <-r.Broadcast:
			r.mu.Lock()
			if message.Type == TypeCode {
				r.CodeState = message.Content.(string)
			}

			for client := range r.Clients {
				select {
				case client.SendChan <- message:
				default:
					close(client.SendChan)
					delete(r.Clients, client)
				}
			}
			r.mu.Unlock()

		case <-ticker.C:
			r.mu.Lock()
			// Cleanup inactive clients
			for client := range r.Clients {
				if err := client.Conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second)); err != nil {
					client.Conn.Close()
					delete(r.Clients, client)
				}
			}
			r.mu.Unlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections with improved client handling
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// roomID := r.FormValue("room_id")
	roomID := r.URL.Query().Get("room_id")
	log.Println("roomID=", roomID)
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create new client with buffered channel
	client := &Client{
		Conn:     conn,
		SendChan: make(chan *WebSocketMessage, 100),
		UserID:   generateUserID(), // Implement this helper function
	}

	roomManager.mu.Lock()
	room, exists := roomManager.Rooms[roomID]
	if !exists {
		if len(roomManager.Rooms) >= roomManager.maxRooms {
			roomManager.cleanupOldRooms()
		}
		room = CreateRoom(roomID)
		roomManager.Rooms[roomID] = room
		client.Role = "Author"
	} else {
		client.Role = "Collaborator"
	}
	client.Room = room
	roomManager.mu.Unlock()

	// Write something to the client
	client.SendChan <- &WebSocketMessage{
		Type:    TypeSync,
		RoomID:  client.Room.ID,
		Content: `Are you still watching??`,
		UserID:  client.UserID,
	}

	// Start client message handlers
	go client.writePump()
	go client.readPump()

	// Register client with room
	room.Register <- client
}

func generateUserID() string {
	return uuid.New().String()
}

// Client message reading routine
func (c *Client) readPump() {
	defer func() {
		c.Room.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(4096) // Limit message size
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		var msg WebSocketMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}
		msg.UserID = c.UserID
		c.Room.Broadcast <- &msg
	}
}

// Client message writing routine
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.SendChan:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			json.NewEncoder(w).Encode(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Cleanup old rooms to manage server resources
func (rm *RoomManager) cleanupOldRooms() {
	threshold := time.Now().Add(-24 * time.Hour)
	for id, room := range rm.Rooms {
		if room.CreatedAt.Before(threshold) && len(room.Clients) == 0 {
			delete(rm.Rooms, id)
		}
	}
}

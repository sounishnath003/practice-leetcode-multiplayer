package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	ErrInvalidRoomId = fmt.Errorf("invalid room_id or no room id provided")
)

// Room represents a WebSocket room with a maximum of 2 participants.
type Room struct {
	ID           string
	Clients      map[*websocket.Conn]string // Map of WebSocket connections to user roles (Author/Collaborator)
	Broadcast    chan []byte
	Register     chan *websocket.Conn
	Unregister   chan *websocket.Conn
	CurrentState []byte // Stores the current state of the document
	mu           sync.Mutex
}

// RoomManager manages all active rooms.
type RoomManager struct {
	Rooms map[string]*Room
	mu    sync.Mutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var roomManager = &RoomManager{
	Rooms: make(map[string]*Room),
}

// CreateRoom creates a new room with a unique ID.
func CreateRoom(roomID string, conn *websocket.Conn) *Room {
	room := &Room{
		ID:           roomID,
		Clients:      make(map[*websocket.Conn]string),
		Broadcast:    make(chan []byte),
		Register:     make(chan *websocket.Conn),
		Unregister:   make(chan *websocket.Conn),
		CurrentState: []byte{}, // Initialize with an empty state
	}
	room.Clients[conn] = "Author" // The first connection is the Author
	go room.Run()
	return room
}

// Run handles the room's WebSocket connections and messages.
func (r *Room) Run() {
	for {
		select {
		case conn := <-r.Register:
			r.mu.Lock()
			if len(r.Clients) < 2 {
				r.Clients[conn] = "Collaborator" // New user is a Collaborator
				log.Printf("Collaborator joined room %s", r.ID)

				// Send the current state of the document to the new user
				if len(r.CurrentState) > 0 {
					err := conn.WriteMessage(websocket.TextMessage, r.CurrentState)
					if err != nil {
						log.Printf("Error sending current state: %v", err)
					}
				}
			} else {
				log.Printf("Room %s is full", r.ID)
				conn.Close()
			}
			r.mu.Unlock()

		case conn := <-r.Unregister:
			r.mu.Lock()
			if _, ok := r.Clients[conn]; ok {
				delete(r.Clients, conn)
				conn.Close()
				log.Printf("Client disconnected from room %s", r.ID)
			}
			r.mu.Unlock()

		case message := <-r.Broadcast:
			r.mu.Lock()
			// Update the current state of the document
			r.CurrentState = message

			// Broadcast the message to all clients
			for conn := range r.Clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error sending message: %v", err)
					conn.Close()
					delete(r.Clients, conn)
				}
			}
			r.mu.Unlock()
		}
	}
}

// HandleWebSocket handles WebSocket connections and assigns them to rooms.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := r.FormValue("room_id")
	// roomID := r.URL.Query().Get("room_id")
	log.Println("roomID====", roomID)
	
	if roomID == "" {
		http.Error(w, "room_id is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	roomManager.mu.Lock()
	room, exists := roomManager.Rooms[roomID]
	if !exists {
		// Create a new room and assign the first user as the Author
		room = CreateRoom(roomID, conn)
		roomManager.Rooms[roomID] = room
		log.Printf("Room %s created by Author", roomID)
	} else {
		// Add the new user to the existing room
		room.Register <- conn
	}
	roomManager.mu.Unlock()

	go func() {
		defer func() { room.Unregister <- conn }()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Read error: %v", err)
				break
			}
			room.Broadcast <- message
		}
	}()
}

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
	TypeLeave MessageType = "leave"
	TypeError MessageType = "error"
	// Language change message type
	TypeLanguageChange MessageType = "language_change"
	// WebRTC signaling message types
	TypeOffer        MessageType = "offer"
	TypeAnswer       MessageType = "answer"
	TypeIceCandidate MessageType = "ice-candidate"
)

type UserInfo struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

// WebSocketMessage represents the structure of messages
type WebSocketMessage struct {
	Type               MessageType `json:"type"`
	RoomID             string      `json:"room_id"`
	Content            interface{} `json:"content"`
	ProblemTitle       string      `json:"problem_title"`
	ProblemDescription string      `json:"problem_description"`
	QuestionMeta       string      `json:"question_meta,omitempty"`
	QuestionHints      string      `json:"question_hints,omitempty"`
	QuestionSnippets   string      `json:"question_snippets,omitempty"`
	UserID             string      `json:"user_id"`
	Role               string      `json:"role"`
	ConnectedUsers     []UserInfo  `json:"connected_users,omitempty"`
	Language           string      `json:"language,omitempty"`
	// WebRTC specific fields
	TargetUserID string      `json:"target_user_id,omitempty"`
	SDP          interface{} `json:"sdp,omitempty"`
	IceCandidate interface{} `json:"ice_candidate,omitempty"`
}

// Room represents a WebSocket room with a maximum of 2 participants
type Room struct {
	ID                 string
	Clients            map[*Client]bool
	Broadcast          chan *WebSocketMessage
	Register           chan *Client
	Unregister         chan *Client
	ProblemTitle       string // Current problem title
	ProblemDescription string // Current problem description
	QuestionMeta       string // Current question meta HTML
	QuestionHints      string // Current question hints HTML
	QuestionSnippets   string // Current question snippets HTML
	CodeState          string // Current code state
	CurrentLanguage    string // Current programming language
	CreatedAt          time.Time
	mu                 sync.RWMutex
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
		Register:   make(chan *Client, 5),
		Unregister: make(chan *Client, 5),
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

				// Get list of existing users
				var connectedUsers []UserInfo
				for c := range r.Clients {
					if c.UserID != client.UserID {
						connectedUsers = append(connectedUsers, UserInfo{
							UserID: c.UserID,
							Role:   c.Role,
						})
					}
				}

				// Send current state to new client
				syncMsg := &WebSocketMessage{
					Type:               TypeSync,
					Content:            r.CodeState,
					ProblemTitle:       r.ProblemTitle,
					ProblemDescription: r.ProblemDescription,
					QuestionMeta:       r.QuestionMeta,
					QuestionHints:      r.QuestionHints,
					QuestionSnippets:   r.QuestionSnippets,
					RoomID:             r.ID,
					UserID:             client.UserID,
					Role:               client.Role,
					ConnectedUsers:     connectedUsers,
					Language:           r.CurrentLanguage,
				}
				client.SendChan <- syncMsg

				// Broadcast join event
				joinMsg := &WebSocketMessage{
					Type:   TypeJoin,
					UserID: client.UserID,
					Role:   client.Role,
					RoomID: r.ID,
				}
				r.Broadcast <- joinMsg
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

				// Broadcast leave event
				leaveMsg := &WebSocketMessage{
					Type:   TypeLeave,
					UserID: client.UserID,
					Role:   client.Role,
					RoomID: r.ID,
				}
				r.Broadcast <- leaveMsg
			}
			r.mu.Unlock()

		case message := <-r.Broadcast:
			r.mu.Lock()
			if message.Type == TypeCode {
				r.CodeState = message.Content.(string)
				// Persist granular question state if present in the message
				if message.ProblemTitle != "" {
					r.ProblemTitle = message.ProblemTitle
				}
				if message.ProblemDescription != "" {
					r.ProblemDescription = message.ProblemDescription
				}
				if message.QuestionMeta != "" {
					r.QuestionMeta = message.QuestionMeta
				}
				if message.QuestionHints != "" {
					r.QuestionHints = message.QuestionHints
				}
				if message.QuestionSnippets != "" {
					r.QuestionSnippets = message.QuestionSnippets
				}
			} else if message.Type == TypeLanguageChange {
				r.CurrentLanguage = message.Language
			}

			for client := range r.Clients {
				// Don't send code updates or language changes back to the sender
				if client.UserID == message.UserID && (message.Type == TypeCode || message.Type == TypeLanguageChange) {
					continue
				}

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
		UserID:   generateUserID(),
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
		// Check if there are any existing clients in the room
		if len(room.Clients) == 0 {
			client.Role = "Author"
		} else {
			client.Role = "Collaborator"
		}
	}
	client.Room = room
	roomManager.mu.Unlock()

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

	c.Conn.SetReadLimit(512 * 1024) // Limit message size to 512KB
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

		// Handle WebRTC signaling messages
		if msg.Type == TypeOffer || msg.Type == TypeAnswer || msg.Type == TypeIceCandidate {
			// Find target client in the room
			c.Room.mu.RLock()
			for client := range c.Room.Clients {
				if client.UserID == msg.TargetUserID {
					client.SendChan <- &msg
					break
				}
			}
			c.Room.mu.RUnlock()
		} else {
			// Broadcast other messages to all clients in the room
			c.Room.Broadcast <- &msg
		}
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

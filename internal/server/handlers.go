package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/leetcode"
)

var (
	ErrorSlug           = fmt.Errorf("no slug provided or no data found")
	ErrNotFound         = fmt.Errorf("not found")
	ErrMethodNotAllowed = fmt.Errorf("method not allowed")
	ErrRoomFull         = fmt.Errorf("room is full")
	ErrEmptyRoomId      = fmt.Errorf("please enter a valid room ID")
	ErrRoomNotFound     = fmt.Errorf("room not found. please check the room ID and try again")
	ErrRoomFullMsg      = fmt.Errorf("room is full. please try another room")
	ErrJoinFailed       = fmt.Errorf("failed to join room. please try again")
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Get the tmpl from request context
	tmpl := r.Context().Value("template").(*template.Template)
	if r.URL.Path != "/" {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if r.Method != http.MethodGet {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "Index", nil); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

func JoinCollaborativeSessionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roomID := vars["room_id"]

	if roomID == "" {
		SendErrorResponse(w, http.StatusBadRequest, ErrInvalidRoomId)
		return
	}

	roomManager.mu.RLock()
	room, exists := roomManager.Rooms[roomID]
	roomManager.mu.RUnlock()
	log.Println("room ==== ", room, "roomID=", roomID)
	if !exists {
		SendErrorResponse(w, http.StatusBadRequest, ErrInvalidRoomId)
		return
	}

	SendJSONResponse(w, http.StatusOK, room)
}

// HealthHandler handles the API healthz params.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	SendJSONResponse(w, http.StatusOK, "hurray. api is working fine.")
}

// SearchQuestionHandler
func SearchQuestionHandler(w http.ResponseWriter, r *http.Request) {
	// Get the template from  context
	tmpl := r.Context().Value("template").(*template.Template)

	questionSlug := r.FormValue("questionTitleSlug")

	// When slug are not provided nicely
	if len(questionSlug) == 0 {
		SendErrorResponse(w, http.StatusBadRequest, ErrorSlug)
		return
	}
	graphQLOutput, err := leetcode.FetchQuestionByTitleSlugFromLeetcodeGql(questionSlug)
	if err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}

	data := QuestionData{
		Title:            graphQLOutput.Data.Question.Title,
		Description:      template.HTML(graphQLOutput.Data.Question.Content),
		Difficulty:       graphQLOutput.Data.Question.Difficulty,
		CodeEditorCode:   graphQLOutput.Data.Question.CodeSnippetsMap["python3"].Code,
		Likes:            graphQLOutput.Data.Question.Likes,
		Hints:            graphQLOutput.Data.Question.Hints,
		ProblemLink:      fmt.Sprintf(`https://leetcode.com/problems/%s`, graphQLOutput.Data.Question.TitleSlug),
		AskedInCompanies: []string{"Microsoft", "Intuit", "Amazon"},
	}

	if err := tmpl.ExecuteTemplate(w, "QuestionBlock", data); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

// CreateRoomHandler handles the creation of new rooms
func CreateRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the templ from the context
	tmpl := r.Context().Value("template").(*template.Template)

	// Generate a unique room ID
	roomID := uuid.New().String()

	roomManager.mu.Lock()
	if len(roomManager.Rooms) >= roomManager.maxRooms {
		roomManager.cleanupOldRooms()
	}

	// Create room in memory (it will be fully initialized when WebSocket connects)
	room := CreateRoom(roomID)
	roomManager.Rooms[roomID] = room
	roomManager.mu.Unlock()

	// Setting up the data
	data := CollaborativeRoomPageData{
		Title:                     "Practice Leetcode Multiplayer",
		SupportedProgrammingLangs: []string{"Python", "Java", "Javascript"},
		Message:                   "Hello Sounish, Welcome to the Leetcode Practice Problems",
		Room: RoomResponse{
			RoomID:       roomID,
			Message:      "Room created successfully",
			WebSocketURL: "/ws?room_id=" + roomID,
		},
	}

	if err := tmpl.ExecuteTemplate(w, "HomePage", data); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

// JoinRoomHandler handles requests to join existing rooms
func JoinRoomHandler(w http.ResponseWriter, r *http.Request) {
	// Grab the templ from the context
	tmpl := r.Context().Value("template").(*template.Template)

	roomID := r.FormValue("room_id")
	if roomID == "" {
		SendErrorResponse(w, http.StatusBadRequest, ErrEmptyRoomId)
		return
	}

	roomManager.mu.RLock()
	room, exists := roomManager.Rooms[roomID]
	roomManager.mu.RUnlock()

	if !exists {
		SendErrorResponse(w, http.StatusBadRequest, ErrRoomNotFound)
		return
	}

	// Check if room is full
	room.mu.RLock()
	clientCount := len(room.Clients)
	room.mu.RUnlock()

	if clientCount == 2 {
		SendErrorResponse(w, http.StatusConflict, ErrRoomFullMsg)
		return
	}

	// Setting up the data
	data := CollaborativeRoomPageData{
		Title:                     "Practice Leetcode Multiplayer",
		SupportedProgrammingLangs: []string{"Python", "Java", "Javascript"},
		Message:                   "Hello Sounish, Welcome to the Leetcode Practice Problems",
		Room: RoomResponse{
			RoomID:       roomID,
			Message:      "Room joined successfully",
			WebSocketURL: "/ws?room_id=" + roomID,
		},
	}

	if err := tmpl.ExecuteTemplate(w, "HomePage", data); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, ErrJoinFailed)
		return
	}
}

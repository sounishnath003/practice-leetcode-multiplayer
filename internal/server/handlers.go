package server

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

var (
	ErrorSlug           = fmt.Errorf("no slug provided or no data found")
	ErrNotFound         = fmt.Errorf("not found")
	ErrMethodNotAllowed = fmt.Errorf("method not allowed")
	ErrRoomFull         = fmt.Errorf("room is full")
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

	question_slug := r.FormValue("searchQuestion")

	// When slug are not provided nicely
	if len(question_slug) == 0 {
		SendErrorResponse(w, http.StatusBadRequest, ErrorSlug)
		return
	}
	data := QuestionData{
		Title: "Two Sum",
		Description: template.HTML(`
		<h3> Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.<h3>

		<p>
		You may assume that each input would have exactly one solution, and you may not use the same element twice.
		You can return the answer in any order.
		</p>

<code>
<pre>
Example 1:

Input: nums = [2,7,11,15], target = 9
Output: [0,1]
Explanation: Because nums[0] + nums[1] == 9, we return [0, 1].
Example 2:

Input: nums = [3,2,4], target = 6
Output: [1,2]
Example 3:

Input: nums = [3,3], target = 6
Output: [0,1]
</pre>
</code>
		
		<p>
		Constraints:
		</p>

		<ol>
		<li> 2 <= nums.length <= 104 </li>
		<li> -109 <= nums[i] <= 109 </li>
		<li> -109 <= target <= 109 </li>
		</ol>

		</br>
		
		<p> <b> Follow-up: </b> Can you come up with an algorithm that is less than O(n2) time complexity? </p>
		`),
		Difficulty:       "Easy",
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
		SendErrorResponse(w, http.StatusBadRequest, ErrInvalidRoomId)
		return
	}

	roomManager.mu.RLock()
	room, exists := roomManager.Rooms[roomID]
	roomManager.mu.RUnlock()

	if !exists {
		SendErrorResponse(w, http.StatusBadRequest, ErrInvalidRoomId)
		return
	}

	// Check if room is full
	room.mu.RLock()
	clientCount := len(room.Clients)
	room.mu.RUnlock()

	if clientCount == 2 {
		SendErrorResponse(w, http.StatusConflict, ErrRoomFull)
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
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

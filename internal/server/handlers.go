package server

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/core"
	"github.com/sounishnath003/practice-leetcode-multiplayer/internal/leetcode"
	"google.golang.org/api/idtoken"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
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

func ExecuteCodeHandler(w http.ResponseWriter, r *http.Request) {
	var req ExecuteCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendErrorResponse(w, http.StatusBadRequest, fmt.Errorf("invalid request body"))
		return
	}
	log.Printf("ExecuteCodeHandler: RoomID=%s, UserID=%s, Language=%s", req.RoomID, req.UserID, req.Language)

	// Grab the templ from the context
	co := r.Context().Value("core").(*core.Core)

	// Context timeout of 3 seconds as requested
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Base64 encode the code
	encodedCode := base64.StdEncoding.EncodeToString([]byte(req.Code))

	// Normalize language
	lang := strings.ToLower(req.Language)
	if req.Language == "C++" || lang == "c++" {
		lang = "cpp"
	}

	// Prepare payload for execution engine
	engineReq := map[string]string{
		"language": lang,
		"code":     encodedCode,
		"stdin":    req.Stdin,
	}

	payload, err := json.Marshal(engineReq)
	if err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to marshal request"))
		return
	}

	engineURL := co.CodeRunnerEngine // os.Getenv("CODE_RUNNER_ENGINE_API")
	if engineURL == "" {
		engineURL = "https://code-execution-engine-797087556919.asia-south1.run.app"
	}

	proxyReq, err := http.NewRequestWithContext(ctx, "POST", engineURL, bytes.NewBuffer(payload))
	if err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to create request"))
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")

	// Add Authorization header if not running locally
	if !strings.Contains(engineURL, "localhost") && !strings.Contains(engineURL, "127.0.0.1") {
		// Create an ID token source for the target audience (the engine URL)
		tokenSource, err := idtoken.NewTokenSource(ctx, engineURL)
		if err != nil {
			log.Printf("Failed to create token source: %v", err)
			SendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("authentication configuration error"))
			return
		}

		token, err := tokenSource.Token()
		if err != nil {
			log.Printf("Failed to fetch ID token: %v", err)
			SendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to authenticate with execution engine"))
			return
		}

		proxyReq.Header.Set("Authorization", "Bearer "+token.AccessToken)

	}

	// Set Origin header to satisfy the engine's domain check

	proxyReq.Header.Set("Origin", "https://practice-leetcode-multiplayer-797087556919.asia-south1.run.app")

	client := &http.Client{}

	resp, err := client.Do(proxyReq)

	if err != nil {

		if ctx.Err() == context.DeadlineExceeded {

			SendErrorResponse(w, http.StatusGatewayTimeout, fmt.Errorf("execution timed out"))

		} else {
			SendErrorResponse(w, http.StatusBadGateway, fmt.Errorf("failed to call execution engine: %v", err))
		}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, fmt.Errorf("failed to read response"))
		return
	}

	// Broadcast execution result to room
	if req.RoomID != "" {
		var execResp map[string]interface{}
		if err := json.Unmarshal(body, &execResp); err == nil {
			log.Printf("Broadcasting execution output to room %s", req.RoomID)
			roomManager.mu.RLock()
			room, exists := roomManager.Rooms[req.RoomID]
			roomManager.mu.RUnlock()

			if exists {
				var role string
				// Find role of the user
				room.mu.RLock()
				for client := range room.Clients {
					if client.UserID == req.UserID {
						role = client.Role
						break
					}
				}
				room.mu.RUnlock()

				msg := &WebSocketMessage{
					Type:    TypeExecutionOutput,
					RoomID:  req.RoomID,
					UserID:  req.UserID,
					Role:    role,
					Content: execResp,
				}
				// Use non-blocking send to avoid hanging if channel is full
				select {
				case room.Broadcast <- msg:
				default:
					log.Printf("Warning: room %s broadcast channel full, dropping execution output", req.RoomID)
				}
			}
		} else {
			log.Printf("Failed to unmarshal execution response for broadcast: %v", err)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Get the tmpl from request context
	tmpl := r.Context().Value("template").(*template.Template)

	roomID := r.URL.Query().Get("room_id")
	if roomID != "" {
		// Attempt auto-join
		roomManager.mu.RLock()
		room, exists := roomManager.Rooms[roomID]
		roomManager.mu.RUnlock()

		if exists {
			// Check capacity
			room.mu.RLock()
			count := len(room.Clients)
			room.mu.RUnlock()

			if count < 2 {
				// Prepare data for HomePage
				data := CollaborativeRoomPageData{
					Title:                     "Practice Leetcode Multiplayer",
					SupportedProgrammingLangs: []string{"Python", "Java", "Javascript", "C++"},
					Message:                   "Welcome to the room!",
					Room: RoomResponse{
						RoomID:       roomID,
						Message:      "Joined via link",
						WebSocketURL: "/ws?room_id=" + roomID,
					},
				}
				if err := tmpl.ExecuteTemplate(w, "Index", data); err != nil {
					SendErrorResponse(w, http.StatusInternalServerError, err)
				}
				return
			}
		}
		// If room invalid or full, just fall through to standard index (maybe could show error param?)
	}

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

type SearchSuggestionData struct {
	Suggestions []leetcode.SearchQuestion
}

func SearchSuggestionsHandler(w http.ResponseWriter, r *http.Request) {
	// tmpl := r.Context().Value("template").(*template.Template)
	keyword := r.FormValue("questionTitleSlug")

	if len(keyword) < 2 {
		// Don't show suggestions for very short keywords
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	questions, err := leetcode.SearchQuestionsListFromLeetcode(ctx, keyword)
	if err != nil {
		log.Printf("Error fetching suggestions: %v", err)
		return
	}

	data := SearchSuggestionData{
		Suggestions: questions,
	}

	// We'll create a small template for suggestions
	const suggestionsTmpl = `
	<div id="searchSuggestions" class="absolute z-50 w-full mt-1 bg-white dark:bg-gray-800 border dark:border-gray-700 rounded-lg shadow-xl overflow-hidden">
		{{ range .Suggestions }}
		<div class="px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 cursor-pointer border-b last:border-b-0 dark:border-gray-700"
			 hx-post="/api/search?questionTitleSlug={{ .TitleSlug }}"
			 hx-target="#questionBlock"
			 hx-swap="outerHTML"
			 hx-on::after-request="document.getElementById('searchSuggestions').remove(); document.getElementById('questionTitleSlug').value = '{{ .TitleSlug }}'">
			<div class="flex justify-between items-center">
				<span class="text-sm font-medium dark:text-white">{{ .Title }}</span>
				<span class="text-xs px-2 py-0.5 rounded {{ if eq .Difficulty "Easy" }}bg-green-100 text-green-700{{ else if eq .Difficulty "Medium" }}bg-amber-100 text-amber-700{{ else }}bg-red-100 text-red-700{{ end }}">
					{{ .Difficulty }}
				</span>
			</div>
		</div>
		{{ end }}
	</div>
	`

	t, err := template.New("suggestions").Parse(suggestionsTmpl)
	if err != nil {
		log.Printf("Error parsing suggestions template: %v", err)
		return
	}

	if err := t.Execute(w, data); err != nil {
		log.Printf("Error executing suggestions template: %v", err)
	}
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	graphQLOutput, err := leetcode.FetchQuestionByTitleSlugFromLeetcodeGql(ctx, questionSlug)
	if err != nil {
		data := QuestionData{Error: "Failed to fetch question: " + err.Error()}
		if err := tmpl.ExecuteTemplate(w, "QuestionBlock", data); err != nil {
			SendErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	if graphQLOutput.Data.Question.Title == "" {
		data := QuestionData{Error: "Question not found for slug: " + questionSlug}
		if err := tmpl.ExecuteTemplate(w, "QuestionBlock", data); err != nil {
			SendErrorResponse(w, http.StatusInternalServerError, err)
		}
		return
	}

	data := QuestionData{
		Title:                 graphQLOutput.Data.Question.Title,
		Description:           template.HTML(graphQLOutput.Data.Question.Content),
		Difficulty:            graphQLOutput.Data.Question.Difficulty,
		PythonCodeSnippet:     graphQLOutput.Data.Question.CodeSnippetsMap["python3"].Code,
		JavaCodeSnippet:       graphQLOutput.Data.Question.CodeSnippetsMap["java"].Code,
		JavascriptCodeSnippet: graphQLOutput.Data.Question.CodeSnippetsMap["javascript"].Code,
		CppCodeSnippet:        graphQLOutput.Data.Question.CodeSnippetsMap["cpp"].Code,
		Likes:                 graphQLOutput.Data.Question.Likes,
		Hints:                 graphQLOutput.Data.Question.Hints,
		ProblemLink:           fmt.Sprintf(`https://leetcode.com/problems/%s`, graphQLOutput.Data.Question.TitleSlug),
		AskedInCompanies:      []string{"Microsoft", "Intuit", "Amazon"},
	}

	if err != nil {
		data.Error = err.Error()
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
		SupportedProgrammingLangs: []string{"Python", "Java", "Javascript", "C++"},
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

	if clientCount >= 2 {
		SendErrorResponse(w, http.StatusConflict, ErrRoomFullMsg)
		return
	}

	// Setting up the data
	data := CollaborativeRoomPageData{
		Title:                     "Practice Leetcode Multiplayer",
		SupportedProgrammingLangs: []string{"Python", "Java", "Javascript", "C++"},
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

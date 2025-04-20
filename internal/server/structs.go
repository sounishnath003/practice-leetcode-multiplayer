package server

import (
	"encoding/json"
	"html/template"
)

type RoomResponse struct {
	RoomID       string `json:"room_id"`
	Message      string `json:"message"`
	WebSocketURL string `json:"ws_url"`
}

type CollaborativeRoomPageData struct {
	Title                     string
	SupportedProgrammingLangs []string
	Message                   string
	Room                      RoomResponse
}

type QuestionData struct {
	Title            string
	Description      template.HTML
	Difficulty       string
	CodeEditorCode   string
	ProblemLink      string
	Hints            []string
	Likes            int64
	AskedInCompanies []string
}

// GraphQLRequest represents the structure of a GraphQL request
type GraphQLRequest struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables"`
}

// GraphQLResponse represents the structure of a GraphQL response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []interface{}   `json:"errors,omitempty"`
}

package server

import "html/template"

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
	AskedInCompanies []string
}

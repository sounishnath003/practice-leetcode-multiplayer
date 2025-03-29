package server

import "html/template"

type IndexPageData struct {
	Title                     string
	SupportedProgrammingLangs []string
	Message                   string
}

type QuestionData struct {
	Title            string
	Description      template.HTML
	Difficulty       string
	AskedInCompanies []string
}

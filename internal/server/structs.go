package server

type IndexPageData struct {
	Title                     string
	SupportedProgrammingLangs []string
	Message                   string
}

type QuestionData struct {
	Title            string
	Description      string
	Difficulty       string
	AskedInCompanies []string
}

package leetcode

// GraphQLRequest represents the structure of a GraphQL request
type GraphQLRequest struct {
	Query     string            `json:"query"`
	Variables map[string]string `json:"variables"`
}

// GraphQLResponse represents the structure of a GraphQL response
type GraphQLResponse struct {
	Data   QuestionNode  `json:"data"`
	Errors []interface{} `json:"errors,omitempty"`
}

type QuestionNode struct {
	Question Question `json:"question"`
}

type Question struct {
	QuestionID         string                 `json:"questionId"`
	QuestionFrontendID string                 `json:"questionFrontendId"`
	Title              string                 `json:"title"`
	TitleSlug          string                 `json:"titleSlug"`
	Content            string                 `json:"content"`
	CodeSnippets       []CodeSnippet          `json:"codeSnippets"`
	CodeSnippetsMap     map[string]CodeSnippet `json:"codeSnippetsMap"`
	Difficulty         string                 `json:"difficulty"`
	Likes              int64                  `json:"likes"`
	Hints              []string               `json:"hints"`
}

type CodeSnippet struct {
	Lang     string `json:"lang"`
	LangSlug string `json:"langSlug"`
	Code     string `json:"code"`
}

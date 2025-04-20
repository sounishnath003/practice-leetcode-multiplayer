package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Github References: https://github.com/akarsh1995/leetcode-graphql-queries/blob/main/problemset_page/problemset_page.graphql
// Kudos to @Author: https://github.com/akarsh1995/
// Who has wrote the GraphQL queries from leetcode
// FetchQuestionByTitleSlugFromLeetcodeGql fetches the question details from LeetCode using the titleSlug
func FetchQuestionByTitleSlugFromLeetcodeGql(w http.ResponseWriter, r *http.Request) {
	// Get the titleSlug from the query parameters
	titleSlug := r.URL.Query().Get("titleSlug")
	if titleSlug == "" {
		http.Error(w, "titleSlug is required", http.StatusBadRequest)
		return
	}

	// Define the GraphQL query
	query := `
        query questionTitle($titleSlug: String!) {
            question(titleSlug: $titleSlug) {
                questionId
                questionFrontendId
                title
                titleSlug
                content
                codeSnippets {
                    lang
                    langSlug
                    code
                }
                isPaidOnly
                difficulty
                likes
                dislikes
                hints
            }
        }
    `

	// Prepare the GraphQL request payload
	requestPayload := GraphQLRequest{
		Query: query,
		Variables: map[string]string{
			"titleSlug": titleSlug,
		},
	}

	// Marshal the request payload to JSON
	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		http.Error(w, "Failed to create request payload", http.StatusInternalServerError)
		return
	}

	// Make the HTTP POST request to the LeetCode GraphQL API
	resp, err := http.Post("https://leetcode.com/graphql", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		http.Error(w, "Failed to fetch question from LeetCode", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response from LeetCode", http.StatusInternalServerError)
		return
	}

	// Parse the GraphQL response
	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		http.Error(w, "Failed to parse response from LeetCode", http.StatusInternalServerError)
		return
	}

	// Check for errors in the GraphQL response
	if len(graphqlResponse.Errors) > 0 {
		http.Error(w, "LeetCode API returned errors", http.StatusInternalServerError)
		return
	}

	// Return the JSON response to the frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(graphqlResponse.Data)
}

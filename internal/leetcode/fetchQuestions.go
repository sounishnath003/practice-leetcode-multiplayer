package leetcode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Github References: https://github.com/akarsh1995/leetcode-graphql-queries/blob/main/problemset_page/problemset_page.graphql
//
// Kudos to @Author https://github.com/akarsh1995/
// Who has wrote the GraphQL queries from leetcode
// FetchQuestionByTitleSlugFromLeetcodeGql fetches the question details from LeetCode using the titleSlug
func FetchQuestionByTitleSlugFromLeetcodeGql(titleSlug string) (GraphQLResponse, error) {
	// Get the titleSlug from the query parameters
	if titleSlug == "" {
		return GraphQLResponse{}, fmt.Errorf("titleslug is required")
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
				difficulty
				likes
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
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}

	// Make the HTTP POST request to the LeetCode GraphQL API
	resp, err := http.Post("https://leetcode.com/graphql", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GraphQLResponse{}, fmt.Errorf("failed to read response from leetcode")
	}

	// Parse the GraphQL response
	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		return GraphQLResponse{}, fmt.Errorf("failed to parse response from leetcode")
	}

	// Check for errors in the GraphQL response
	if len(graphqlResponse.Errors) > 0 {
		return GraphQLResponse{}, fmt.Errorf("leetcode api returned errors")
	}
	// Create a map to store the filtered CodeSnippets by language slug
	filteredSnippetsMap := make(map[string]CodeSnippet)
	for _, snippet := range graphqlResponse.Data.Question.CodeSnippets {
		if snippet.LangSlug == "java" || snippet.LangSlug == "python3" || snippet.LangSlug == "javascript" {
			filteredSnippetsMap[snippet.LangSlug] = snippet
		}
	}
	graphqlResponse.Data.Question.CodeSnippets = nil                    // Clear the original slice
	graphqlResponse.Data.Question.CodeSnippetsMap = filteredSnippetsMap // Keep only 3 languages supports

	return graphqlResponse, nil
}

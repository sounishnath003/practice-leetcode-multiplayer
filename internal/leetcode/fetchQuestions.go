package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Github References: https://github.com/akarsh1995/leetcode-graphql-queries/blob/main/problemset_page/problemset_page.graphql
//
// Kudos to @Author https://github.com/akarsh1995/
// Who has wrote the GraphQL queries from leetcode
// FetchQuestionByTitleSlugFromLeetcodeGql fetches the question details from LeetCode using the titleSlug
func FetchQuestionByTitleSlugFromLeetcodeGql(ctx context.Context, titleSlug string) (GraphQLResponse, error) {
	// Get the titleSlug from the query parameters
	if titleSlug == "" {
		fmt.Println("[LeetcodeGQL] Error: titleslug is empty")
		return GraphQLResponse{}, fmt.Errorf("titleslug is required")
	}
	fmt.Printf("[LeetcodeGQL] Starting fetch for titleslug: %s\n", titleSlug)

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
		fmt.Printf("[LeetcodeGQL] Error marshaling request payload: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}
	fmt.Println("[LeetcodeGQL] JSON marshaled successfully. Sending POST request to LeetCode GraphQL endpoint.")

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", "https://leetcode.com/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		return GraphQLResponse{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a client with a timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[LeetcodeGQL] Error in request: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}
	defer resp.Body.Close()

	fmt.Printf("[LeetcodeGQL] Received response: HTTP %d\n", resp.StatusCode)

	// Log HTTP error if status is not 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyErrMsg, _ := io.ReadAll(resp.Body)
		fmt.Printf("[LeetcodeGQL] HTTP ERROR: status=%d message=%s\n", resp.StatusCode, string(bodyErrMsg))
		return GraphQLResponse{}, fmt.Errorf("leetcode api http error: status=%d msg=%s", resp.StatusCode, string(bodyErrMsg))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("[LeetcodeGQL] Error reading response body: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to read response from leetcode")
	}

	fmt.Printf("[LeetcodeGQL] Response body read (%d bytes). Attempting to parse JSON...\n", len(body))

	// Parse the GraphQL response
	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		fmt.Printf("[LeetcodeGQL] Error unmarshaling GraphQL response JSON: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to parse response from leetcode")
	}

	// Check if the response contains valid data
	if graphqlResponse.Data.Question.QuestionID == "" {
		fmt.Printf("[LeetcodeGQL] No question found for the slug: '%s' (empty QuestionID in response)\n", titleSlug)
		return GraphQLResponse{}, fmt.Errorf(
			"slug not found: `%s`; empty response from leetcode API", titleSlug,
		)
	}

	// Check for errors in the GraphQL response
	if len(graphqlResponse.Errors) > 0 {
		fmt.Printf("[LeetcodeGQL] GraphQL errors received: %v\n", graphqlResponse.Errors)
		return GraphQLResponse{}, fmt.Errorf("leetcode api returned errors")
	}

	// Create a map to store the filtered CodeSnippets by language slug
	filteredSnippetsMap := make(map[string]CodeSnippet)
	for _, snippet := range graphqlResponse.Data.Question.CodeSnippets {
		if snippet.LangSlug == "java" || snippet.LangSlug == "python3" || snippet.LangSlug == "javascript" {
			filteredSnippetsMap[snippet.LangSlug] = snippet
		}
	}
	fmt.Printf("[LeetcodeGQL] Filtered code snippets for java, python3, javascript: %+v\n", filteredSnippetsMap)
	graphqlResponse.Data.Question.CodeSnippets = nil                    // Clear the original slice
	graphqlResponse.Data.Question.CodeSnippetsMap = filteredSnippetsMap // Keep only 3 languages supports

	fmt.Println("[LeetcodeGQL] Successfully fetched and processed LeetCode question.")
	return graphqlResponse, nil
}

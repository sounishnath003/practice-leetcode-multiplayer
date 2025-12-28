package leetcode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Github References: https://github.com/akarsh1995/leetcode-graphql-queries/blob/main/problemset_page/problemset_page.graphql
//
// Kudos to @Author https://github.com/akarsh1995/
// Who has wrote the GraphQL queries from leetcode
// FetchQuestionByTitleSlugFromLeetcodeGql fetches the question details from LeetCode using the titleSlug
func FetchQuestionByTitleSlugFromLeetcodeGql(ctx context.Context, keyword string) (GraphQLResponse, error) {
	if keyword == "" {
		log.Println("[LeetcodeGQL] Error: keyword is empty")
		return GraphQLResponse{}, fmt.Errorf("keyword is required")
	}

	log.Printf("[LeetcodeGQL] Starting smart fetch for keyword: %s\n", keyword)

	// 1. Search for the question using the keyword
	log.Printf("[LeetcodeGQL] Attempting to search for keyword: %s\n", keyword)
	searchResp, err := searchQuestionsFromLeetcode(ctx, keyword)
	if err != nil {
		log.Printf("[LeetcodeGQL] Search failed for '%s': %v. Falling back to direct fetch assume keyword is a slug.\n", keyword, err)
		// Fallback: Try fetching directly assuming keyword is a valid slug
		return fetchQuestionDetailsBySlug(ctx, keyword)
	}

	if searchResp.Data.ProblemsetQuestionList == nil || len(searchResp.Data.ProblemsetQuestionList.Questions) == 0 {
		log.Printf("[LeetcodeGQL] Search returned no results for keyword: %s. Falling back to direct fetch.\n", keyword)
		// Fallback or Error? Let's try direct fetch just in case it's a specific slug that search missed
		return fetchQuestionDetailsBySlug(ctx, keyword)
	}

	// 2. Pick the first result
	bestMatch := searchResp.Data.ProblemsetQuestionList.Questions[0]
	log.Printf("[LeetcodeGQL] Search matched: '%s' (resolved slug: %s). Proceeding to fetch full details.\n", bestMatch.Title, bestMatch.TitleSlug)

	// 3. Fetch details for the found slug
	return fetchQuestionDetailsBySlug(ctx, bestMatch.TitleSlug)
}

func searchQuestionsFromLeetcode(ctx context.Context, keyword string) (GraphQLResponse, error) {
	log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Searching for: %s\n", keyword)
	query := `query problemsetQuestionList($filters: QuestionListFilterInput) {
        problemsetQuestionList: questionList(
                categorySlug: ""
                limit: 1
                skip: 0
                filters: $filters
        ) {
                questions: data {
                        title
                        titleSlug
                        difficulty
                }
        }
}`

	type SearchVariables struct {
		Filters map[string]string `json:"filters"`
	}
	type SearchRequest struct {
		Query     string          `json:"query"`
		Variables SearchVariables `json:"variables"`
	}

	reqBodyStruct := SearchRequest{
		Query: query,
		Variables: SearchVariables{
			Filters: map[string]string{
				"searchKeywords": keyword,
			},
		},
	}

	requestBody, err := json.Marshal(reqBodyStruct)
	if err != nil {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Error marshaling search request: %v\n", err)
		return GraphQLResponse{}, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://leetcode.com/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Error creating search request object: %v\n", err)
		return GraphQLResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Sending search request to LeetCode...\n")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] HTTP error during search request: %v\n", err)
		return GraphQLResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Received non-2xx status code: %d\n", resp.StatusCode)
		return GraphQLResponse{}, fmt.Errorf("search http error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Error reading search response body: %v\n", err)
		return GraphQLResponse{}, err
	}

	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Error unmarshaling search response JSON: %v\n", err)
		return GraphQLResponse{}, err
	}

	log.Printf("[LeetcodeGQL] [searchQuestionsFromLeetcode] Successfully received search results.\n")
	return graphqlResponse, nil
}

// SearchQuestionsListFromLeetcode fetches top 5 suggestions for a keyword
func SearchQuestionsListFromLeetcode(ctx context.Context, keyword string) ([]SearchQuestion, error) {
	if keyword == "" {
		return nil, nil
	}

	log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Fetching suggestions for: %s\n", keyword)

	query := "\n\t\tquery problemsetQuestionList($filters: QuestionListFilterInput) {\n\t\t\tproblemsetQuestionList: questionList(\n\t\t\t\tcategorySlug: \"\"\n\t\t\t\tlimit: 5\n\t\t\t\tskip: 0\n\t\t\t\tfilters: $filters\n\t\t\t) {\n\t\t\t\tquestions: data {\n\t\t\t\t\ttitle\n\t\t\t\t\ttitleSlug\n\t\t\t\t\tdifficulty\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t"

	type SearchVariables struct {
		Filters map[string]string `json:"filters"`
	}
	type SearchRequest struct {
		Query     string          `json:"query"`
		Variables SearchVariables `json:"variables"`
	}

	reqBodyStruct := SearchRequest{
		Query: query,
		Variables: SearchVariables{
			Filters: map[string]string{
				"searchKeywords": keyword,
			},
		},
	}

	requestBody, err := json.Marshal(reqBodyStruct)
	if err != nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Error marshaling suggestions request: %v\n", err)
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://leetcode.com/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Error creating request object: %v\n", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] HTTP error: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Error reading body: %v\n", err)
		return nil, err
	}

	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Error parsing JSON: %v\n", err)
		return nil, err
	}

	if graphqlResponse.Data.ProblemsetQuestionList == nil {
		log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] No suggestions found.\n")
		return nil, nil
	}

	log.Printf("[LeetcodeGQL] [SearchQuestionsListFromLeetcode] Found %d suggestions.\n", len(graphqlResponse.Data.ProblemsetQuestionList.Questions))
	return graphqlResponse.Data.ProblemsetQuestionList.Questions, nil
}

// fetchQuestionDetailsBySlug is the original FetchQuestionByTitleSlugFromLeetcodeGql logic
func fetchQuestionDetailsBySlug(ctx context.Context, titleSlug string) (GraphQLResponse, error) {
	if titleSlug == "" {
		log.Println("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Error: titleslug is empty")
		return GraphQLResponse{}, fmt.Errorf("titleslug is required")
	}
	log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Fetching full details for slug: %s\n", titleSlug)

	// Define the GraphQL query
	query := "query questionTitle($titleSlug: String!) {\n        question(titleSlug: $titleSlug) {\n                questionId\n                questionFrontendId\n                title\n                titleSlug\n                content\n                codeSnippets {\n                        lang\n                        langSlug\n                        code\n                }\n                difficulty\n                likes\n                hints\n        }\n}"

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
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Error marshaling request payload: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "POST", "https://leetcode.com/graphql", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Error creating request object: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a client with a timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Sending details request to LeetCode...\n")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] HTTP error during details request: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to fetch question from leetcode")
	}
	defer resp.Body.Close()

	log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Received response: HTTP %d\n", resp.StatusCode)

	// Log HTTP error if status is not 2xx
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyErrMsg, _ := io.ReadAll(resp.Body)
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] HTTP ERROR: status=%d message=%s\n", resp.StatusCode, string(bodyErrMsg))
		return GraphQLResponse{}, fmt.Errorf("leetcode api http error: status=%d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Error reading response body: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to read response from leetcode")
	}

	// Parse the GraphQL response
	var graphqlResponse GraphQLResponse
	if err := json.Unmarshal(body, &graphqlResponse); err != nil {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Error unmarshaling GraphQL response JSON: %v\n", err)
		return GraphQLResponse{}, fmt.Errorf("failed to parse response from leetcode")
	}

	// Check if the response contains valid data
	if graphqlResponse.Data.Question.QuestionID == "" {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] No question found for the slug: '%s' (empty QuestionID in response)\n", titleSlug)
		return GraphQLResponse{}, fmt.Errorf(
			"slug not found: `%s`; empty response from leetcode API", titleSlug,
		)
	}

	// Check for errors in the GraphQL response
	if len(graphqlResponse.Errors) > 0 {
		log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] GraphQL errors received: %v\n", graphqlResponse.Errors)
		return GraphQLResponse{}, fmt.Errorf("leetcode api returned errors")
	}

	// Create a map to store the filtered CodeSnippets by language slug
	filteredSnippetsMap := make(map[string]CodeSnippet)
	for _, snippet := range graphqlResponse.Data.Question.CodeSnippets {
		if snippet.LangSlug == "java" || snippet.LangSlug == "python3" || snippet.LangSlug == "javascript" || snippet.LangSlug == "cpp" {
			filteredSnippetsMap[snippet.LangSlug] = snippet
		}
	}
	log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Filtered code snippets for: %v\n", []string{"java", "python3", "javascript", "cpp"})
	graphqlResponse.Data.Question.CodeSnippets = nil                    // Clear the original slice
	graphqlResponse.Data.Question.CodeSnippetsMap = filteredSnippetsMap // Keep only 4 languages supports

	log.Printf("[LeetcodeGQL] [fetchQuestionDetailsBySlug] Successfully processed question: %s\n", graphqlResponse.Data.Question.Title)
	return graphqlResponse, nil
}

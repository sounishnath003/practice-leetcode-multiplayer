package server

import (
	"fmt"
	"html/template"
	"net/http"
)

var (
	ErrorSlug   = fmt.Errorf("no slug provided or no data found")
	ErrNotFound = fmt.Errorf("not found")
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	// Get the tmpl from request context
	tmpl := r.Context().Value("template").(*template.Template)
	if r.URL.Path != "/" {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}
	if r.Method != http.MethodGet {
		SendErrorResponse(w, http.StatusNotFound, ErrNotFound)
		return
	}

	// Setting up the data
	data := IndexPageData{
		Title:                     "Practice Leetcode Multiplayer",
		SupportedProgrammingLangs: []string{"Python", "Java", "Javascript"},
		Message:                   "Hello Sounish, Welcome to the Leetcode Practice Problems",
	}

	if err := tmpl.ExecuteTemplate(w, "Index", data); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

// HealthHandler handles the API healthz params.
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	SendJSONResponse(w, http.StatusOK, "hurray. api is working fine.")
}

// SearchQuestionHandler
func SearchQuestionHandler(w http.ResponseWriter, r *http.Request) {
	// Get the template from  context
	tmpl := r.Context().Value("template").(*template.Template)

	question_slug := r.FormValue("searchQuestion")

	// When slug are not provided nicely
	if len(question_slug) == 0 {
		SendErrorResponse(w, http.StatusBadRequest, ErrorSlug)
		return
	}
	data := QuestionData{
		Title: "Two Sum",
		Description: template.HTML(`
		<h3> Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.<h3>

		<p>
		You may assume that each input would have exactly one solution, and you may not use the same element twice.
		You can return the answer in any order.
		</p>

<code>
<pre>
Example 1:

Input: nums = [2,7,11,15], target = 9
Output: [0,1]
Explanation: Because nums[0] + nums[1] == 9, we return [0, 1].
Example 2:

Input: nums = [3,2,4], target = 6
Output: [1,2]
Example 3:

Input: nums = [3,3], target = 6
Output: [0,1]
</pre>
</code>
		
		<p>
		Constraints:
		</p>

		<ol>
		<li> 2 <= nums.length <= 104 </li>
		<li> -109 <= nums[i] <= 109 </li>
		<li> -109 <= target <= 109 </li>
		</ol>

		</br>
		
		<p> <b> Follow-up: </b> Can you come up with an algorithm that is less than O(n2) time complexity? </p>
		`),
		Difficulty:       "Hard",
		AskedInCompanies: []string{"Microsoft", "Intuit", "Amazon"},
	}

	if err := tmpl.ExecuteTemplate(w, "QuestionBlock", data); err != nil {
		SendErrorResponse(w, http.StatusInternalServerError, err)
		return
	}
}

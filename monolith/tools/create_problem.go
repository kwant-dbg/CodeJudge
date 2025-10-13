package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Define structs to match the JSON structure
type TestCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type Problem struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Difficulty  string     `json:"difficulty"`
	TestCases   []TestCase `json:"test_cases"`
}

func main() {
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoyLCJ1c2VybmFtZSI6InRlc3RhZG1pbiIsInJvbGUiOiJ1c2VyIiwic3ViIjoiMiIsImV4cCI6MTc2MDQ3MzAwMywiaWF0IjoxNzYwMzg2NjAzfQ.YPhk8TPUv19Kw5BjdIXbBDWAHpQORIMdm91rG7jnUeI"
	url := "http://localhost:8080/api/problems"

	// Create the problem data using Go structs
	problem := Problem{
		Title: "Valid Parentheses",
		Description: "Given a string s containing just the characters '(', ')', '{', '}', '[' and ']', determine if the input string is valid.\n\nAn input string is valid if:\n1. Open brackets must be closed by the same type of brackets.\n2. Open brackets must be closed in the correct order.\n3. Every close bracket has a corresponding open bracket of the same type.\n\n### Input\nThe first and only line of input contains a single string s.\n\n### Output\nPrint `true` if the string is valid, and `false` otherwise.\n\n### Examples\n**Example 1:**\nInput: `s = \"()\"`\nOutput: `true`\n\n**Example 2:**\nInput: `s = \"()[]{}\" `\nOutput: `true`\n\n**Example 3:**\nInput: `s = \"(]\"`\nOutput: `false`\n\n### Constraints\n- $1 \\le s.length \\le 10^4$
- `s` consists of parentheses only '()[]{}'.",
		Difficulty: "easy",
		TestCases: []TestCase{
			{Input: "()", Output: "true"},
			{Input: "()[]{}", Output: "true"},
			{Input: "(]", Output: "false"},
			{Input: "{[()]}", Output: "true"},
			{Input: "([)]", Output: "false"},
			{Input: "((( ", Output: "false"},
		},
	}

	// Marshal the Go struct to a JSON byte array
	jsonBody, err := json.Marshal(problem)
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Read and print the response
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	fmt.Printf("Response Body: %s\n", string(body))
}
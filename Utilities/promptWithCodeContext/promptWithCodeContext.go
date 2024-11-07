package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Constants for directories and files
const (
	CloneDir           = "./cloned_repos"       // Directory to clone repositories
	SessionContextFile = "session_context.json" // File to store session context
	OpenAIAPIURL       = "https://api.openai.com/v1/chat/completions"
	Model              = "gpt-4"                // OpenAI model to use
	MaxTokens          = 2048                   // Maximum tokens for OpenAI response
)

// Structures for session context and OpenAI API interactions
type SessionContext struct {
	Repositories map[string]string `json:"repositories"` // repo_name: aggregated_code
}

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int            `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

func main() {
	// Define command-line flags
	prompt := flag.String("prompt", "", "The prompt to send to OpenAI")
	repos := flag.String("repos", "", "Comma-separated list of GitHub repository URLs")
	flag.Parse()

	// Validate required flags
	if *prompt == "" || *repos == "" {
		flag.Usage()
		log.Fatal("Both -prompt and -repos arguments are required.")
	}

	// Split repository URLs
	repoURLs := strings.Split(*repos, ",")

	// Ensure clone directory exists
	if err := os.MkdirAll(CloneDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create clone directory: %v", err)
	}

	// Load existing session context or create a new one
	sessionContext := loadSessionContext()

	// Process each repository
	for _, url := range repoURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		repoName, err := extractRepoName(url)
		if err != nil {
			log.Printf("Skipping URL '%s': %v", url, err)
			continue
		}

		// Clone the repository if not already cloned
		repoPath := filepath.Join(CloneDir, repoName)
		if _, err := os.Stat(repoPath); os.IsNotExist(err) {
			log.Printf("Cloning repository: %s", url)
			if err := gitClone(url, repoPath); err != nil {
				log.Printf("Failed to clone '%s': %v", url, err)
				continue
			}
		} else {
			log.Printf("Repository already cloned: %s", repoName)
			// Pull the latest changes
			if err := gitPull(repoPath); err != nil {
				log.Printf("Failed to pull latest changes for '%s': %v", repoName, err)
				continue
			}
		}

		// Aggregate code from the repository
		aggregatedCode, err := aggregateCode(repoPath)
		if err != nil {
			log.Printf("Failed to aggregate code for '%s': %v", repoName, err)
			continue
		}

		// Update session context
		sessionContext.Repositories[repoName] = aggregatedCode
	}

	// Save updated session context
	saveSessionContext(sessionContext)

	// Prepare the combined context
	combinedContext := prepareCombinedContext(sessionContext)

	// Create OpenAI prompt
	openaiPrompt := fmt.Sprintf("%s\n\n%s", combinedContext, *prompt)

	// Send request to OpenAI
	response, err := sendOpenAIRequest(openaiPrompt)
	if err != nil {
		log.Fatalf("OpenAI request failed: %v", err)
	}

	// Display the response
	for _, choice := range response.Choices {
		fmt.Printf("ChatGPT: %s\n", choice.Message.Content)
	}
}

// extractRepoName extracts the repository name from the GitHub URL
func extractRepoName(url string) (string, error) {
	// Example URL formats:
	// https://github.com/user/repo.git
	// https://github.com/user/repo
	parts := strings.Split(url, "/")
	if len(parts) < 5 {
		return "", fmt.Errorf("invalid GitHub URL format")
	}
	repoPart := parts[len(parts)-1]
	repoName := strings.TrimSuffix(repoPart, ".git")
	return repoName, nil
}

// gitClone clones a GitHub repository to the specified path
func gitClone(url, path string) error {
	cmd := exec.Command("git", "clone", url, path)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone error: %v, %s", err, stderr.String())
	}
	return nil
}

// gitPull pulls the latest changes in the specified repository path
func gitPull(path string) error {
	cmd := exec.Command("git", "-C", path, "pull")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git pull error: %v, %s", err, stderr.String())
	}
	return nil
}

// aggregateCode traverses the repository directory and aggregates code from relevant files
func aggregateCode(repoPath string) (string, error) {
	var aggregatedCode strings.Builder

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and hidden files
		if info.IsDir() || strings.HasPrefix(info.Name(), ".") {
			return nil
		}

		// Consider code files based on extension
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !isCodeFile(ext) {
			return nil
		}

		// Read the file content
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Printf("Failed to read file '%s': %v", path, err)
			return nil // Continue with other files
		}

		// Append to aggregated code with file path as a header
		relativePath, _ := filepath.Rel(repoPath, path)
		aggregatedCode.WriteString(fmt.Sprintf("\n\n// File: %s\n", relativePath))
		aggregatedCode.Write(content)
		aggregatedCode.WriteString("\n")
		return nil
	})

	if err != nil {
		return "", err
	}

	return aggregatedCode.String(), nil
}

// isCodeFile checks if a file extension corresponds to a code file
func isCodeFile(ext string) bool {
	codeExtensions := []string{
		".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".cs",
		".rb", ".php", ".swift", ".kt", ".rs", ".scala", ".sh",
		".html", ".css", ".json", ".yaml", ".yml", ".md",
	}
	for _, ce := range codeExtensions {
		if ext == ce {
			return true
		}
	}
	return false
}

// loadSessionContext loads the session context from the JSON file or initializes a new one
func loadSessionContext() SessionContext {
	var context SessionContext

	file, err := os.Open(SessionContextFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Initialize a new session context
			context = SessionContext{
				Repositories: make(map[string]string),
			}
			return context
		}
		log.Fatalf("Failed to open session context file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&context); err != nil {
		log.Fatalf("Failed to decode session context: %v", err)
	}

	// Ensure the map is initialized
	if context.Repositories == nil {
		context.Repositories = make(map[string]string)
	}

	return context
}

// saveSessionContext saves the session context to the JSON file
func saveSessionContext(context SessionContext) {
	file, err := os.Create(SessionContextFile)
	if err != nil {
		log.Fatalf("Failed to create session context file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(context); err != nil {
		log.Fatalf("Failed to encode session context: %v", err)
	}
}

// prepareCombinedContext combines all repository codes into a single string
func prepareCombinedContext(context SessionContext) string {
	var combined strings.Builder

	combined.WriteString("### Repository Code Context\n\n")
	for repo, code := range context.Repositories {
		combined.WriteString(fmt.Sprintf("#### %s\n", repo))
		combined.WriteString("```go\n") // Adjust language as needed or omit
		combined.WriteString(code)
		combined.WriteString("\n```\n\n")
	}

	return combined.String()
}

// sendOpenAIRequest sends the prompt along with context to OpenAI API and returns the response
func sendOpenAIRequest(prompt string) (*OpenAIResponse, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	requestBody := OpenAIRequest{
		Model: Model,
		Messages: []OpenAIMessage{
			{
				Role:    "system",
				Content: "You are a helpful assistant with access to the following repository code context.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   MaxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal OpenAI request: %v", err)
	}

	req, err := http.NewRequest("POST", OpenAIAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request error: %v", err)
	}
	defer resp.Body.Close()

	// Read and parse the response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenAI response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var openAIResp OpenAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal OpenAI response: %v", err)
	}

	return &openAIResp, nil
}
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

var client *openai.Client
var tracer = otel.Tracer("llm-chat-service")

func init() {
	// Try loading from parent directory first
	err := godotenv.Load("../.env")
	if err != nil {
		// If that fails, try loading from current directory
		err = godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	// Debug: Print loaded env vars
	fmt.Println("Loaded Environment Variables:")
	fmt.Printf("OPENAI_API_KEY: %s\n", maskSecret(os.Getenv("OPENAI_API_KEY")))
	fmt.Printf("PHOENIX_COLLECTOR_ENDPOINT: %s\n", os.Getenv("PHOENIX_COLLECTOR_ENDPOINT"))
	fmt.Printf("PHOENIX_API_KEY: %s\n", maskSecret(os.Getenv("PHOENIX_API_KEY")))

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set in .env file")
	}

	// Debug: Print first 4 and last 4 characters of the key and validate length
	keyLength := len(apiKey)
	if keyLength != 51 {
		log.Printf("Warning: API key length is %d, expected 51 characters", keyLength)
	}
	if keyLength > 8 {
		log.Printf("API Key starts with: %s... ends with: ...%s (length: %d)\n",
			apiKey[:4], apiKey[keyLength-4:], keyLength)
	}

	client = openai.NewClient(apiKey)
}

// Helper function to mask secrets
func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "chat_completion")
	defer span.End()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add input text to span
	span.SetAttributes(attribute.String("input.text", req.Message))

	// Create ChatGPT request
	resp, err := client.CreateChatCompletion(
		ctx, // Pass the context with trace
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: req.Message,
				},
			},
		},
	)

	if err != nil {
		log.Printf("ChatCompletion error: %v\n", err)
		span.RecordError(err)
		http.Error(w, "Error getting response from OpenAI", http.StatusInternalServerError)
		return
	}

	// Add output text to span
	span.SetAttributes(attribute.String("output.text", resp.Choices[0].Message.Content))

	// Send response back to client
	response := ChatResponse{
		Response: resp.Choices[0].Message.Content,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Initialize OpenTelemetry
	shutdown, err := initTracer()
	if err != nil {
		log.Printf("Warning: Failed to initialize tracer: %v", err)
		// Continue without tracing
	} else {
		defer shutdown(context.Background())
	}

	// Serve static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Route handlers
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/chat", handleChat)

	fmt.Println("\n===========================================")
	fmt.Println("🚀 Server starting at http://localhost:8080")
	fmt.Println("===========================================")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

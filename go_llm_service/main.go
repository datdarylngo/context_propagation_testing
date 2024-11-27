package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	pb "go_llm_service/proto"

	"github.com/joho/godotenv"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type ChatResponse struct {
	Response string `json:"response"`
}

var (
	client     *openai.Client
	tracer     = otel.Tracer("llm-chat-service")
	envVars    = make(map[string]string)
	uiMessages = make(chan string, 100)
)

// loadEnv loads environment variables using multiple fallback methods
func loadEnv() error {
	// Define required environment variables
	required := []string{
		"OPENAI_API_KEY",
		"PHOENIX_COLLECTOR_ENDPOINT",
		"PHOENIX_API_KEY",
		"OTEL_EXPORTER_OTLP_HEADERS",
		"PHOENIX_CLIENT_HEADERS",
	}

	// Try loading .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, checking environment variables")
	}

	// Load and validate environment variables
	missing := []string{}
	for _, key := range required {
		value := os.Getenv(key)
		if value == "" {
			missing = append(missing, key)
			continue
		}
		envVars[key] = strings.TrimSpace(value)
	}

	// Check for missing variables
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %v", missing)
	}

	// Validate OpenAI API key format
	if !strings.HasPrefix(envVars["OPENAI_API_KEY"], "sk-") {
		return fmt.Errorf("OPENAI_API_KEY must start with 'sk-'")
	}

	// Print loaded variables (masked)
	log.Println("Environment variables loaded successfully:")
	for key, value := range envVars {
		if strings.Contains(strings.ToLower(key), "key") {
			log.Printf("  %s: %s", key, maskSecret(value))
		} else {
			log.Printf("  %s: %s", key, value)
		}
	}

	return nil
}

func maskSecret(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return fmt.Sprintf("%s...%s", s[:4], s[len(s)-4:])
}

func init() {
	if err := loadEnv(); err != nil {
		log.Fatalf("Environment setup failed: %v", err)
	}

	// Initialize OpenAI client
	client = openai.NewClient(envVars["OPENAI_API_KEY"])
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}
	tmpl.Execute(w, nil)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	log.Println("Received chat request")
	ctx, span := tracer.Start(r.Context(), "chat_completion")
	defer span.End()

	// Add service name attribute
	span.SetAttributes(attribute.String("service.name", "Go LLM Gateway Service"))

	if r.Method != http.MethodPost {
		log.Printf("Invalid method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Debug request body
	var body []byte
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading body: %v", err)
		http.Error(w, "Error reading request", http.StatusBadRequest)
		return
	}
	log.Printf("Raw request body: %s", string(body))

	// Parse request
	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Error parsing JSON: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("Parsed request message: %s", req.Message)

	// Add input text to span
	span.SetAttributes(attribute.String("input.text", req.Message))

	// Create ChatGPT request
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: req.Message,
				},
			},
			ResponseFormat: openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeText,
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
	response := resp.Choices[0].Message.Content
	span.SetAttributes(attribute.String("output.text", response))
	log.Printf("Generated response: %s", response)

	// Send response back to client
	w.Header().Set("Content-Type", "application/json")
	responseJSON := ChatResponse{
		Response: response,
	}
	if err := json.NewEncoder(w).Encode(responseJSON); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
	log.Println("Successfully sent response")
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create channel for client disconnect
	notify := w.(http.CloseNotifier).CloseNotify()

	// Send messages to UI
	for {
		select {
		case msg := <-uiMessages:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-notify:
			return
		}
	}
}

func tracingUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		propagator := propagation.TraceContext{}
		carrier := propagation.MapCarrier{}
		for k, v := range md {
			if len(v) > 0 {
				carrier[k] = v[0]
			}
		}
		ctx = propagator.Extract(ctx, carrier)
	}
	return handler(ctx, req)
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
	http.HandleFunc("/events", handleEvents)

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.UnaryInterceptor(tracingUnaryInterceptor),
	)
	pb.RegisterLLMServiceServer(s, &llmServer{})
	go func() {
		log.Printf("gRPC server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	fmt.Println("\n===========================================")
	fmt.Println("🚀 Server starting at http://localhost:8080")
	fmt.Println("===========================================")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

# Two-Service Project

This project consists of two microservices:
1. Python Service - A simple Python service
2. Go LLM UI Service - A Go service with a web UI for LLM interactions

## Project Structure
- `python_service/` - Contains the Python service
- `go_llm_service/` - Contains the Go service with UI
  - `main.go` - Main Go server file with LLM chat implementation
  - `otel.go` - OpenTelemetry configuration for tracing
  - `static/` - Static assets (CSS, JavaScript)
  - `templates/` - HTML templates

## Running the Services

### Go LLM Service

1. Make sure you're using Go 1.20:
   ```bash
   go version  # Should show go1.20.x
   ```

2. Create a `.env` file in the root directory with your configuration:
   ```env
   # OpenAI Configuration
   OPENAI_API_KEY=your-openai-api-key  # Must be 51 characters starting with 'sk-'

   # Phoenix/OpenTelemetry Configuration
   OTEL_EXPORTER_OTLP_HEADERS=api_key=your-phoenix-api-key
   PHOENIX_CLIENT_HEADERS=api_key=your-phoenix-api-key
   PHOENIX_COLLECTOR_ENDPOINT=https://app.phoenix.arize.com
   PHOENIX_API_KEY=your-phoenix-api-key
   ```

3. Navigate to the Go service directory:
   ```bash
   cd go_llm_service
   ```

4. Clean and initialize the Go modules:
   ```bash
   # Remove existing module files
   rm -f go.sum
   go clean -modcache

   # Install dependencies (this step is crucial)
   go mod tidy

   # Verify modules are correct
   go mod verify
   ```

5. Run the Go server:
   ```bash
   go run main.go otel.go
   ```
   You should see output like:
   ```
   Server starting at http://localhost:8080
   ```

6. Open your web browser and visit the printed URL:
   ```
   http://localhost:8080
   ```

### Troubleshooting

1. OpenAI API Key:
   - Must be 51 characters long
   - Must start with 'sk-'
   - No spaces or newlines
   - If you see "Warning: API key length is X", verify your key

2. Module Issues:
   - Always run `go mod tidy` after cleaning
   - Make sure you're in the `go_llm_service` directory
   - If you see dependency errors, repeat steps 4-5

3. Environment Variables:
   - `.env` file should be in the root directory
   - No spaces around the '=' signs
   - No quotes around values

### Features
The UI provides a chat interface where you can:
- Type messages in the text area
- Send messages using the "Send" button or Enter key
- View the conversation history in the chat window
- Interact with OpenAI's GPT model through the chat interface

### Monitoring
The service sends traces to Arize Phoenix, where you can:
- Monitor LLM interactions
- Track latency and errors
- View input/output pairs
- Analyze conversation patterns

### Python Service
[To be implemented]
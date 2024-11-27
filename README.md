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

#### Initial Setup

1. Make sure you're using Go 1.20:
   ```bash
   go version  # Should show go1.20.x
   ```

2. Create a `.env` file in the go_llm_service directory:
   ```bash
   cd go_llm_service
   cat > .env << 'EOF'
   # OpenAI Configuration
   OPENAI_API_KEY=your-openai-api-key  # Must be 51 characters starting with 'sk-'

   # Phoenix/OpenTelemetry Configuration
   OTEL_EXPORTER_OTLP_HEADERS=api_key=your-phoenix-api-key
   PHOENIX_CLIENT_HEADERS=api_key=your-phoenix-api-key
   PHOENIX_COLLECTOR_ENDPOINT=https://app.phoenix.arize.com
   PHOENIX_API_KEY=your-phoenix-api-key
   EOF
   ```

#### Running the Service

IMPORTANT: Run these commands in order whenever you:
- Change environment variables
- Update Go code
- Modify dependencies

```bash
# 1. Navigate to the service directory
cd go_llm_service

# 2. Clean up existing builds
rm -f go.sum
go clean -modcache

# 3. Install dependencies
go mod tidy

# 4. Run the service
go run main.go otel.go grpc.go
```

You should see output exactly like this:
```
Environment variables loaded successfully:
  OPENAI_API_KEY: sk-z...2Ghs
  PHOENIX_COLLECTOR_ENDPOINT: https://app.phoenix.arize.com
  PHOENIX_API_KEY: KQ9P...xy2y
  OTEL_EXPORTER_OTLP_HEADERS: api_key=KQ9Pzpz15ydRbpGepdv4:9yiixy2y
  PHOENIX_CLIENT_HEADERS: api_key=KQ9Pzpz15ydRbpGepdv4:9yiixy2y
Initializing tracer...
Setting up OTLP with headers configured
OTLP client created
Exporter created
Resource created
TracerProvider created
Tracer initialization complete

===========================================
🚀 Server starting at http://localhost:8080
===========================================
```

If you don't see this exact output, something is wrong. Check the troubleshooting section.

#### Troubleshooting

1. Environment Variables:
   - `.env` file must be in the `go_llm_service` directory
   - No spaces around the '=' signs
   - No quotes around values
   - No trailing spaces
   - If variables aren't loading, run the commands in order again

2. OpenAI API Key:
   - Must be 51 characters long
   - Must start with 'sk-'
   - No spaces or newlines
   - If you see "Warning: API key length is X", verify your key

3. Common Issues:
   - "undefined: initTracer" - Make sure to run both files: `go run main.go otel.go`
   - "missing go.sum entry" - Run `go mod tidy`
   - Empty environment variables - Check `.env` file location and format

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
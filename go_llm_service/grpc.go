package main

import (
	"context"
	pb "go_llm_service/proto"

	"fmt"
	"log"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

type llmServer struct {
	pb.UnimplementedLLMServiceServer
}

func (s *llmServer) ProcessText(ctx context.Context, req *pb.TextRequest) (*pb.TextResponse, error) {
	// Extract trace context from metadata
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

	// Create span with extracted context
	ctx, span := tracer.Start(ctx, "grpc_process_text")
	defer span.End()

	// Add service name and input text to span
	span.SetAttributes(
		attribute.String("service.name", "Go LLM Gateway Service"),
		attribute.String("input.text", req.Text),
	)

	// Log incoming gRPC request
	log.Printf("Received gRPC request from Python service: %s", req.Text)

	// Send incoming message to UI with clear formatting
	uiMessage := fmt.Sprintf("[From Python Service] Input: %s", req.Text)
	uiMessages <- uiMessage

	// Use existing OpenAI client
	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: req.Text,
				},
			},
			ResponseFormat: openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeText,
			},
		},
	)

	if err != nil {
		span.RecordError(err)
		return nil, err
	}

	response := resp.Choices[0].Message.Content

	// Send complete response to UI with HTML line breaks
	uiMessage = fmt.Sprintf("[From Python Service] Complete Response:<br>%s",
		strings.ReplaceAll(response, "\n", "<br>"))
	uiMessages <- uiMessage

	span.SetAttributes(attribute.String("output.text", response))

	// Log the response
	log.Printf("Sending gRPC response back to Python service: %s", response)

	return &pb.TextResponse{
		Response: response,
	}, nil
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"google.golang.org/grpc"
)

// UISpanExporter sends spans to both UI and console
type UISpanExporter struct {
	messages chan<- string
}

func (e *UISpanExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	for _, span := range spans {
		// Convert attributes to a map for better JSON serialization
		attrs := make(map[string]string)
		for _, kv := range span.Attributes() {
			attrs[string(kv.Key)] = kv.Value.Emit()
		}

		// Format span data as JSON
		spanData := map[string]interface{}{
			"type":       "span",
			"name":       span.Name(),
			"trace_id":   span.SpanContext().TraceID().String(),
			"span_id":    span.SpanContext().SpanID().String(),
			"start":      span.StartTime().Format(time.RFC3339),
			"end":        span.EndTime().Format(time.RFC3339),
			"attributes": attrs, // Use our converted attributes
		}

		// Convert to JSON
		jsonData, err := json.Marshal(spanData)
		if err != nil {
			fmt.Printf("Error marshaling span data: %v\n", err)
			continue
		}

		// Send to UI
		e.messages <- string(jsonData)

		// Also print to console for debugging
		fmt.Printf("Span: %s\n", span.Name())
		fmt.Printf("  TraceID: %s\n", span.SpanContext().TraceID().String())
		fmt.Printf("  SpanID: %s\n", span.SpanContext().SpanID().String())
		fmt.Printf("  Attributes:\n")
		for _, kv := range span.Attributes() {
			fmt.Printf("    %s: %s\n", kv.Key, kv.Value.Emit())
		}
		fmt.Println()
	}
	return nil
}

func (e *UISpanExporter) Shutdown(ctx context.Context) error {
	return nil
}

func initTracer() (func(context.Context) error, error) {
	fmt.Println("Initializing tracer...")

	// Create UI exporter
	uiExporter := &UISpanExporter{
		messages: uiMessages,
	}

	endpoint := envVars["PHOENIX_COLLECTOR_ENDPOINT"]
	apiKey := envVars["PHOENIX_API_KEY"]

	headers := map[string]string{
		"api_key": apiKey,
	}
	fmt.Printf("Setting up OTLP with headers configured\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	phoenixClient := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithTimeout(5*time.Second)),
	)
	fmt.Println("OTLP client created")

	phoenixExp, err := otlptrace.New(ctx, phoenixClient)
	if err != nil {
		fmt.Printf("Error creating Phoenix exporter: %v\n", err)
	}
	fmt.Println("Phoenix exporter created")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("llm-chat-service"),
		),
	)
	if err != nil {
		fmt.Printf("Error creating resource: %v\n", err)
		return nil, err
	}
	fmt.Println("Resource created")

	// Create processors for both exporters
	var processors []sdktrace.SpanProcessor
	processors = append(processors, sdktrace.NewBatchSpanProcessor(uiExporter))
	if phoenixExp != nil {
		processors = append(processors, sdktrace.NewBatchSpanProcessor(phoenixExp))
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
	)

	// Register all processors
	for _, processor := range processors {
		tp.RegisterSpanProcessor(processor)
	}

	fmt.Println("TracerProvider created")

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	fmt.Println("Tracer initialization complete")

	return tp.Shutdown, nil
}

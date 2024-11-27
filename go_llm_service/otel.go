package main

import (
	"context"
	"fmt"
	"log"
	"os"
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

func initTracer() (func(context.Context) error, error) {
	fmt.Println("Initializing tracer...")

	// Print all environment variables for debugging
	fmt.Println("Environment variables:")
	fmt.Printf("PHOENIX_COLLECTOR_ENDPOINT: %s\n", os.Getenv("PHOENIX_COLLECTOR_ENDPOINT"))
	fmt.Printf("PHOENIX_API_KEY: %s\n", os.Getenv("PHOENIX_API_KEY"))
	fmt.Printf("OTEL_EXPORTER_OTLP_HEADERS: %s\n", os.Getenv("OTEL_EXPORTER_OTLP_HEADERS"))

	endpoint := os.Getenv("PHOENIX_COLLECTOR_ENDPOINT")
	if endpoint == "" {
		log.Fatal("PHOENIX_COLLECTOR_ENDPOINT is not set")
	}
	fmt.Printf("Using Phoenix endpoint: %s\n", endpoint)

	apiKey := os.Getenv("PHOENIX_API_KEY")
	if apiKey == "" {
		log.Fatal("PHOENIX_API_KEY is not set")
	}
	fmt.Printf("Phoenix API Key found with length: %d\n", len(apiKey))

	headers := map[string]string{
		"api_key": apiKey,
	}
	fmt.Printf("Setting up OTLP with headers configured\n")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := otlptracegrpc.NewClient(
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithTimeout(5*time.Second)),
	)
	fmt.Println("OTLP client created")

	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		fmt.Printf("Error creating exporter: %v\n", err)
		return nil, err
	}
	fmt.Println("Exporter created")

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

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	fmt.Println("TracerProvider created")

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	fmt.Println("Tracer initialization complete")

	return tp.Shutdown, nil
}

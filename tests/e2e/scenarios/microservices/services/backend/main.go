package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer      trace.Tracer
	db          *sql.DB
	redisClient *redis.Client
)

type Response struct {
	Service   string    `json:"service"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

type DataItem struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Value     float64   `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

func initTracer() func() {
	ctx := context.Background()

	exporter, err := otlptrace.New(
		ctx,
		otlptracegrpc.NewClient(
			otlptracegrpc.WithInsecure(),
			otlptracegrpc.WithEndpoint(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")),
		),
	)
	if err != nil {
		log.Fatal("Failed to create exporter:", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("backend"),
			semconv.ServiceVersion("1.0.0"),
			attribute.String("deployment.environment", "e2e"),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		log.Fatal("Failed to create resource:", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer("backend")

	return func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}

func initDatabase() {
	var err error
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/e2e?sslmode=disable"
	}

	// Retry connection
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dbURL)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Printf("Database connection attempt %d failed: %v", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Create table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS data_items (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			value FLOAT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	log.Println("Database initialized successfully")
}

func initRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Failed to parse Redis URL:", err)
	}

	redisClient = redis.NewClient(opt)

	// Test connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	log.Println("Redis connected successfully")
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "get-data")
	defer span.End()

	// Extract trace context from incoming request
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

	// Check cache first
	cacheKey := "data:latest"
	cachedData, err := getCachedData(ctx, cacheKey)
	if err == nil && cachedData != nil {
		span.SetAttributes(attribute.Bool("cache.hit", true))
		response := Response{
			Service:   "backend",
			Message:   "Data from cache",
			Timestamp: time.Now(),
			Data:      cachedData,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	span.SetAttributes(attribute.Bool("cache.hit", false))

	// Fetch from database
	data, err := fetchDataFromDB(ctx)
	if err != nil {
		span.RecordError(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache the result
	if err := setCachedData(ctx, cacheKey, data); err != nil {
		span.RecordError(err)
		// Continue even if caching fails
	}

	response := Response{
		Service:   "backend",
		Message:   "Data from database",
		Timestamp: time.Now(),
		Data:      data,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getCachedData(ctx context.Context, key string) ([]DataItem, error) {
	_, span := tracer.Start(ctx, "redis-get")
	defer span.End()

	val, err := redisClient.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var data []DataItem
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}

	return data, nil
}

func setCachedData(ctx context.Context, key string, data []DataItem) error {
	_, span := tracer.Start(ctx, "redis-set")
	defer span.End()

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return redisClient.Set(ctx, key, jsonData, 30*time.Second).Err()
}

func fetchDataFromDB(ctx context.Context) ([]DataItem, error) {
	_, span := tracer.Start(ctx, "db-query")
	defer span.End()

	// Simulate slow query
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

	rows, err := db.QueryContext(ctx, "SELECT id, name, value, created_at FROM data_items ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []DataItem
	for rows.Next() {
		var item DataItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Value, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	span.SetAttributes(attribute.Int("db.rows_returned", len(items)))

	// If no data, create some
	if len(items) == 0 {
		items = generateSampleData(ctx)
	}

	return items, nil
}

func generateSampleData(ctx context.Context) []DataItem {
	_, span := tracer.Start(ctx, "generate-data")
	defer span.End()

	var items []DataItem
	for i := 0; i < 5; i++ {
		item := DataItem{
			Name:  fmt.Sprintf("Item-%d", rand.Intn(1000)),
			Value: rand.Float64() * 100,
		}

		err := db.QueryRowContext(ctx,
			"INSERT INTO data_items (name, value) VALUES ($1, $2) RETURNING id, created_at",
			item.Name, item.Value,
		).Scan(&item.ID, &item.CreatedAt)

		if err != nil {
			span.RecordError(err)
			continue
		}

		items = append(items, item)
	}

	return items
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "health-check")
	defer span.End()

	// Check database
	if err := db.PingContext(ctx); err != nil {
		span.RecordError(err)
		http.Error(w, "Database unhealthy", http.StatusServiceUnavailable)
		return
	}

	// Check Redis
	if err := redisClient.Ping(ctx).Err(); err != nil {
		span.RecordError(err)
		http.Error(w, "Redis unhealthy", http.StatusServiceUnavailable)
		return
	}

	response := Response{
		Service:   "backend",
		Message:   "healthy",
		Timestamp: time.Now(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	shutdown := initTracer()
	defer shutdown()

	initDatabase()
	initRedis()

	// Setup routes with instrumentation
	mux := http.NewServeMux()
	mux.HandleFunc("/api/data", dataHandler)
	mux.HandleFunc("/health", healthHandler)

	// Wrap with OpenTelemetry instrumentation
	handler := otelhttp.NewHandler(mux, "backend-server")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Backend service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal("Server failed:", err)
	}
}
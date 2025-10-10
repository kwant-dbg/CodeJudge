package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	commonauth "codejudge/common/auth"
	"codejudge/common/dbutil"
	"codejudge/common/env"
	"codejudge/common/health"
	"codejudge/common/httpx"
	"codejudge/common/redisutil"

	"codejudge/monolith-service/handlers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	logger    *zap.Logger
	dbManager *dbutil.ConnectionManager
	rdb       *redis.Client
	jwtSecret []byte
	ctx       = context.Background()
)

func performHealthCheck() {
	port := env.Get("PORT", "8080")
	healthURL := fmt.Sprintf("http://localhost:%s/health", port)
	
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	
	resp, err := client.Get(healthURL)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
	
	fmt.Println("Health check passed")
	os.Exit(0)
}

func initJWTSecret() error {
	// In production, use environment variable or Azure Key Vault
	secretEnv := env.Get("JWT_SECRET", "")
	if secretEnv != "" {
		jwtSecret = []byte(secretEnv)
		logger.Info("JWT secret loaded from environment")
		return nil
	} else {
		// Generate random secret for development
		jwtSecret = make([]byte, 32)
		if _, err := rand.Read(jwtSecret); err != nil {
			return fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		logger.Info("Generated random JWT secret for development")
		return nil
	}
}

func main() {
	// Handle health check flag for Docker health checks
	healthCheck := flag.Bool("health-check", false, "Perform health check and exit")
	flag.Parse()

	if *healthCheck {
		performHealthCheck()
		return
	}

	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer logger.Sync()

	logger.Info("Starting CodeJudge Monolith Service")

	// Load JWT secret
	err = initJWTSecret()
	if err != nil {
		logger.Fatal("Failed to load JWT secret", zap.Error(err))
	}

	// Connect to database
	connectDB()
	defer dbManager.Close()

	// Connect to Redis
	connectRedis()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(logger, dbManager, jwtSecret)
	problemsHandler := handlers.NewProblemsHandler(logger, dbManager)
	submissionsHandler := handlers.NewSubmissionsHandler(logger, dbManager, rdb)
	plagiarismHandler := handlers.NewPlagiarismHandler(logger, dbManager, rdb)

	// Create database tables
	authHandler.CreateTables()
	problemsHandler.CreateTables()
	problemsHandler.PrepareStatements()
	submissionsHandler.CreateTables()
	plagiarismHandler.CreateTables()

	// Start background workers
	plagiarismHandler.StartWorker()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httpx.RecoveryMiddleware(logger))

	// CORS middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // In production, specify your frontend domain
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health endpoints (no auth required)
	r.Get("/health", health.HealthHandler())
	r.Get("/ready", health.ReadyHandler(func(ctx context.Context) error {
		// Check both database and Redis connectivity
		if err := dbManager.GetDB().PingContext(ctx); err != nil {
			return fmt.Errorf("database ping failed: %w", err)
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis ping failed: %w", err)
		}
		return nil
	}))

	// Public auth endpoints
	r.Route("/api/auth", func(authRouter chi.Router) {
		authRouter.Post("/register", authHandler.Register)
		authRouter.Post("/login", authHandler.Login)
		authRouter.Post("/validate", authHandler.ValidateToken)
		
		// Protected auth endpoints
		authRouter.Group(func(protected chi.Router) {
			protected.Use(commonauth.RequireAuth(jwtSecret, logger))
			protected.Get("/me", authHandler.Me)
		})
	})

	// Protected API endpoints (require authentication)
	r.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(commonauth.RequireAuth(jwtSecret, logger))

		// Problems endpoints
		apiRouter.Route("/problems", func(problemsRouter chi.Router) {
			problemsRouter.Get("/", problemsHandler.GetProblems)
			problemsRouter.Post("/", problemsHandler.CreateProblem)
			problemsRouter.Get("/{id}", problemsHandler.GetProblem)
			problemsRouter.Post("/{id}/testcases", problemsHandler.CreateTestCase)
		})

		// Submissions endpoints
		apiRouter.Post("/submissions", submissionsHandler.CreateSubmission)

		// Plagiarism endpoints
		apiRouter.Route("/plagiarism", func(plagiarismRouter chi.Router) {
			plagiarismRouter.Get("/reports", plagiarismHandler.GetReports)
		})
	})

	// Serve static files (for the frontend)
	r.Handle("/*", http.FileServer(http.Dir("./static/")))

	// Get port from environment or use default
	port := env.Get("PORT", "8080")
	
	// Create server
	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	logger.Info("CodeJudge Monolith starting", zap.String("port", port))
	
	// Start server with graceful shutdown
	httpx.StartServerWithGracefulShutdown(server, logger, 30*time.Second)
}

func connectDB() {
	databaseURL := env.Get("DATABASE_URL", "")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL not set")
	}
	dbManager = dbutil.ConnectManagerWithRetry(logger, databaseURL, 5, 2*time.Second)
	logger.Info("Database connected successfully")
}

func connectRedis() {
	redisURL := env.Get("REDIS_URL", "")
	if redisURL == "" {
		logger.Fatal("REDIS_URL not set")
	}
	rdb = redisutil.ConnectWithRetry(ctx, logger, redisURL, 5, 2*time.Second)
	logger.Info("Redis connected successfully")
}
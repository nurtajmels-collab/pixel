package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"

	"pixellife-tracker/internal/assistant"
	"pixellife-tracker/internal/bot"
	"pixellife-tracker/internal/config"
	"pixellife-tracker/internal/db"
	"pixellife-tracker/internal/handlers"
	"pixellife-tracker/internal/services"
)

func main() {
	cfg := config.Load()

	database, err := db.Init(cfg.DBPath)
	if err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer database.Close()

	publicDir, publicErr := config.ResolvePublicDir(".", "./public", "../frontend/dist")
	if publicErr != nil {
		log.Printf("Frontend assets not found yet; continuing without static UI. Build frontend first: %v", publicErr)
		publicDir = "./public"
	}

	engine := html.New(publicDir, ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.FrontendURL,
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	userSvc := services.NewUserService(database)
	studySvc := services.NewStudyService(database)
	mediaSvc := services.NewMediaService(database)
	sportSvc := services.NewSportService(database)
	sleepSvc := services.NewSleepService(database)
	hobbySvc := services.NewHobbyService(database)
	f1Svc := services.NewF1Service(database)
	radarSvc := services.NewRadarService(database)

	authHandler := handlers.NewAuthHandler(userSvc, cfg)
	studyHandler := handlers.NewStudyHandler(studySvc)
	mediaHandler := handlers.NewMediaHandler(mediaSvc)
	sportHandler := handlers.NewSportHandler(sportSvc)
	sleepHandler := handlers.NewSleepHandler(sleepSvc)
	hobbyHandler := handlers.NewHobbyHandler(hobbySvc)
	f1Handler := handlers.NewF1Handler(f1Svc)
	dashboardHandler := handlers.NewDashboardHandler(radarSvc)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	alfred := assistant.NewAlfredService(database, cfg.GeminiAPIKey, cfg.GeminiModel)
	telegram := bot.New(cfg.TelegramToken, userSvc, sleepSvc, sportSvc, mediaSvc, database, alfred, cfg.PersonalAuth)
	telegramErr := telegram.Validate(ctx)
	if telegramErr != nil {
		log.Printf("Telegram disabled: %v", telegramErr)
	} else {
		log.Println("Telegram bot connected")
		go telegram.Start(ctx)
		go telegram.StartReminderWorker(ctx)
	}
	assistantClient := assistant.New(database, cfg.GeminiAPIKey, cfg.GeminiModel, telegram)
	go assistantClient.Start(ctx)

	api := app.Group("/api")
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "telegram_configured": cfg.TelegramToken != "", "gemini_configured": cfg.GeminiAPIKey != "", "personal_auth": cfg.PersonalAuth})
	})
	api.Post("/auth/telegram", authHandler.TelegramAuth)

	protected := api.Group("/", handlers.AuthMiddleware(cfg))
	protected.Post("/assistant/report", func(c *fiber.Ctx) error {
		go assistantClient.GenerateNow(context.Background())
		return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"status": "queued", "destination": "telegram"})
	})
	dashboardHandler.RegisterRoutes(protected.Group("/dashboard"))
	studyHandler.RegisterRoutes(protected.Group("/study"))
	mediaHandler.RegisterRoutes(protected.Group("/media"))
	sportHandler.RegisterRoutes(protected.Group("/sport"))
	sleepHandler.RegisterRoutes(protected.Group("/sleep"))
	hobbyHandler.RegisterRoutes(protected.Group("/hobby"))
	f1Handler.RegisterRoutes(protected.Group("/f1"))

	if _, err := os.Stat(publicDir); err == nil {
		app.Static("/", publicDir)
		app.Get("*", func(c *fiber.Ctx) error {
			return c.Render("index", fiber.Map{})
		})
	}

	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
}

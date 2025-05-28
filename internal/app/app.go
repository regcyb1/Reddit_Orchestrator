// internal/app/app.go
package app

import (
	"fmt"
	"log"

	"github.com/ersauravadhikari/blueberry-go/blueberry"
	"github.com/ersauravadhikari/blueberry-go/blueberry/store"

	"reddit-orchestrator/internal/client"
	"reddit-orchestrator/internal/config"
	"reddit-orchestrator/internal/processor"
	"reddit-orchestrator/internal/storage"
	"reddit-orchestrator/internal/tasks"
)

type App struct {
	Config      *config.Config
	BlueBerry   *blueberry.BlueBerry
	Storage     storage.StorageInterface
	Client      client.IngestionClientInterface
	Processor   processor.ProcessorInterface
	TaskManager tasks.TaskManagerInterface
}

func Initialize() (*App, error) {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	mongoStore, err := storage.NewMongoStorage(cfg.MongoDBURI, cfg.DatabaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB storage: %w", err)
	}

	schedulerDBName := cfg.DatabaseName
	blueBerryStore, err := store.NewMongoDB(cfg.MongoDBURI, schedulerDBName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BlueBerry MongoDB store: %w", err)
	}

	bb := blueberry.NewBlueBerryInstance(blueBerryStore)

	// Add authentication (required)
	if cfg.WebAuthUser == "" || cfg.WebAuthPassword == "" {
		return nil, fmt.Errorf("web authentication credentials are required")
	}
	bb.AddWebOnlyPasswordAuth(cfg.WebAuthUser, cfg.WebAuthPassword)

	ingestionClient := client.NewIngestionClient(cfg.IngestionAPIURL, cfg.RequestTimeout)

	dataProcessor := processor.NewProcessor()

	taskManager := tasks.NewSubredditTaskManager(bb, mongoStore, ingestionClient, dataProcessor, cfg)

	app := &App{
		Config:      cfg,
		BlueBerry:   bb,
		Storage:     mongoStore,
		Client:      ingestionClient,
		Processor:   dataProcessor,
		TaskManager: taskManager,
	}

	if err := app.TaskManager.RegisterTasks(); err != nil {
		return nil, fmt.Errorf("failed to register tasks: %w", err)
	}

	return app, nil
}

func (a *App) Start() error {
	log.Printf("Initializing task scheduler...")
	a.BlueBerry.InitTaskScheduler()

	log.Printf("Starting API server on port %s...", a.Config.ServerPort)
	a.BlueBerry.RunAPI(a.Config.ServerPort)

	return nil
}

func (a *App) Shutdown() {
	log.Println("Shutting down orchestrator...")
	a.BlueBerry.Shutdown()
	if a.Storage != nil {
		a.Storage.Close()
	}
}
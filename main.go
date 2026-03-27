// main.go (Wails)
package main

import (
	"embed"
	"log"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"p2p_market_data/pkg/config"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Load configuration; both Wails options and the App struct share this instance.
	cfg, err := loadWailsConfig()
	if err != nil {
		log.Printf("Warning: could not load config (%v), falling back to defaults", err)
		cfg, err = config.LoadDefaults()
		if err != nil {
			log.Fatalf("Failed to load default configuration: %v", err)
		}
	}

	// Create an instance of the app structure
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Create application with options
	err = wails.Run(&options.App{
		Title:     "P2P Market Data",
		Width:     1280,
		Height:    930,
		MinWidth:  1024,
		MinHeight: 768,

		// Asset configuration
		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		// Bind our application struct
		Bind: []interface{}{
			app,
		},

		// Application lifecycle
		OnStartup:     app.startup,
		OnDomReady:    app.domReady,
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,

		// Log level driven by configuration, not hardcoded.
		LogLevel: wailsLogLevel(cfg),

		// Window configuration
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},

		// Mac configuration
		Mac: &mac.Options{
			TitleBar: &mac.TitleBar{
				TitlebarAppearsTransparent: true,
				HideTitle:                  false,
				HideTitleBar:               false,
				FullSizeContent:            false,
				UseToolbar:                 false,
				HideToolbarSeparator:       true,
			},
			WebviewIsTransparent: true,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "P2P Market Data",
				Message: "A peer-to-peer market data platform",
				Icon:    icon,
			},
		},
	})

	if err != nil {
		log.Fatal(err)
	}
}

// loadWailsConfig tries to load configuration from well-known paths relative to
// the working directory. If no config file is found it falls back to defaults.
func loadWailsConfig() (*config.Config, error) {
	candidates := []string{
		"./config.yaml",
		"./config/config.yaml",
		"./config/db_config.yaml",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return config.Load(path)
		}
	}
	return config.LoadDefaults()
}

// wailsLogLevel maps the config log-level string to the Wails logger.LogLevel
// type.  Defaults to INFO when the level is unrecognised or the config is nil.
func wailsLogLevel(cfg *config.Config) logger.LogLevel {
	if cfg == nil {
		return logger.INFO
	}
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		return logger.DEBUG
	case "warn", "warning":
		return logger.WARNING
	case "error":
		return logger.ERROR
	default:
		return logger.INFO
	}
}

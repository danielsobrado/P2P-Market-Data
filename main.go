// main.go (Wails)
package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var icon []byte

func main() {
	// Create configuration with defaults
	// cfg := config.DefaultConfig()

	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
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

		// Enable dev tools in debug mode
		LogLevel: determineLogLevel(),

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

func determineLogLevel() logger.LogLevel {
	// You could make this configurable via environment variables
	return logger.DEBUG
}

package app

import (
	"embed"

	"github.com/chenyang-zz/cove/internal/services"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	Name        = "Cove"
	Version     = "0.1.0"
	Description = "Cove desktop application"
)

func New(assets embed.FS) *application.App {
	app := application.New(application.Options{
		Name:        Name,
		Description: Description,
		Services: []application.Service{
			application.NewService(services.NewAppInfoService(Name, Version)),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  Name,
		Width:  1100,
		Height: 760,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(245, 247, 250),
		URL:              "/",
	})

	return app
}

package ui

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/ui/pages"
)

// App is the root component
type App struct {
	app.Compo
}

// Render renders the main application
func (a *App) Render() app.UI {
	return app.Div().
		Class("app-container").
		Body(
			app.Header().
				Class("app-header").
				Body(
					app.H1().Text("Daggerheart Combat Simulator"),
					app.P().
						Class("subtitle").
						Text("Monte Carlo combat analysis for Daggerheart RPG"),
				),
			app.Main().
				Class("app-main").
				Body(
					&pages.SimulatorPage{},
				),
			app.Footer().
				Class("app-footer").
				Body(
					app.P().Text("Built with Go + WASM"),
				),
		)
}

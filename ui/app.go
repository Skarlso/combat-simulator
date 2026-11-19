package ui

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/ui/pages"
)

// App is the root component
type App struct {
	app.Compo

	theme string
}

// OnMount restores the saved theme preference
func (a *App) OnMount(ctx app.Context) {
	_ = ctx.LocalStorage().Get("theme", &a.theme)
	a.applyTheme()
}

// Render renders the main application
func (a *App) Render() app.UI {
	return app.Div().
		Class("app-container").
		Body(
			app.Header().
				Class("app-header").
				Body(
					app.Div().
						Class("brand").
						Body(
							app.Span().Class("brand-mark"),
							app.H1().Text("Daggerheart Combat Simulator"),
							app.Span().Class("brand-meta").Text("Monte Carlo · SRD 2.0"),
						),
					app.Div().
						Class("header-actions").
						Body(
							app.Button().
								Class("btn btn-ghost btn-small").
								Title("Cycle theme: system, light, dark").
								Text(a.themeLabel()).
								OnClick(a.cycleTheme),
						),
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

func (a *App) themeLabel() string {
	switch a.theme {
	case "light":
		return "☀ Light"
	case "dark":
		return "☾ Dark"
	default:
		return "◐ Auto"
	}
}

func (a *App) cycleTheme(ctx app.Context, e app.Event) {
	switch a.theme {
	case "":
		a.theme = "light"
	case "light":
		a.theme = "dark"
	default:
		a.theme = ""
	}
	_ = ctx.LocalStorage().Set("theme", a.theme)
	a.applyTheme()
}

// applyTheme sets data-theme on <html> so CSS color-scheme can override the OS setting
func (a *App) applyTheme() {
	root := app.Window().Get("document").Get("documentElement")
	if a.theme == "" {
		root.Call("removeAttribute", "data-theme")
		return
	}
	root.Call("setAttribute", "data-theme", a.theme)
}

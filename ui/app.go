package ui

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/ui/components"
	"github.com/skarlso/combat-simulator/ui/pages"
)

// Title must not carry the Daggerheart name (DPCGL 2.5a); Tagline is where the licence wants it,
// next to "Compatible" (2.5c)
const (
	Title   = "Combat Simulator"
	Tagline = "Daggerheart™ Compatible"
)

// App is the root component
type App struct {
	app.Compo

	theme    string
	helpOpen bool
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
							app.H1().Text(Title),
							app.Span().Class("brand-meta").Text(Tagline+" · SRD 2.0"),
						),
					app.Div().
						Class("header-actions").
						Body(
							app.Button().
								Class("btn btn-ghost btn-small").
								Title("How to use the simulator").
								Text("? Help").
								OnClick(func(ctx app.Context, e app.Event) { a.helpOpen = true }),
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
					// Wording set by the Darrington Press Community Gaming License, section 4
					app.P().Body(
						app.Text("This product includes materials from the Daggerheart System Reference Document 2.0, © Critical Role, LLC. under the terms of the "),
						app.A().Href("https://www.darringtonpress.com/license").Text("Darrington Press Community Gaming (DPCGL) License"),
						app.Text(". More information can be found at "),
						app.A().Href("https://www.daggerheart.com").Text("https://www.daggerheart.com"),
						app.Text(". There are no previous modifications by others."),
					),
				),
			app.If(a.helpOpen, func() app.UI {
				return &components.HelpDialog{OnClose: func(ctx app.Context) { a.helpOpen = false }}
			}),
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

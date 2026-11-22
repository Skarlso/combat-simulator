package main

import (
	"flag"
	"log"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/ui"
)

func main() {
	repo := flag.String("repo", "", "repository name, when the site is served from a GitHub Pages project URL (user.github.io/<repo>/)")
	flag.Parse()

	app.Route("/", func() app.Composer {
		return &ui.App{}
	})

	app.RunWhenOnBrowser()

	handler := &app.Handler{
		Name:         ui.Title,
		Title:        ui.Title,
		Description:  "Monte Carlo combat simulation, " + ui.Tagline,
		Styles:       []string{"/web/static/app.css"},
		Icon:         app.Icon{Default: "/web/static/icon.png"},
		LoadingLabel: "Loading " + ui.Title + "...",
		RawHeaders:   []string{`<meta name="viewport" content="width=device-width, initial-scale=1">`},
	}
	// A project site lives under /<repo>/, so every resource path needs that prefix
	if *repo != "" {
		handler.Resources = app.GitHubPages(*repo)
	}

	if err := app.GenerateStaticWebsite("docs", handler); err != nil {
		log.Fatalf("Failed to generate static website: %v", err)
	}
	log.Println("Static website generated in docs/; make static-pages adds web/app.wasm and web/static to it")
}

package main

import (
	"log"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/ui"
)

func main() {
	log.Println("Generating static website for GitHub Pages...")

	// Register the app component
	app.Route("/", func() app.Composer {
		return &ui.App{}
	})

	app.RunWhenOnBrowser()

	// Configure the handler
	handler := &app.Handler{
		Name:        "Daggerheart Combat Simulator",
		Description: "Monte Carlo combat simulation for Daggerheart RPG",
		Styles: []string{
			"/web/static/app.css",
		},
		Icon: app.Icon{
			Default: "/web/static/icon.png",
		},
		LoadingLabel: "Loading Daggerheart Simulator...",
	}

	// Generate static website
	if err := app.GenerateStaticWebsite("docs", handler); err != nil {
		log.Fatalf("Failed to generate static website: %v", err)
	}

	log.Println("Static website generated successfully in docs/")
	log.Println("You can now deploy the docs/ directory to GitHub Pages")
}

package main

import (
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/combat-simulator/ui"
)

func main() {
	app.Route("/", func() app.Composer {
		return &ui.App{}
	})

	// Blocks forever in the browser and returns at once on the server, so what follows is server-only
	app.RunWhenOnBrowser()

	handler := &app.Handler{
		Name:         ui.Title,
		Title:        ui.Title,
		Description:  "Monte Carlo combat simulation, " + ui.Tagline,
		Styles:       []string{"/web/static/app.css"},
		Icon:         app.Icon{Default: "/web/static/icon.png"},
		LoadingLabel: "Loading " + ui.Title + "...",
		Resources:    app.LocalDir("."),
		RawHeaders:   []string{`<meta name="viewport" content="width=device-width, initial-scale=1">`},
	}

	log.Println("Starting server on http://localhost:8080")
	log.Println("Serving from current directory")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}

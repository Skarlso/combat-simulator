package main

import (
	"log"
	"net/http"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"github.com/skarlso/daggerheart-combat-simulator/ui"
)

func main() {
	// Register the app component
	app.Route("/", func() app.Composer {
		return &ui.App{}
	})

	// Once the routes set up, the next thing to do is to either launch the app
	// or the server that serves the app.
	//
	// When executed on the client-side, the RunWhenOnBrowser() function
	// launches the app,  starting a loop that listens for app events and
	// executes client instructions. Since it is a blocking call, the code below
	// it will never be executed.
	//
	// When executed on the server-side, RunWhenOnBrowser() does nothing, which
	// lets room for server implementation without the need for precompiling
	// instructions.
	app.RunWhenOnBrowser()

	// Configure the HTTP handler
	handler := &app.Handler{
		Name:         "Daggerheart Combat Simulator",
		Description:  "Monte Carlo combat simulation for Daggerheart RPG",
		Styles:       []string{"/web/static/app.css"},
		Icon:         app.Icon{Default: "/web/static/icon.png"},
		LoadingLabel: "Loading Daggerheart Simulator...",
		Resources:    app.LocalDir("."),
		RawHeaders:   []string{`<meta name="viewport" content="width=device-width, initial-scale=1">`},
	}

	log.Println("Starting server on http://localhost:8080")
	log.Println("Serving from current directory")

	// Start the server
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}

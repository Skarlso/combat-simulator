package components

import (
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// HelpDialog is the "how do I use this" overlay opened from the header
type HelpDialog struct {
	app.Compo

	OnClose func(ctx app.Context)
}

// OnMount moves focus into the dialog so Escape closes it straight away
func (h *HelpDialog) OnMount(ctx app.Context) {
	ctx.JSSrc().Call("focus")
}

func (h *HelpDialog) close(ctx app.Context, e app.Event) {
	if h.OnClose != nil {
		h.OnClose(ctx)
	}
}

// Render renders the help overlay
func (h *HelpDialog) Render() app.UI {
	return app.Div().
		Class("help-overlay").
		Role("dialog").
		Aria("modal", "true").
		Aria("labelledby", "help-title").
		TabIndex(-1).
		OnKeyDown(func(ctx app.Context, e app.Event) {
			if e.Get("key").String() == "Escape" {
				h.close(ctx, e)
			}
		}).
		OnClick(func(ctx app.Context, e app.Event) {
			// Only a click on the backdrop itself closes, not one inside the card
			if e.Get("target").Equal(ctx.JSSrc()) {
				h.close(ctx, e)
			}
		}).
		Body(
			app.Div().Class("help-card").Body(
				app.Div().Class("panel-header").Body(
					app.H2().ID("help-title").Text("How this works"),
					app.Button().Class("btn btn-small").Text("Close").OnClick(h.close),
				),
				app.P().Class("help-lead").Text("Create a party, add some enemies and pit them against each other a couple of hundred times to see who comes out as the winner."),

				app.Ol().Class("help-steps").Body(
					helpItem("Party", "Add a couple of players with + Add. Under Abilities, the Library has every class and subclass feature, ancestry, community and domain card. The ones tagged Modelled do something in a fight. Text only ones are there to be nice."),
					helpItem("Adversaries", "Add a couple of adversaries. Every adversary from the SRD 2.0 is in the Library, with filters for tier and type. Edit one you added to re-skin it, or build a completely new one with + Custom."),
					helpItem("Settings", "Set the number of fights, and how aggressive each side should be with Hope, Fear and features: greedy, conservative, desperate or random."),
					helpItem("Run", "Hit Run. Every battle is a new one and resets everything, so there is no need for rests in between."),
					helpItem("Results", "See how often each side won, then go through single fights in the Combat Log. It can show only the fights the players won, or only the ones the enemies won."),
				),

				app.H3().Text("The Combat Log"),
				app.P().Text("Every action declares where its numbers come from:"),
				app.Ul().Body(
					helpItem("Roll", "each die and modifier by name, and the Difficulty or Evasion it was rolled against."),
					helpItem("Damage", "the formula and all its parts, the severity, and the Hit Points marked."),
					helpItem("State", "HP, Stress, Hope and Armor for PCs, HP and Stress for adversaries, after the action."),
					helpItem("Hope / Fear", "both pools, and why Fear changed in that row."),
				),

				app.H3().Text("Good to know"),
				app.Ul().Body(
					app.Li().Text("Everything runs in your browser. No account, no cookies, and nothing you enter is sent anywhere."),
					app.Li().Text("Your party is gone when you close or reload the page. Export JSON saves it, Import JSON brings it back."),
					app.Li().Text("Not in yet: ranges and movement, Druid Beastform, the Ranger companion, Martial Stances, dice pools, rerolls, Experiences, Help an Ally and Tag Team Rolls."),
				),

				app.H3().Text("Disclaimer"),
				app.Ul().Body(
					app.Li().Text("This is an Alpha. The engine follows the SRD 2.0 rules, but it does not have all of them. Take the numbers as a hint, not as a promise."),
					app.Li().Text("This is a fan-made tool. It is not affiliated with, sponsored by or endorsed by Darrington Press or Critical Role."),
				),
			),
		)
}

func helpItem(title, text string) app.UI {
	return app.Li().Body(app.Strong().Text(title+": "), app.Text(text))
}

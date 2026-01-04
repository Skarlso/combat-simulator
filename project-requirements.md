# Project Description

We are going to build a [Daggerheart](https://www.daggerheart.com) combat simulator. The short is this: The player provides adversaries, their stats, feats, etc, and players, sheets, stats, hope, hp, etc. And then using MonteCarlo simulations, like a 100 or so, we are going to calculate the chances of the party winning against the foes.

This is the high level overview. But, of course, this is a lot harder than this. Especially because Daggerheart has a narrative system on how to use certain resources. This will make it difficult to quantify them in some way. We could run multiple scenarios in different ways... Like greedy fear and hope usage, tokens for spotlighting characters and enemies, using feats sparingly or aggressively... etc. All settings the user could set.

## Scope

- Users providing standard adversaries
- Users providing player stats
- Users setting number of simulations
- Users setting aggressiveness of resource usage
- Users setting the way the GM uses Fear ( aggressive, no fear, always use fear until empty, random fear, clever fear ( use in certain situations when enemy starts to be desparate ( aka low on hp ) )))
- The system should be using the latest SRD (System Reference document https://www.daggerheart.com/srd/ ) -> To be decided how to obtain it... Maybe we bake in the rules and update once in a while. Convert the PDF into a CVS or something ? )
- We need to parse and understand skills. That said, all skill can be boiled down to simple terms. They either do damage somehow, or give some resource to the player. Be that health, or hope or remove a stress, or give a dice for advantage or increase damage output. The problem becomes when token usage starts. Like for a stalwart defender the skill Unstoppable. Or Token refresh for skills like Unleash Chaos. That's going to be difficult to track.
- Tracking all basic requirements and statuses like, advantage, disadvantage, fully stressed, etc. 

## Out of Scope

- Custom skills and home-brew rules
- Distances and terrain calculations ( everything is always in-range )
- Uploading characters as PDF and parsing out data
- Saving player data
- Requiring any kind of login
- Tracking of any kind
- Saving encounters ( we just display the statistics )
- Death Moves

## Restrictions

- Number of maximum simulations

## UX

- We will build this whole thing as a WASM app so people can run it locally using [go-app](https://github.com/maxence-charriere/go-app).
- No users are required. The entire thing just runs in the browser and will be statically served from github pages.

## Technology Stack

- go-app https://github.com/maxence-charriere/go-app
- Go
- Nothing else...

## Requirements

We will build this incrementally. It's going to be difficult because there are a lot of moving parts but luckily, it's actually not that difficult code. It's just a lot of code considering all the moving variables. But still it should be doable.

### Phase 1

Let's begin simple. First, somehow obtain the SRD and parse it to get the combat rules. I'll check what section describe combat.
Then, build a very simple first UX that lets the player construct multiple characters. Here, the SRD would be very helpful because it shows us how a character can be created.
But maybe, we could just plain get a character sheet as a PDF and parse it for data? The character sheets are pretty standard and come out of demiplane.

### Phase 2

Try to build a simple fighting simulation using montecarlo method. And then visualize the data.

### Phase 3

I have to still think about this.

## Non-functional requirements

- Ability to download the report or otherwise save it as CSV or PDF.

# Daggerheart SRD 2.0 Rules Digest

Distilled from `srd/DH_SRD_2_2026_08_25.pdf` (224 pages) for agents working on the simulator.
Page numbers in parentheses point back to that PDF. When this file and the PDF disagree, the PDF wins.

**Covered:** Core Mechanics (p46-54), equipment rules that affect combat (p55, p72), character baseline numbers (p4-6), Running an Adventure (p85-96), environment basics (p158-159).

**Not covered:** classes, ancestries, communities, domain cards, weapon and armor tables, loot, campaign frames, supplemental campaign mechanics.

**Adversary and environment stat blocks (p97-182) are images.** They have no extractable text, so they need OCR. Only the adversary name lists (p95-96) are text.

**Reading the PDF:** no poppler on this machine. Extract text with macOS PDFKit from Swift (`PDFDocument(url:)`, then `page(at:).string` per page).

## 1. Action rolls (p47-48)

Roll 2d12 Duality Dice (one Hope Die, one Fear Die), add the trait modifier and other modifiers, compare to a Difficulty. Meet or beat succeeds.

| Outcome | Condition | Effect | Spotlight |
|---|---|---|---|
| Success with Hope | total >= Difficulty, Hope Die higher | PC gains 1 Hope | stays with players |
| Success with Fear | total >= Difficulty, Fear Die higher | GM gains 1 Fear, success has a cost | GM takes it |
| Failure with Hope | total < Difficulty, Hope Die higher | PC gains 1 Hope, minor consequence | swings to GM |
| Failure with Fear | total < Difficulty, Fear Die higher | GM gains 1 Fear, major consequence | swings to GM |
| Critical Success | dice match | auto success, PC gains 1 Hope and clears 1 Stress, attack deals critical damage | stays with players |

- A Critical Success counts as a roll with Hope. It grants exactly 1 Hope.
- Modifiers, Experiences and extra dice must be declared before the roll unless a feature says otherwise.
- Moves that are trivial or where failure is boring succeed with no roll.

### Special rolls (p48-49)
- **Trait roll:** an action roll with a named trait, written `Agility Roll (12)`.
- **Spellcast Roll:** a trait roll using the subclass Spellcast trait. If it can damage a target it is also an attack roll. The GM can spend a Fear to end a temporary spell effect.
- **Reaction roll:** like an action roll but generates no Hope or Fear, triggers no GM move, and cannot receive Help an Ally. A crit on a reaction roll gives no Hope or Stress clear but ignores the effect entirely.
- **Group action roll:** one leader makes the action roll, the others make reaction rolls. Leader gets +1 per success and -1 per failure.
- **Tag Team roll:** once per session per player, costs 3 Hope. Both PCs roll, one result is chosen for both. With Hope: every PC involved gains a Hope. With Fear: GM gains 1 Fear per PC involved. On a successful attack both roll damage and the totals are summed as one source. Counts as one action roll for countdowns.

### Advantage and disadvantage for PCs (p49)
- Advantage: add 1d6 to the total. Disadvantage: subtract 1d6.
- Never more than one such die per roll. Advantage and disadvantage cancel one for one.
- Exception: Help an Ally. Each helper rolls their own advantage die, and only the highest die counts (including the roller's own).

## 2. Spotlight and turn flow (p46-48, p86)

- No initiative and no fixed number of actions. A player with the spotlight acts, then the spotlight moves by mechanical trigger, by fiction, or to someone who has not acted in a while.
- The GM makes a move when a player rolls with Fear, fails an action roll, does something with consequences, gives a golden opportunity, or looks to the GM.
- A GM turn begins when the spotlight passes to the GM and ends when it passes back to the players.
- On the GM turn the GM can spotlight one adversary as a GM move. Each additional adversary costs 1 Fear.
- A spotlighted adversary does one of: move within Close and make a standard attack, move within Close and use an adversary action, clear a temporary condition, sprint within Far or Very Far, or whatever the fiction demands.
- The GM can spend 1 Fear at any time to interrupt the players and take the spotlight.
- **Optional spotlight tracker:** each player gets 3 tokens and spends one per action. A player with none is skipped. When all are spent, everyone refills.

## 3. Hope and Fear (p49, p87-88)

**Hope (per PC):** starts at 2, maximum 6, carries over between sessions. Spend to:
- Help an Ally (1 Hope, roll an advantage die for them).
- Utilize an Experience (1 Hope each, add its modifier).
- Initiate a Tag Team roll (3 Hope).
- Activate a Hope Feature. Class Hope features cost 3 Hope.
- Hope gained from a roll can be spent on a Hope Feature used on that same action.

**Fear (GM pool):** starts a campaign at 1 per PC, maximum 12, carries over. Gained when a PC rolls with Fear, on rests, and from effects. Spend 1 Fear to:
- Interrupt the players and take the spotlight.
- Make an additional GM move (including spotlighting another adversary).
- Use an adversary or environment Fear Feature. This is in addition to any Fear spent to seize the spotlight.
- Add an adversary's Experience to a roll.

Fear spend per scene: Incidental 0-1, Minor 1-3, Standard 2-4, Major 4-8, Climactic 6-12. Guidance says do not hoard Fear.

## 4. Combat (p50-51)

### Attack rolls
- A PC attack is an action roll using the weapon's or spell's trait against the target's **Difficulty**.
- Any roll made against a PC uses the PC's **Evasion** as its Difficulty.
- Unarmed attacks use Strength or Finesse and deal `[Proficiency]d4`.

### Damage rolls
- Weapons: roll a number of damage dice equal to **Proficiency**. Proficiency never changes the flat modifier. Proficiency 2 with `d8+2` deals `2d8+2`.
- Spellcast-trait damage: roll dice equal to the Spellcast trait. At +0 or lower roll nothing.
- **Critical damage:** roll normally, then add the maximum possible result of the damage dice. `2d8+1` becomes `2d8+1+16`.
- **Damage types:** physical and magic. Mundane weapons and unarmed are physical, spells are magic.
- **Resistance:** halve incoming damage of that type before comparing to thresholds. Apply resistance before other reductions such as Armor Slots. Multiple resistances do not stack.
- **Immunity:** ignore damage of that type.
- Mixed physical and magic damage only benefits from resistance or immunity if the target has it for both types.
- **Direct damage:** cannot be reduced by marking Armor Slots.
- **Multi-target:** one attack roll and one damage roll, applied to each target individually.
- **Multiple simultaneous sources:** total the damage first, then compare to thresholds.
- **Incoming damage** means the total from one source before Armor Slots are marked.

### Hit Points and thresholds
- Damage >= Severe: mark 3 HP. Damage >= Major and < Severe: mark 2 HP. Damage < Major: mark 1 HP. Damage reduced to 0 or less: mark nothing.
- Optional Massive Damage: damage >= twice the Severe threshold marks 4 HP.
- PC thresholds = level + the equipped armor's base thresholds. HP slots max 12.
- Marking the last HP: an adversary is defeated, a PC makes a death move.

### Armor (p72)
- Armor Score is the number of Armor Slots. PC maximum 12.
- When taking damage you **can** mark one Armor Slot to reduce severity by one step (Severe to Major, Major to Minor, Minor to nothing). It is a choice, and one slot per instance of damage.
- Unarmored: Armor Score 0, Major threshold = level, Severe threshold = 2 x level.
- Tier 1 armor base thresholds run from 4/10 (Mage Robes, score 2) to 8/17 (Full Plate, score 4).

### Stress (p50)
- All classes start with 6 Stress slots, maximum 12.
- Marking the last Stress makes the character **Vulnerable** until they clear at least 1 Stress.
- If a character must mark 1 or more Stress but cannot, they mark **1 HP instead** (not 1 per Stress).
- A character with all Stress marked cannot use a move that requires marking Stress.
- Each adversary tracks its own Stress.

### Conditions (p52)
Standard conditions:
- **Hidden:** rolls against the creature have disadvantage. Ends when an adversary moves to where it would see you, you enter line of sight, or you attack.
- **Restrained:** cannot move, can still act from the current position.
- **Vulnerable:** all rolls targeting the creature have advantage.

Rules:
- The same condition cannot be applied twice to one target.
- **Temporary** conditions: a PC clears one with a move, normally a successful action roll. An adversary clears one by spending its spotlight, no roll.
- **Special** conditions clear only as their own text says.
- Other conditions (burning, poisoned, stunned and so on) are feature-specific, not standard.

### Range and movement (p51)
- Melee (touch), Very Close (5-10 ft), Close (10-30 ft), Far (30-100 ft), Very Far (100-300 ft), Out of Range.
- Grid option: Melee 1, Very Close 3, Close 6, Far 12, Very Far 13+ squares.
- A PC making an action roll can move within Close as part of it. Moving farther, or moving without an action roll, needs an Agility Roll.
- An adversary moves within Close for free as part of an action, or within Very Far as a separate action.
- Group effects hit targets within Very Close of one origin point unless stated.
- Attacking through cover has disadvantage. Total obstruction means no line of sight.

### Death moves (p53)
When a PC marks their last HP they choose one:
- **Blaze of Glory:** one final action that automatically critically succeeds, then the PC dies.
- **Avoid Death:** fall unconscious. Cannot act or be targeted. Wake when an ally clears 1+ of their HP or after a long rest. Roll the Hope Die: if <= level, gain a scar (permanently lose a Hope slot).
- **Risk It All:** roll the Duality Dice. Hope higher: stay up and clear HP or Stress equal to the Hope Die, split freely. Fear higher: die. Matching: stay up and clear all HP and Stress.

## 5. GM side (p86-91)

### Adversary rolls
- The GM rolls a single d20.
- **Attack:** d20 + Attack Modifier vs the target's Evasion. Meet or beat hits, then deal the stat block damage.
- **Natural 20:** auto hit, roll damage and add the maximum of the damage dice (not the flat modifier). `3d6+2` becomes `18+3d6+2`.
- **Advantage or disadvantage:** roll a second d20 and keep the higher or lower. No d6.
- The GM can spend 1 Fear before rolling to add a relevant Experience.
- Multi-target attack: one attack roll compared to each Evasion separately.
- Other adversary actions simply succeed. If failure should be possible, PCs make reaction rolls instead. A crit on an adversary reaction roll only auto-succeeds.

### Difficulty
- Attack rolls against an adversary use its Difficulty. Other action rolls against it use Difficulty plus a relevant Experience if applicable.
- Benchmarks for unspecified Difficulty: 5, 10, 15, 20, 25, 30 from trivial to legendary.
- Prefer granting advantage or disadvantage over shifting the Difficulty.

### GM moves by roll outcome
- Critical Success: player describes, gets an extra opportunity.
- Success with Hope: show how the world reacts.
- Success with Fear or Failure with Hope: minor consequence (an adversary attacks, the PC marks a Stress, new threat, raised stakes).
- Failure with Fear: major consequence.
- Soft moves on Hope, hard moves on Fear.

### Countdowns (p91)
- Start at a value, tick down by 1 when advanced, trigger at 0.
- **Standard:** advances on every player action roll. `Countdown [n]` means standard with start n.
- **Dynamic:** advances by roll outcome.

| Roll result | Progress countdown | Consequence countdown |
|---|---|---|
| Failure with Fear | 0 | 3 |
| Failure with Hope | 0 | 2 |
| Success with Fear | 1 | 1 |
| Success with Hope | 2 | 0 |
| Critical Success | 3 | 0 |

- Variants: random start, Loop (resets after triggering), Increasing, Decreasing, Linked, Long-term (advances on rests).

### Optional GM mechanics (p92)
- Fate roll: a player rolls one Duality Die and the GM interprets it.
- Falling: Very Close 1d10+3, Close 1d20+5, Far or Very Far 1d100+15 physical. Collision: 1d20+5 direct physical.
- Underwater attacks have disadvantage. Breath is a standard countdown (3).
- PC vs PC: attacks roll against Evasion. Other actions are an action roll against the target's reaction roll total.

## 6. Adversaries (p93-95)

### Stat block fields
Name, Tier, Type, Description, Motives & Tactics, Difficulty, Thresholds (Major/Severe), HP, Stress, Attack Modifier, Standard Attack (name, range, damage), Experience (optional), Features.

### Types and Battle Point cost
| Type | Role | Cost |
|---|---|---|
| Minion | easily dispatched, dangerous in numbers | 1 per group equal to party size |
| Social | conversation challenges | 1 |
| Support | buffs allies, disrupts opponents | 1 |
| Horde | many identical creatures as one unit, `X/HP` creatures per HP | 2 |
| Ranged | fragile up close, high damage at range | 2 |
| Skulk | ambushers | 2 |
| Standard | representative of their group | 2 |
| Leader | commands and summons | 3 |
| Bruiser | tough, powerful attacks | 4 |
| Solo | a challenge for a whole party | 5 |

### Encounter budget
Start with `(3 x number of PCs) + 2` Battle Points, then adjust:
- -1 easier or shorter fight
- -2 if using 2 or more Solos
- -2 if adding +1d4 (or +2) to all adversary damage rolls
- +1 if an adversary is from a lower tier
- +1 if there are no Bruisers, Hordes, Leaders or Solos
- +2 harder or longer fight

### Feature kinds
- **Action:** used when the adversary has the spotlight.
- **Reaction:** fires on its trigger whether or not the adversary has the spotlight.
- **Passive:** always on, no cost.
- **Fear Feature:** costs Fear, on top of any Fear spent to spotlight.
- A Stress cost must be paid by the adversary that owns the feature.

### Named feature patterns
- **Minion (X):** defeated by any damage. For every X damage a PC deals, defeat one additional Minion in range that the attack would succeed against.
- **Horde (X):** once half or more HP is marked, the standard attack deals X damage instead.
- **Relentless (X):** can be spotlighted up to X times per GM turn, paying Fear as usual.
- **Slow:** first spotlight places a token and does nothing. The next spotlight clears the token and acts.
- **Momentum:** on a successful attack against a PC, the GM gains a Fear.
- **Heavy Hitter:** spend a Fear for +X to a standard attack's damage.
- **Team-Up:** mark a Stress to add a standard attack and combine the damage.
- **Haymaker:** attack dealing direct physical damage.
- **Shredding Strike:** on a hit the target also marks an Armor Slot without benefit.
- **Armored Carapace:** reduce physical damage taken by X.
- **Arcane Form:** resistant to magic damage.
- **Explosion (Fear):** attack all targets within Close, knock back to Far.

### New in 2.0
- **Pool:** a named collection of tokens shared by several adversaries. Empty at scene start, cleared at scene end.
- **Evolution:** a feature that, when triggered, usually clears HP or Stress, changes the standard attack and grants new features. At most once per scene per adversary.
- **Horde `X/HP` notation.**

### Stat benchmarks
| Stat | Tier 1 | Tier 2 | Tier 3 | Tier 4 |
|---|---|---|---|---|
| Attack Modifier | +1 | +2 | +3 | +4 |
| Damage Dice | 1d6+2 to 1d12+4 | 2d6+3 to 2d12+4 | 3d8+3 to 3d12+5 | 4d8+10 to 4d12+15 |
| Difficulty | 11 | 14 | 17 | 20 |
| Thresholds (Major/Severe) | 7/12 | 10/20 | 20/32 | 25/45 |

Environment benchmarks use the same Difficulty row and slightly lower damage (Tier 1: 1d6+1 to 1d8+3).

## 7. Characters, tiers and rest (p4-6, p52-54)

- Six traits: Agility, Strength, Finesse, Instinct, Presence, Knowledge. Starting array +2, +1, +1, +0, +0, -1.
- Level 1: Proficiency 1, 2 Hope, 6 Stress slots, two Experiences at +2. Evasion and HP come from the class.
- Tiers: Tier 1 = level 1, Tier 2 = levels 2-4, Tier 3 = levels 5-7, Tier 4 = levels 8-10.
- Proficiency +1 at levels 2, 5 and 8. All damage thresholds +1 every level.
- 13 classes in 2.0 (was 9): Assassin, Bard, Brawler, Druid, Guardian, Ranger, Rogue, Seraph, Sorcerer, Warlock, Warrior, Witch, Wizard. Ten domains (adds Dread).
- **Short rest:** two downtime moves from Tend to Wounds (clear 1d4+Tier HP), Clear Stress (1d4+Tier), Repair Armor (1d4+Tier slots), Prepare (gain 1 Hope, or 2 each if done together). GM gains 1d4 Fear.
- **Long rest:** the same moves clear everything instead. GM gains 1d4 + number of PCs Fear.
- Three short rests in a row force a long rest next.

## 8. General rules (p53)

- Round up unless told otherwise. Resolve ambiguity in favor of the PCs.
- A reroll always replaces the old result.
- Effects stack, except conditions and advantage or disadvantage.
- You cannot spend Hope or mark Stress several times on one feature to repeat its effect on the same roll.
- "After the roll" features can be used after success or failure is declared, but not after consequences unfold.

## 9. Simulator status (checked against the code on 2026-10-11)

**Modeled:** duality rolls and the four outcomes, crit rules, per-PC Hope, GM Fear pool and starting value, spotlight passing, Fear for extra adversary spotlights and for seizing the spotlight, Relentless, thresholds, resistance and immunity, direct damage, Stress and Vulnerable, clearing temporary conditions, adversary d20 advantage, death moves, attack rolls using a trait (`AttackTrait`), damage dice from Proficiency, reaction rolls against effects (per-effect trait, Difficulty and half-on-success), six PC traits, armor-shredding features (mark a slot without benefit, or 1 HP if none is left), damage that costs extra Armor Slots to reduce (`ArmorCost`).

**Known gaps and simplifications:**
- PCs without custom thresholds default to `3+level` / `6+level` (`DamageThresholdFromAmount` in `internal/models/types.go`). The SRD unarmored values are `level` / `2 x level`.
- Not modeled: ranges and movement, Experiences, Help an Ally, Tag Team rolls, Massive Damage, Minion, Horde and Slow passives, Pools, Evolutions, countdowns, environments, rests, scars from Avoid Death.
- Area effects: with no ranges, an area is assumed to catch the chosen target plus random others, up to `AreaTargets` (default 2, 0 = everyone). One roll is compared to each of them separately.
- Avoid Death is treated as out of the fight, since no effect heals an ally.
- Armor is marked automatically on every damaging hit (`takeDamage` in `internal/models/combatant.go`). The SRD makes it a choice, but this is deliberate: a slot always saves exactly 1 HP and the sim has no rests, so holding slots back never helps.
- Risk It All always clears HP before Stress. Blaze of Glory is a basic attack on the lowest-HP enemy.

**Not yet checked:** whether Hidden ends when the hidden creature attacks.

**Data caveat:** `data/*.json` (adversaries by tier, feature mappings, domains, core mechanics) and `data/daggerheart_quick_reference.md` were built from the September 2025 SRD and have not been compared with 2.0.

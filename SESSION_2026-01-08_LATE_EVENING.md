# Session Notes - 2026-01-08 Late Evening

## Session Focus: Resistance System Refactor

### Summary
Completely revamped the resistance/immunity system from direct checkbox toggles to a feature-based passive effect system. This change makes the system more flexible, lore-friendly, and consistent with other passive effects in the game.

---

## Changes Made

### 1. Resistance/Immunity System Refactor ✅

**Problem**:
- Original implementation used checkboxes directly on combatant creation forms
- Felt disconnected from the feature system
- Not lore-friendly (why does a character have "Physical Resistance" checkbox?)

**Solution**:
Moved resistance/immunity to feature effect types that can be granted by passive features.

**Implementation Details**:

#### UI Changes
- **Removed**: All resistance/immunity checkboxes from both player and adversary forms
- **Added**: Two new effect types in both feature forms:
  - "Resistance (50%)"
  - "Immunity (100%)"
- When selected, damage type dropdown appears (Physical/Magical)
- Magnitude is set automatically (50 or 100) behind the scenes

**Files Modified**:
- `ui/components/combatant_list.go` - Removed checkboxes, removed state fields
- `ui/components/adversary_list.go` - Removed checkboxes, removed state fields
- `ui/components/adversary_feature_form.go` - Added effect types, magnitude logic
- `ui/components/domain_ability_form.go` - Added effect types, magnitude logic

#### Backend Logic

**New Method: `ApplyPassiveResistances()`**
```go
// In internal/models/combatant.go:370-413
func (c *Combatant) ApplyPassiveResistances() {
    // Scans all adversary features
    for _, feature := range c.AdversaryFeatures {
        if !feature.Passive { continue }
        for _, effect := range feature.Effects {
            if effect.Category == FeatureEffectDamageResistance {
                damageType := parseDamageType(effect.DamageType)
                if effect.Magnitude >= 100 {
                    c.AddImmunity(damageType)
                } else {
                    c.AddResistance(damageType)
                }
            }
        }
    }

    // Same for domain abilities...
}
```

**Called automatically** when combatants are created in both UI forms.

**Helper Function**:
```go
func parseDamageType(s string) DamageType {
    if s == "magic" { return DamageMagic }
    return DamagePhysical
}
```

#### Usage Example

To give a combatant resistance:
1. Create a passive feature (e.g., "Thick Hide")
2. Mark it as Passive ✓
3. Set Effect Type to "Resistance (50%)"
4. Select Damage Type: "Physical"
5. On combatant creation, `ApplyPassiveResistances()` grants the resistance

**Benefits**:
- ✅ More flexible - can have multiple resistance-granting features
- ✅ Lore-friendly - "Ethereal Form" grants immunity, not a checkbox
- ✅ Consistent with other passive effects (damage reduction, etc.)
- ✅ No stacking issues (existing `TakeDamage()` logic already prevents stacking)

---

### 2. Passive Domain Abilities ✅

**Problem**:
Domain abilities had no `Passive` field, unlike adversary features.

**Solution**:
Added full passive support to domain abilities for feature parity.

**Changes**:
1. Added `Passive bool` field to `DomainAbility` struct (line 94 in feature.go)
2. Added passive checkbox to UI form (similar to adversary feature form)
3. Wired up in `createAbility()` and `resetForm()` methods

**Files Modified**:
- `internal/models/feature.go:94` - Added field
- `ui/components/domain_ability_form.go:180-194` - Added checkbox UI
- `ui/components/domain_ability_form.go:511` - Set field on creation
- `ui/components/domain_ability_form.go:554` - Reset default value

---

## Testing

**Build Status**: ✅ WASM compiled successfully
```bash
GOARCH=wasm GOOS=js go build -o ./ui/web/app.wasm ./cmd/wasm
```

**Manual Testing**:
- UI forms load correctly
- Effect type dropdowns show resistance/immunity options
- Damage type selection appears when resistance/immunity selected
- Passive checkbox works for domain abilities

---

## Files Changed Summary

### Core Logic
- `internal/models/combatant.go` - Added `ApplyPassiveResistances()` method and helper

### UI Components
- `ui/components/combatant_list.go` - Removed resistance checkboxes
- `ui/components/adversary_list.go` - Removed resistance checkboxes
- `ui/components/adversary_feature_form.go` - Added resistance effect types
- `ui/components/domain_ability_form.go` - Added resistance effect types + passive checkbox

### Models
- `internal/models/feature.go` - Added Passive field to DomainAbility

---

## Documentation Updated

- ✅ `CURRENT_STATE.md` - Added Late Evening session section
- ✅ `TODO.md` - Marked resistance refactor as completed
- ✅ This session notes file

---

## Next Session Priorities

From TODO.md:

1. **Domain Ability Triggers** 🟡
   - Passive abilities ✅ (done this session)
   - Triggers ⬜ (still needed)
   - Would enable reactive player abilities

2. **Armor Threshold Mechanics** (was in internal todo)
   - Current: Damage absorbed by armor regardless of amount
   - Goal: Damage < threshold absorbed without consuming slot
   - Goal: Damage >= threshold consumes slot AND deals damage

3. **AOE Verification** 🟢
   - Low priority testing task
   - Verify multi-target damage works correctly

---

## Notes

- Resistance system is now much more elegant and flexible
- Passive abilities create a nice foundation for reactive effects
- The feature-based approach scales better than direct checkboxes
- No breaking changes to existing save files (if we had any)

**Session Duration**: ~2 hours
**Lines Changed**: ~300
**Build Status**: ✅ All green

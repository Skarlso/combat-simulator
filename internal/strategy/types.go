package strategy

// ResourceUsageStrategy defines how aggressively to use Hope/Fear tokens
type ResourceUsageStrategy string

const (
	ResourceGreedy       ResourceUsageStrategy = "greedy"       // Use resources aggressively
	ResourceConservative ResourceUsageStrategy = "conservative" // Hoard resources
	ResourceDesperate    ResourceUsageStrategy = "desperate"    // Use when low HP
	ResourceRandom       ResourceUsageStrategy = "random"       // Stochastic usage
	ResourceClever       ResourceUsageStrategy = "clever"       // Context-aware usage
)

// ConditionClearStrategy defines how to handle clearing conditions
type ConditionClearStrategy string

const (
	ClearImmediately   ConditionClearStrategy = "immediate"     // Clear harmful conditions ASAP
	ClearWhenSafe      ConditionClearStrategy = "safe"          // Clear only when not in danger
	ClearNever         ConditionClearStrategy = "never"         // Ignore conditions, focus offense
	ClearProbabilistic ConditionClearStrategy = "probabilistic" // Based on HP percentage
)

// TargetSelectionStrategy defines how to choose targets
type TargetSelectionStrategy string

const (
	TargetLowestHP      TargetSelectionStrategy = "lowest_hp"      // Focus fire weakest
	TargetHighestHP     TargetSelectionStrategy = "highest_hp"     // Attack strongest
	TargetRandom        TargetSelectionStrategy = "random"         // Random selection
	TargetHighestThreat TargetSelectionStrategy = "highest_threat" // Target most dangerous
)

// AbilityUsageStrategy defines when to use special abilities
type AbilityUsageStrategy string

const (
	AbilityAlways     AbilityUsageStrategy = "always"     // Use whenever available
	AbilitySparingly  AbilityUsageStrategy = "sparingly"  // Save for critical moments
	AbilityNever      AbilityUsageStrategy = "never"      // Basic attacks only
	AbilityContextual AbilityUsageStrategy = "contextual" // Situational usage
)

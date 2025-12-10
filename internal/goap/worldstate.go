package goap

import (
	"fmt"
	"sort"
	"strings"
)

// WorldState represents the current state of the world as a set of key-value pairs.
// Keys are strings (state variables) and values are interface{} to support any type.
type WorldState map[string]interface{}

// NewWorldState creates a new empty WorldState.
func NewWorldState() WorldState {
	return make(WorldState)
}

// Clone creates a deep copy of the WorldState.
func (ws WorldState) Clone() WorldState {
	clone := NewWorldState()
	for k, v := range ws {
		clone[k] = v
	}
	return clone
}

// Set sets a value in the WorldState.
func (ws WorldState) Set(key string, value interface{}) {
	ws[key] = value
}

// Get retrieves a value from the WorldState.
// Returns nil if the key doesn't exist.
func (ws WorldState) Get(key string) interface{} {
	return ws[key]
}

// Has checks if a key exists in the WorldState.
func (ws WorldState) Has(key string) bool {
	_, exists := ws[key]
	return exists
}

// Matches checks if this WorldState satisfies all conditions in another WorldState.
// Returns true if all key-value pairs in 'conditions' match this WorldState.
func (ws WorldState) Matches(conditions WorldState) bool {
	for key, expectedValue := range conditions {
		actualValue, exists := ws[key]
		if !exists {
			return false
		}
		if actualValue != expectedValue {
			return false
		}
	}
	return true
}

// Apply merges another WorldState into this one, overwriting existing values.
func (ws WorldState) Apply(changes WorldState) {
	for key, value := range changes {
		ws[key] = value
	}
}

// Diff returns the keys that differ between this WorldState and another.
func (ws WorldState) Diff(other WorldState) []string {
	differences := []string{}

	// Check keys in ws that differ from other
	for key, value := range ws {
		otherValue, exists := other[key]
		if !exists || otherValue != value {
			differences = append(differences, key)
		}
	}

	// Check keys in other that don't exist in ws
	for key := range other {
		if _, exists := ws[key]; !exists {
			differences = append(differences, key)
		}
	}

	return differences
}

// Distance calculates a heuristic distance to a goal state.
// This is used for A* pathfinding. Returns the number of mismatched conditions.
func (ws WorldState) Distance(goal WorldState) int {
	distance := 0
	for key, goalValue := range goal {
		currentValue, exists := ws[key]
		if !exists || currentValue != goalValue {
			distance++
		}
	}
	return distance
}

// String returns a string representation of the WorldState.
func (ws WorldState) String() string {
	if len(ws) == 0 {
		return "{}"
	}

	// Sort keys for consistent output
	keys := make([]string, 0, len(ws))
	for k := range ws {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(ws))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s: %v", k, ws[k]))
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

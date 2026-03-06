// Package skill manages the loading, registration and lifecycle of GoPaw Skills.
package skill

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gopaw/gopaw/pkg/plugin"
)

// Entry holds a loaded skill and its associated metadata.
type Entry struct {
	Manifest *plugin.SkillManifest
	// Prompt is the text fragment injected into the system prompt.
	Prompt string
	// CodeSkill is non-nil for Level-3 skills.
	CodeSkill plugin.Skill
	// Enabled indicates whether this skill is active.
	Enabled bool
}

// Registry is a thread-safe store of loaded skills.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]*Entry)}
}

// Register adds or replaces a skill entry.
func (r *Registry) Register(e *Entry) error {
	if e.Manifest == nil {
		return fmt.Errorf("skill registry: manifest is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[e.Manifest.Name] = e
	return nil
}

// Get retrieves a skill entry by name.
func (r *Registry) Get(name string) (*Entry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("skill registry: %q not found", name)
	}
	return e, nil
}

// All returns all registered skill entries.
func (r *Registry) All() []*Entry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	return out
}

// ActivePromptFragmentsForInput returns prompt fragments for skills that match the user input.
// Rules:
//   - always:true  → always included
//   - always:false → included only when the input contains at least one keyword (case-insensitive)
func (r *Registry) ActivePromptFragmentsForInput(input string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lowerInput := strings.ToLower(input)
	var combined string
	for _, e := range r.entries {
		if !e.Enabled || e.Prompt == "" {
			continue
		}
		if e.Manifest.Activation.Always {
			combined += "\n\n" + e.Prompt
			continue
		}
		for _, kw := range e.Manifest.Activation.Keywords {
			if strings.Contains(lowerInput, strings.ToLower(kw)) {
				combined += "\n\n" + e.Prompt
				break
			}
		}
	}
	return combined
}

// Clear removes all registered skills.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = make(map[string]*Entry)
}

// SetEnabled enables or disables a skill by name.
func (r *Registry) SetEnabled(name string, enabled bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[name]
	if !ok {
		return fmt.Errorf("skill registry: %q not found", name)
	}
	e.Enabled = enabled
	return nil
}

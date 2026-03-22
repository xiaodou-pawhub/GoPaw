// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

// SkillLevel categorises the implementation complexity of a skill.
type SkillLevel int

const (
	// SkillLevelPrompt is a pure prompt-injection skill (manifest.yaml + prompt.md, no code).
	SkillLevelPrompt SkillLevel = 1
	// SkillLevelCode is a script/code skill executed via subprocess (manifest.yaml + script file).
	// Note: Level 3 (compiled Go) from older versions is treated as Level 2 for simplicity.
	SkillLevelCode SkillLevel = 2
)

// SkillManifest describes the metadata parsed from a skill's manifest.yaml.
type SkillManifest struct {
	Name          string            `yaml:"name"`
	Version       string            `yaml:"version"`
	DisplayName   string            `yaml:"display_name"`
	Description   string            `yaml:"description"`
	Author        string            `yaml:"author"`
	Level         SkillLevel        `yaml:"level"`
	Activation    SkillActivation   `yaml:"activation"`
	RequiresTools []string          `yaml:"requires_tools"`
	ConfigSchema  map[string]interface{} `yaml:"config_schema"`
}

// SkillActivation is kept for backwards compatibility with existing manifest.yaml files,
// but is no longer used for selection logic — the AI decides when to use skills.
// Deprecated: activation keywords and always-flag are ignored.
type SkillActivation struct {
	Always   bool     `yaml:"always"`
	Keywords []string `yaml:"keywords"`
}

// Skill is the interface that Level-3 (code) skills must implement.
// Level-1 and Level-2 skills are handled entirely through their manifest and prompt files.
type Skill interface {
	// Manifest returns the parsed skill metadata.
	Manifest() *SkillManifest

	// PromptFragment returns the text fragment injected into the system prompt.
	PromptFragment() string

	// Init is called once when the skill is activated, with its user configuration.
	Init(cfg map[string]interface{}) error

	// Tools returns any additional Tool implementations this skill registers.
	// Return nil for skills that don't need custom tools.
	Tools() []Tool
}

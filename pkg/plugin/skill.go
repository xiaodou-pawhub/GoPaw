// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

// SkillLevel categorises the implementation complexity of a skill.
type SkillLevel int

const (
	// SkillLevelPrompt is a pure prompt-injection skill (no code required).
	SkillLevelPrompt SkillLevel = 1
	// SkillLevelConfig is a YAML-workflow skill (low-code).
	SkillLevelConfig SkillLevel = 2
	// SkillLevelCode is a full Go-code skill (complete control).
	SkillLevelCode SkillLevel = 3
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

// SkillActivation controls when and how the skill is injected into the system prompt.
type SkillActivation struct {
	// Always means the skill prompt is always included regardless of user input.
	Always   bool     `yaml:"always"`
	// Keywords are hint words shown to users to trigger the skill.
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

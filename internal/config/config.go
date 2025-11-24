package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	LLM          LLMConfig    `mapstructure:"llm"`
	Projects     []Project    `mapstructure:"projects"`
	Beacons      []Beacon     `mapstructure:"beacons"`
	FocusGroups  []FocusGroup `mapstructure:"focus_groups"`
	DefaultQuota int          `mapstructure:"default_quota"` // Default tasks per project in focus list
}

type FocusGroup struct {
	Name     string   `mapstructure:"name"`
	Patterns []string `mapstructure:"patterns"` // Glob patterns like "er.*", "personal.*"
	Quota    int      `mapstructure:"quota"`
}

type LLMConfig struct {
	Provider  string `mapstructure:"provider"` // anthropic, openai, ollama
	Model     string `mapstructure:"model"`
	APIKeyEnv string `mapstructure:"api_key_env"`
	BaseURL   string `mapstructure:"base_url"` // for ollama or custom endpoints
}

type Project struct {
	Name     string   `mapstructure:"name"`
	Keywords []string `mapstructure:"keywords"`
	Quota    int      `mapstructure:"quota"` // Tasks per focus list, default 2
}

type Beacon struct {
	Name        string      `mapstructure:"name"`
	Tag         string      `mapstructure:"tag"`
	Description string      `mapstructure:"description"`
	Directions  []Direction `mapstructure:"directions"`
}

type Direction struct {
	Name        string `mapstructure:"name"`
	Tag         string `mapstructure:"tag"`
	Description string `mapstructure:"description"`
}

func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config dir: %w", err)
	}

	configPath := filepath.Join(configDir, "tg")

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")

	// Set defaults
	viper.SetDefault("llm.provider", "anthropic")
	viper.SetDefault("llm.model", "claude-sonnet-4-5-20250929")
	viper.SetDefault("llm.api_key_env", "ANTHROPIC_API_KEY")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found, use defaults + embedded beacons
			cfg := &Config{}
			if err := viper.Unmarshal(cfg); err != nil {
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}
			cfg.Beacons = DefaultBeacons()
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply defaults for empty fields (viper defaults only work when config file is missing)
	if cfg.LLM.Provider == "" {
		cfg.LLM.Provider = "anthropic"
	}
	if cfg.LLM.Model == "" {
		cfg.LLM.Model = "claude-sonnet-4-5-20250929"
	}
	if cfg.LLM.APIKeyEnv == "" {
		cfg.LLM.APIKeyEnv = "ANTHROPIC_API_KEY"
	}

	// Use default beacons if none configured
	if len(cfg.Beacons) == 0 {
		cfg.Beacons = DefaultBeacons()
	}

	// Default quota for focus list
	if cfg.DefaultQuota == 0 {
		cfg.DefaultQuota = 2
	}

	return &cfg, nil
}

func (c *Config) GetAPIKey() string {
	if c.LLM.APIKeyEnv == "" {
		return ""
	}
	return os.Getenv(c.LLM.APIKeyEnv)
}

// GetProjectQuota returns the quota for a specific project, or default if not set
func (c *Config) GetProjectQuota(projectName string) int {
	for _, p := range c.Projects {
		if p.Name == projectName && p.Quota > 0 {
			return p.Quota
		}
	}
	return c.DefaultQuota
}

// GetFocusGroup returns the focus group name for a project, or empty string if no match
func (c *Config) GetFocusGroup(projectName string) string {
	for _, fg := range c.FocusGroups {
		// First check if any exclusion pattern matches
		excluded := false
		for _, pattern := range fg.Patterns {
			if len(pattern) > 0 && pattern[0] == '!' {
				excludePattern := pattern[1:]
				if matchPattern(excludePattern, projectName) {
					excluded = true
					break
				}
			}
		}

		if excluded {
			continue // Skip this group
		}

		// Check if any positive pattern matches
		for _, pattern := range fg.Patterns {
			if len(pattern) > 0 && pattern[0] != '!' {
				if matchPattern(pattern, projectName) {
					return fg.Name
				}
			}
		}
	}
	return ""
}

// GetFocusGroupQuota returns the quota for a focus group
func (c *Config) GetFocusGroupQuota(groupName string) int {
	for _, fg := range c.FocusGroups {
		if fg.Name == groupName && fg.Quota > 0 {
			return fg.Quota
		}
	}
	return c.DefaultQuota
}

// matchPattern checks if project matches a glob pattern (supports * and ?)
func matchPattern(pattern, project string) bool {
	if pattern == "*" {
		return true
	}
	// Simple glob matching: "er.*" matches "er.release", "er.sre", etc.
	// "personal" matches exactly "personal"
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(project) >= len(prefix) && project[:len(prefix)] == prefix
	}
	return pattern == project
}

// DefaultBeacons returns the embedded Beacons system
func DefaultBeacons() []Beacon {
	return []Beacon{
		{
			Name:        "Be Organized",
			Tag:         "b.organized",
			Description: "Focus on personal organization and productivity",
			Directions: []Direction{
				{Name: "Develop organization habits", Tag: "d.org.habits", Description: "Building habits for better organization"},
				{Name: "Improve tooling", Tag: "d.org.tooling", Description: "Better tools and systems for organization"},
				{Name: "Improve writing skills", Tag: "d.writing", Description: "Better written communication"},
				{Name: "Time management", Tag: "d.time.mgmt", Description: "Better time management and prioritization"},
				{Name: "Project planning", Tag: "d.project.plan", Description: "Better planning skills"},
				{Name: "Keep order", Tag: "d.order", Description: "Maintaining order in physical and digital spaces"},
			},
		},
		{
			Name:        "Be Significant in a Field",
			Tag:         "b.significant.field",
			Description: "Become recognized expert in a domain",
			Directions: []Direction{
				{Name: "Improve writing skills", Tag: "d.writing", Description: "Better written communication for sharing knowledge"},
				{Name: "Improve coverage", Tag: "d.coverage", Description: "Broader visibility and reach"},
				{Name: "Improve communication", Tag: "d.comm", Description: "Better verbal and presentation skills"},
			},
		},
		{
			Name:        "Be a Great Software Developer",
			Tag:         "b.great.dev",
			Description: "Excel in software development",
			Directions: []Direction{
				{Name: "Algorithm skills", Tag: "d.algo", Description: "Better algorithmic thinking"},
				{Name: "Build workspace", Tag: "d.workspace", Description: "Better development environment"},
				{Name: "Programming languages", Tag: "d.prog.lang", Description: "Deeper language knowledge"},
				{Name: "Software design", Tag: "d.sw.design", Description: "Better architecture and design patterns"},
				{Name: "Dev tooling", Tag: "d.dev.tooling", Description: "Better tooling knowledge (IDE, debugging, etc.)"},
				{Name: "Test writing", Tag: "d.test.write", Description: "Better testing skills"},
				{Name: "Advanced tooling", Tag: "d.tooling.adv", Description: "Databases, queues, messaging systems"},
				{Name: "OS and networks", Tag: "d.os.network", Description: "System-level knowledge"},
			},
		},
		{
			Name:        "Be a Great DevOps",
			Tag:         "b.great.devops",
			Description: "Excel in DevOps and infrastructure",
			Directions: []Direction{
				{Name: "System design", Tag: "d.sys.design", Description: "Better infrastructure architecture"},
				{Name: "Software design", Tag: "d.sw.design", Description: "Better application design for ops"},
				{Name: "Advanced tooling", Tag: "d.tooling.adv", Description: "Databases, queues, messaging"},
				{Name: "OS and networks", Tag: "d.os.network", Description: "Deep system knowledge"},
				{Name: "Cloud knowledge", Tag: "d.cloud", Description: "Cloud platforms and services"},
			},
		},
		{
			Name:        "Be Great in Hardware Development",
			Tag:         "b.great.hardware",
			Description: "Excel in hardware and electronics",
			Directions: []Direction{
				{Name: "HW software tooling", Tag: "d.hw.sw.tooling", Description: "Firmware, embedded software tools"},
				{Name: "HW tooling", Tag: "d.hw.tooling", Description: "Physical tools for hardware work"},
				{Name: "Build workspace", Tag: "d.workspace", Description: "Hardware development workspace"},
				{Name: "Software design", Tag: "d.sw.design", Description: "Embedded software design"},
				{Name: "HW development", Tag: "d.hw.dev", Description: "General hardware development skills"},
				{Name: "Circuit design", Tag: "d.circuits", Description: "Electronic circuit design"},
				{Name: "Soldering", Tag: "d.soldering", Description: "Soldering and assembly skills"},
			},
		},
		{
			Name:        "Be Great in Relationships",
			Tag:         "b.great.rel",
			Description: "Excel in personal and professional relationships",
			Directions: []Direction{
				{Name: "Help friends", Tag: "d.help.friends", Description: "Supporting friends"},
				{Name: "Kids", Tag: "d.kids", Description: "Being a great parent"},
				{Name: "Help family", Tag: "d.help.family", Description: "Supporting family"},
				{Name: "Communication", Tag: "d.comm", Description: "Better interpersonal communication"},
				{Name: "Psychological help", Tag: "d.psych.help", Description: "Ability to help others emotionally"},
			},
		},
		{
			Name:        "Be Prepared to Draft",
			Tag:         "b.prep.draft",
			Description: "Military readiness and preparedness",
			Directions: []Direction{
				{Name: "Physical endurance", Tag: "d.endurance", Description: "Physical fitness and stamina"},
				{Name: "Martial arts", Tag: "d.martial.arts", Description: "Combat and self-defense skills"},
				{Name: "Orientation", Tag: "d.orientation", Description: "Navigation and orientation skills"},
				{Name: "Military knowledge", Tag: "d.mil.gen.knowledge", Description: "General military knowledge"},
				{Name: "Equipment prep", Tag: "d.mil.equip.prep", Description: "Equipment and gear preparation"},
				{Name: "FPV skills", Tag: "d.fpv", Description: "Drone piloting skills"},
				{Name: "Soldering", Tag: "d.soldering", Description: "Field repair skills"},
			},
		},
		{
			Name:        "Be Healthy",
			Tag:         "b.healthy",
			Description: "Physical and mental health",
			Directions: []Direction{
				{Name: "Physical endurance", Tag: "d.endurance", Description: "Physical fitness"},
				{Name: "Martial arts", Tag: "d.martial.arts", Description: "Physical activity through martial arts"},
				{Name: "Healthy habits", Tag: "d.healthy.habits", Description: "Diet, sleep, routines"},
			},
		},
		{
			Name:        "Help to Win the War",
			Tag:         "b.war.help",
			Description: "Contributing to Ukraine's defense",
			Directions: []Direction{
				{Name: "Help soldiers", Tag: "d.war.help", Description: "Direct support to military"},
				{Name: "War tools", Tag: "d.war.tools", Description: "Creating tools for battlefield"},
				{Name: "Knowledge help", Tag: "d.war.knowledge.help", Description: "Sharing expertise"},
				{Name: "Psychological help", Tag: "d.psych.help", Description: "Emotional support"},
			},
		},
	}
}

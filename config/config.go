// config/config.go
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type MonitoringConfig struct {
	// General settings
	PollInterval       time.Duration `yaml:"pollInterval"`
	EnableEventLogging bool          `yaml:"enableEventLogging"`
	LogPath            string        `yaml:"logPath"`

	// Process monitoring thresholds
	MemoryChangeThreshold float64 `yaml:"memoryChangeThreshold"`
	CPUThreshold          float64 `yaml:"cpuThreshold"`
	ThreadChangeThreshold int32   `yaml:"threadChangeThreshold"`
	HandleChangeThreshold int32   `yaml:"handleChangeThreshold"`

	// Feature flags
	MonitorCommandLine bool `yaml:"monitorCommandLine"`
	MonitorMemory      bool `yaml:"monitorMemory"`
	MonitorThreads     bool `yaml:"monitorThreads"`
	MonitorHandles     bool `yaml:"monitorHandles"`
	MonitorWorkingDir  bool `yaml:"monitorWorkingDir"`
	MonitorConnections bool `yaml:"monitorConnections"`

	// Process filtering
	ExcludedProcesses []string `yaml:"excludedProcesses"`
	IncludedProcesses []string `yaml:"includedProcesses"`
	ExcludedUsers     []string `yaml:"excludedUsers"`

	// Thresholds
	ProcessPriorityThreshold int32    `yaml:"processPriorityThreshold"`
	MaxProcessesToMonitor    int32    `yaml:"maxProcessesToMonitor"`
	ProcessAgeThreshold      duration `yaml:"processAgeThreshold"`
}

type duration struct {
	time.Duration
}

func LoadConfig(path string) (*MonitoringConfig, error) {
	configFile, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func DefaultConfig() *MonitoringConfig {
	return &MonitoringConfig{
		PollInterval:       time.Second,
		EnableEventLogging: true,
		LogPath:            "process_monitor.log",

		MemoryChangeThreshold: 0.1,  // 10% change
		CPUThreshold:          0.75, // 75% CPU usage
		ThreadChangeThreshold: 2,    // Thread count change >= 2
		HandleChangeThreshold: 10,   // Handle count change >= 10

		MonitorCommandLine: true,
		MonitorMemory:      true,
		MonitorThreads:     true,
		MonitorHandles:     true,
		MonitorWorkingDir:  true,
		MonitorConnections: false,

		ExcludedProcesses: []string{"svchost.exe", "RuntimeBroker.exe"},
		IncludedProcesses: []string{},
		ExcludedUsers:     []string{"root"},

		ProcessPriorityThreshold: 32768, // Normal priority
		MaxProcessesToMonitor:    1000,
		ProcessAgeThreshold:      duration{1 * time.Minute},
	}
}
func (c *MonitoringConfig) Validate() error {
	if c.PollInterval < time.Millisecond*100 {
		return fmt.Errorf("poll interval too short, minimum 100ms")
	}

	if c.MemoryChangeThreshold <= 0 || c.MemoryChangeThreshold > 1 {
		return fmt.Errorf("memory change threshold must be between 0 and 1")
	}

	if c.CPUThreshold <= 0 || c.CPUThreshold > 1 {
		return fmt.Errorf("CPU threshold must be between 0 and 1")
	}

	if c.ThreadChangeThreshold < 0 {
		return fmt.Errorf("thread change threshold must be non-negative")
	}

	if c.HandleChangeThreshold < 0 {
		return fmt.Errorf("handle change threshold must be non-negative")
	}

	return nil
}

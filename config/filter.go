package config

import (
	"ArgusSentinel/types"
	"path/filepath"
	"strings"
)

type ProcessFilter struct {
	config *MonitoringConfig
}

func NewProcessFilter(config *MonitoringConfig) *ProcessFilter {
	return &ProcessFilter{
		config: config,
	}
}

func (pf *ProcessFilter) ShouldMonitorProcess(info types.ProcessInfo) bool {
	// Check process name exclusions
	processName := strings.ToLower(info.Name)
	for _, excluded := range pf.config.ExcludedProcesses {
		if match, _ := filepath.Match(strings.ToLower(excluded), processName); match {
			return false
		}
	}

	// Check process inclusions if specified
	if len(pf.config.IncludedProcesses) > 0 {
		included := false
		for _, includedProc := range pf.config.IncludedProcesses {
			if match, _ := filepath.Match(strings.ToLower(includedProc), processName); match {
				included = true
				break
			}
		}

		if !included {
			return false
		}
	}

	// Check user exclusions
	username := strings.ToLower(info.Username)
	for _, excludedUser := range pf.config.ExcludedUsers {
		if strings.ToLower(excludedUser) == username {
			return false
		}
	}

	// Check process age
	// if time.Since(time.Unix(0, info.CreateTime)).Hours() > pf.config.ProcessAgeThreshold.Hours() {
	// 	return false
	// }

	// // Check process priority
	// if info.Priority < pf.config.ProcessPriorityThreshold {
	// 	return false
	// }

	return true
}

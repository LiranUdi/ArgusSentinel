package collector

import (
	"context"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"

	cfg "ArgusSentinel/config"
	"ArgusSentinel/types"
	"ArgusSentinel/utils"
)

/*
* ProcessCollector struct
* Used to hold process information to be used in analysis
 */
type ProcessCollector struct {
	// Add fields for Windows API handles
	// and process tracking
	currentProcesses map[int32]types.ProcessInfo
	events           chan<- types.ProcessEvent
	config           *cfg.MonitoringConfig
	filter           *cfg.ProcessFilter
	mutex            sync.RWMutex
}

/*
* Create a new ProcessCollector
 */
func NewProcessCollector(events chan<- types.ProcessEvent, config *cfg.MonitoringConfig) *ProcessCollector {
	// Init windows API handles
	// Set up process tracking
	return &ProcessCollector{
		currentProcesses: make(map[int32]types.ProcessInfo),
		events:           events,
		config:           config,
		filter:           cfg.NewProcessFilter(config), // Initialize the filter
		mutex:            sync.RWMutex{},
	}
}

/*
* ProcessCollector getProcessInfo method
* Extract information from a *process.Process and return it as types.ProcessInfo
 */
func (pc *ProcessCollector) getProcessInfo(p *process.Process) (types.ProcessInfo, error) {
	info := types.ProcessInfo{
		PID: p.Pid,
	}

	utils.SetField(&info.Name, p.Name)
	utils.SetField(&info.CreateTime, p.CreateTime)
	utils.SetField(&info.ParentPID, p.Ppid)
	utils.SetField(&info.Executable, p.Exe)
	utils.SetField(&info.CommandLine, p.Cmdline)
	utils.SetField(&info.Username, p.Username)
	utils.SetField(&info.CPUPercent, p.CPUPercent)
	utils.SetField(&info.NumThreads, p.NumThreads)
	utils.SetField(&info.Cwd, p.Cwd)

	// MemoryInfo requires custom handling
	if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
		info.MemoryUsage = memInfo.RSS // Resident Set Size
	}

	return info, nil
}

/*
* ProcessCollector getRunningProcesses method
* Returnins all running processes
 */
func (pc *ProcessCollector) getRunningProcesses() (map[int32]types.ProcessInfo, error) {
	processes := make(map[int32]types.ProcessInfo)

	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %v", err)
	}

	for _, p := range procs {
		info, err := pc.getProcessInfo(p)
		if err != nil {
			continue
		}

		// Apply filter before adding to the map
		if pc.filter.ShouldMonitorProcess(info) {
			processes[p.Pid] = info
		}
	}

	return processes, nil
}

/*
* ProcessCollector Monitor method
* Monitors processes, checks for newly created, terminated and modified processes
 */
func (pc *ProcessCollector) Monitor(ctx context.Context) error {
	ticker := time.NewTicker(pc.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			newSnapshot, err := pc.getRunningProcesses()
			if err != nil {
				continue
			}

			pc.mutex.Lock()

			// Check for new processes
			for pid, newInfo := range newSnapshot {
				if oldInfo, exists := pc.currentProcesses[pid]; exists {
					if pc.filter.ShouldMonitorProcess(newInfo) {
						// Process exists in both snapshots - check for modifications
						if mods := pc.detectModifications(oldInfo, newInfo); len(mods) > 0 {
							// Send modification events
							for _, mod := range mods {
								pc.events <- types.ProcessEvent{
									Type:        types.ProcessModified,
									Timestamp:   mod.Timestamp,
									Process:     newInfo,
									ModType:     mod.ModType,
									Description: mod.Description,
								}
							}
						}
					}

				} else {
					if pc.filter.ShouldMonitorProcess(newInfo) {
						// New process created
						pc.events <- types.ProcessEvent{
							Type:      types.ProcessCreated,
							Timestamp: time.Now(),
							Process:   newInfo,
							// Add other event details
						}
					}
				}
			}

			// Check for terminated processes
			for pid, info := range pc.currentProcesses {
				if _, exists := newSnapshot[pid]; !exists {
					if pc.filter.ShouldMonitorProcess(info) {
						pc.events <- types.ProcessEvent{
							Type:      types.ProcessTerminated,
							Timestamp: time.Now(),
							Process:   info,
							// Add other event details
						}
					}

				}
			}

			pc.currentProcesses = newSnapshot
			pc.mutex.Unlock()
		}
	}
}

func (pc *ProcessCollector) shouldMonitorProcess(info types.ProcessInfo) bool {
	processName := strings.ToLower(info.Name)

	// Debug logging
	for _, excluded := range pc.config.ExcludedProcesses {
		matched, err := filepath.Match(strings.ToLower(excluded), processName)
		if err == nil && matched {
			log.Printf("Excluding process: %s (matched pattern: %s)", info.Name, excluded)
			return false
		}
	}

	// Check excluded processes
	for _, excluded := range pc.config.ExcludedProcesses {
		if strings.EqualFold(info.Name, excluded) {
			return false
		}
	}

	// Check included processes
	if len(pc.config.IncludedProcesses) > 0 {
		included := false
		for _, includedProc := range pc.config.IncludedProcesses {
			if strings.EqualFold(info.Name, includedProc) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	// Check excluded users
	for _, excludedUser := range pc.config.ExcludedUsers {
		if strings.EqualFold(info.Username, excludedUser) {
			return false
		}
	}

	return true
}

/*
* ProcessCollector detectModification method
* Takes an old and new snapshot of a process and checks how the process was modified
* TODO: Add aditional methods for checking process modification, replace arbitrary checks with more robust checks
 */
func (pc *ProcessCollector) detectModifications(old, new types.ProcessInfo) []types.ProcessModification {
	if !pc.shouldMonitorProcess(new) {
		return nil
	}

	var modifications []types.ProcessModification
	timestamp := time.Now()

	// Check command line changes
	if pc.config.MonitorCommandLine && old.CommandLine != new.CommandLine {
		modifications = append(modifications, types.ProcessModification{
			Timestamp:   timestamp,
			ProcessID:   new.PID,
			ModType:     types.CommandLineChange,
			OldValue:    old.CommandLine,
			NewValue:    new.CommandLine,
			Description: "Command line modified",
		})
	}

	// Check thread count changes
	if pc.config.MonitorThreads {
		threadDiff := int32(math.Abs(float64(new.ThreadCount - old.ThreadCount)))
		if threadDiff >= pc.config.ThreadChangeThreshold {
			modifications = append(modifications, types.ProcessModification{
				Timestamp:   timestamp,
				ProcessID:   new.PID,
				ModType:     types.ThreadCountChange,
				OldValue:    old.ThreadCount,
				NewValue:    new.ThreadCount,
				Description: fmt.Sprintf("Thread count changed by %d", threadDiff),
			})
		}
	}

	if pc.config.MonitorHandles {
		handleDiff := int32(math.Abs(float64(new.HandleCount - old.HandleCount)))
		if handleDiff >= pc.config.HandleChangeThreshold {
			modifications = append(modifications, types.ProcessModification{
				Timestamp:   timestamp,
				ProcessID:   new.PID,
				ModType:     types.HandleCountChange,
				OldValue:    old.HandleCount,
				NewValue:    new.HandleCount,
				Description: fmt.Sprintf("Handle count changed by %d", handleDiff),
			})
		}
	}

	if pc.config.MonitorMemory {
		memDiff := float64(new.MemoryUsage) / float64(old.MemoryUsage)
		if memDiff < (1-pc.config.MemoryChangeThreshold) ||
			memDiff > (1+pc.config.MemoryChangeThreshold) {
			modifications = append(modifications, types.ProcessModification{
				Timestamp:   timestamp,
				ProcessID:   new.PID,
				ModType:     types.MemoryModification,
				OldValue:    old.MemoryUsage,
				NewValue:    new.MemoryUsage,
				Description: "Significant memory usage change",
			})
		}
	}

	if pc.config.MonitorWorkingDir && old.Cwd != new.Cwd {
		modifications = append(modifications, types.ProcessModification{
			Timestamp:   timestamp,
			ProcessID:   new.PID,
			ModType:     types.WorkingDirectoryChange,
			OldValue:    old.Cwd,
			NewValue:    new.Cwd,
			Description: "Working directory changed",
		})
	}

	return modifications
}

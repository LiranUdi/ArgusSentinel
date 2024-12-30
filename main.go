package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"ArgusSentinel/collector"
	"ArgusSentinel/config"
	"ArgusSentinel/types"

	"github.com/shirou/gopsutil/v3/process"
)

/*
* ProcessMonitor struct
* Main controller
 */
type ProcessMonitor struct {
	config *config.MonitoringConfig
	ctx    context.Context
	cancel context.CancelFunc
	events chan types.ProcessEvent
	wg     sync.WaitGroup
}

/*
* Create a new ProcessMonitor instance
 */
func NewProcessMonitor(config *config.MonitoringConfig) *ProcessMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessMonitor{
		config: config,
		ctx:    ctx,
		cancel: cancel,
		events: make(chan types.ProcessEvent, 1000),
	}
}

/*
* ProcessMonitor Start method
* Start monitoring processes
 */
func (pm *ProcessMonitor) Start() error {
	// Start monitoring goroutines
	collector := collector.NewProcessCollector(pm.events, pm.config)

	pm.wg.Add(1)
	go func() {
		defer pm.wg.Done()
		if err := collector.Monitor(pm.ctx); err != nil {
			log.Printf("Process collector error: %v", err)
		}
	}()

	pm.wg.Add(1)
	go pm.processEventLoop()

	return nil
}

/*
* ProcessMonitor Stop method
* Stop monitoring processes
 */
func (pm *ProcessMonitor) Stop() {
	pm.cancel()
	pm.wg.Wait()
}

/*
* ProcessMonitor processEventLoop method
* Poll events and output the processes found (Created/Terminated/Modified)
* TODO: Improve output formatting
 */
func (pm *ProcessMonitor) processEventLoop() {
	defer pm.wg.Done()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case event := <-pm.events:
			switch event.Type {
			case types.ProcessCreated:
				message := fmt.Sprintf("[+] New process: PID=%d Name=%s User=%s",
					event.Process.PID,
					event.Process.Name,
					event.Process.Username)
				log.Println(message)
			case types.ProcessTerminated:
				message := fmt.Sprintf("[-] Process terminated: PID=%d Name=%s",
					event.Process.PID,
					event.Process.Name)
				log.Println(message)
			case types.ProcessModified:
				message := fmt.Sprintf("[***] Process modified: PID=%d Name:%s - %s\n",
					event.Process.PID,
					event.Process.Name,
					event.Description)
				log.Println(message)
			}
		}
	}
}

/*
* ProcessMonitor collectMetrics method
* TODO: Implement metrics collection for processes and events
 */
func (pm *ProcessMonitor) collectMetrics() {
	defer pm.wg.Done()
	// Implement collection
}

// don't know
func (pm *ProcessMonitor) watchProcessEvents() error {
	// Start simple polling approach
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return nil
		case <-ticker.C:
			_, err := process.Processes()
			if err != nil {
				return err
			}
		}
	}
}

func main() {
	// config, err := config.LoadConfig("config.yaml")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	config := config.DefaultConfig()
	log.Printf("%v\n", config)
	monitor := NewProcessMonitor(config)
	if err := monitor.Start(); err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c

	monitor.Stop()
}

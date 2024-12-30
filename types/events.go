// types/events.go
package types

import "time"

type EventType int

/*
* EventType constants
* Represents all process event types
* TODO: Look into additional event types. Sufficient for the current scope
 */
const (
	ProcessCreated EventType = iota
	ProcessTerminated
	ProcessModified
)

/*
* ProcessInfo struct
* Represents most of the information needed to analyze and display a process
 */
type ProcessInfo struct {
	PID           int32
	Name          string
	CreateTime    int64
	ParentPID     int32
	Executable    string
	CommandLine   string
	Username      string
	CPUPercent    float64
	MemoryUsage   uint64
	NumThreads    int32
	Cwd           string
	ThreadCount   int32
	HandleCount   int32
	MemoryRegions []MemoryRegion
	Privileges    []string
	NetworkConns  []NetworkConnection
}

/*
* ProcessEvent struct
* Represents events and modifications that occur in processes
* i.e, new process, terminated, or modified (see ModificationType)
 */
type ProcessEvent struct {
	Type        EventType
	Timestamp   time.Time
	Process     ProcessInfo
	ModType     ModificationType // Only used when Type is ProcessModified
	Description string
}

/*
* ProcessModification struct
* Represents modifications that occur in processes such as new threads being created, behavioral changes, memory modifications and more.
* Used for analysis and comparing a previous to a current state
 */
type ProcessModification struct {
	Timestamp   time.Time
	ProcessID   int32
	ModType     ModificationType
	OldValue    interface{}
	NewValue    interface{}
	Description string
}

/*
* ModificationType int
 */
type ModificationType int

/*
* ModificationType constants
* Represents all the types of modifications that can occur in a program
* TODO: Review modification types, assess what to remove or add
 */
const (
	MemoryModification ModificationType = iota
	ThreadCreation
	HandleTableChange
	PrivilegeChange
	BehaviorChange
	CommandLineChange
	ThreadCountChange
	HandleCountChange
	WorkingDirectoryChange
)

/*
* MemoryRegion struct
* Currently not used
* Represents the memory region for a process, will be used for detecting malicious modifications to a process' memory
* TODO: Implement memory analysis
 */
type MemoryRegion struct {
	BaseAddress uintptr
	Size        uint64
	Protection  uint32 // Memory protection flags
	State       uint32 // Committed, reserved, free
	Type        uint32 // Private, mapped, image
	Usage       string // Description of usage (heap, stack, etc)
}

/*
* NetworkConnection struct
* Currently not used
* Represents information related to a process' network connection
* TODO: Implement process network communication analysis
 */
type NetworkConnection struct {
	LocalIP    string
	LocalPort  uint16
	RemoteIP   string
	RemotePort uint16
	Status     string // ESTABLISHED, LISTENING, etc
	Protocol   string // TCP, UDP
	ProcessID  int32
	CreateTime time.Time
}

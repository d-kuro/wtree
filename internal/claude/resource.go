package claude

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ResourceManager manages resource allocation for Claude tasks
type ResourceManager struct {
	maxClaude      int
	maxDevelopment int
	activeDev      int
	devSlots       chan struct{}
	mu             sync.RWMutex
}

// Slot represents a resource slot allocation
type Slot struct {
	ID         string
	TaskType   TaskType
	AcquiredAt time.Time
	manager    *ResourceManager
}

// NewResourceManager creates a new resource manager
func NewResourceManager(maxClaude, maxDevelopment int) *ResourceManager {
	if maxDevelopment <= 0 {
		maxDevelopment = maxClaude
	}

	return &ResourceManager{
		maxClaude:      maxClaude,
		maxDevelopment: maxDevelopment,
		devSlots:       make(chan struct{}, maxDevelopment),
	}
}

// AcquireSlot attempts to acquire a resource slot for the given task type
func (r *ResourceManager) AcquireSlot(ctx context.Context, taskType TaskType, taskID string) (*Slot, error) {
	slot := &Slot{
		ID:         taskID,
		TaskType:   taskType,
		AcquiredAt: time.Now(),
		manager:    r,
	}

	switch taskType {
	case TaskTypeDevelopment:
		select {
		case r.devSlots <- struct{}{}:
			r.mu.Lock()
			r.activeDev++
			r.mu.Unlock()
			return slot, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

// TryAcquireSlot attempts to acquire a slot without blocking
func (r *ResourceManager) TryAcquireSlot(taskType TaskType, taskID string) (*Slot, error) {
	slot := &Slot{
		ID:         taskID,
		TaskType:   taskType,
		AcquiredAt: time.Now(),
		manager:    r,
	}

	switch taskType {
	case TaskTypeDevelopment:
		select {
		case r.devSlots <- struct{}{}:
			r.mu.Lock()
			r.activeDev++
			r.mu.Unlock()
			return slot, nil
		default:
			return nil, fmt.Errorf("no development slots available")
		}
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}

// Release releases a resource slot
func (s *Slot) Release() {
	switch s.TaskType {
	case TaskTypeDevelopment:
		<-s.manager.devSlots
		s.manager.mu.Lock()
		s.manager.activeDev--
		s.manager.mu.Unlock()
	}
}

// GetStats returns current resource usage statistics
func (r *ResourceManager) GetStats() ResourceStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return ResourceStats{
		MaxClaude:              r.maxClaude,
		MaxDevelopment:         r.maxDevelopment,
		ActiveDevelopment:      r.activeDev,
		AvailableDevelopment:   r.maxDevelopment - r.activeDev,
		TotalActive:            r.activeDev,
		DevelopmentUtilization: float64(r.activeDev) / float64(r.maxDevelopment) * 100,
	}
}

// CanAcquire checks if a slot can be acquired for the given task type
func (r *ResourceManager) CanAcquire(taskType TaskType) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch taskType {
	case TaskTypeDevelopment:
		return r.activeDev < r.maxDevelopment
	default:
		return false
	}
}

// WaitForSlot waits for a slot to become available with a timeout
func (r *ResourceManager) WaitForSlot(ctx context.Context, taskType TaskType, taskID string, timeout time.Duration) (*Slot, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return r.AcquireSlot(ctx, taskType, taskID)
}

// ResourceStats provides information about resource usage
type ResourceStats struct {
	MaxClaude              int     `json:"max_claude"`
	MaxDevelopment         int     `json:"max_development"`
	ActiveDevelopment      int     `json:"active_development"`
	AvailableDevelopment   int     `json:"available_development"`
	TotalActive            int     `json:"total_active"`
	DevelopmentUtilization float64 `json:"development_utilization"`
}

// String returns a human-readable representation of the stats
func (s ResourceStats) String() string {
	return fmt.Sprintf(
		"Active: %d/%d total (%d dev) | Available: %d dev | Utilization: %.1f%% dev",
		s.TotalActive, s.MaxClaude,
		s.ActiveDevelopment,
		s.AvailableDevelopment,
		s.DevelopmentUtilization,
	)
}

// ResourceWaiter helps wait for resources to become available
type ResourceWaiter struct {
	manager   *ResourceManager
	taskType  TaskType
	taskID    string
	timeout   time.Duration
	waitStart time.Time
}

// NewResourceWaiter creates a new resource waiter
func NewResourceWaiter(manager *ResourceManager, taskType TaskType, taskID string, timeout time.Duration) *ResourceWaiter {
	return &ResourceWaiter{
		manager:   manager,
		taskType:  taskType,
		taskID:    taskID,
		timeout:   timeout,
		waitStart: time.Now(),
	}
}

// Wait waits for a resource slot to become available
func (w *ResourceWaiter) Wait(ctx context.Context) (*Slot, error) {
	// Check if we can acquire immediately
	if slot, err := w.manager.TryAcquireSlot(w.taskType, w.taskID); err == nil {
		return slot, nil
	}

	// Wait with timeout
	return w.manager.WaitForSlot(ctx, w.taskType, w.taskID, w.timeout)
}

// WaitTime returns how long we've been waiting
func (w *ResourceWaiter) WaitTime() time.Duration {
	return time.Since(w.waitStart)
}

// SlotManager provides higher-level slot management
type SlotManager struct {
	resourceMgr *ResourceManager
	activeSlots map[string]*Slot
	mu          sync.RWMutex
}

// NewSlotManager creates a new slot manager
func NewSlotManager(resourceMgr *ResourceManager) *SlotManager {
	return &SlotManager{
		resourceMgr: resourceMgr,
		activeSlots: make(map[string]*Slot),
	}
}

// AcquireForTask acquires a slot for a specific task
func (sm *SlotManager) AcquireForTask(ctx context.Context, task *Task) (*Slot, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if task already has a slot
	if slot, exists := sm.activeSlots[task.ID]; exists {
		return slot, nil
	}

	// Determine task type
	taskType := TaskTypeDevelopment

	// Acquire slot
	slot, err := sm.resourceMgr.AcquireSlot(ctx, taskType, task.ID)
	if err != nil {
		return nil, err
	}

	// Track slot
	sm.activeSlots[task.ID] = slot
	return slot, nil
}

// ReleaseForTask releases the slot for a specific task
func (sm *SlotManager) ReleaseForTask(taskID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	slot, exists := sm.activeSlots[taskID]
	if !exists {
		return fmt.Errorf("no slot found for task: %s", taskID)
	}

	slot.Release()
	delete(sm.activeSlots, taskID)
	return nil
}

// GetSlotForTask returns the slot for a specific task
func (sm *SlotManager) GetSlotForTask(taskID string) (*Slot, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	slot, exists := sm.activeSlots[taskID]
	return slot, exists
}

// GetActiveSlots returns all active slots
func (sm *SlotManager) GetActiveSlots() map[string]*Slot {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Slot, len(sm.activeSlots))
	for k, v := range sm.activeSlots {
		result[k] = v
	}
	return result
}

// Cleanup releases any orphaned slots
func (sm *SlotManager) Cleanup() int {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// For now, just release all slots
	// In a full implementation, this would check for orphaned slots
	count := len(sm.activeSlots)
	for taskID, slot := range sm.activeSlots {
		slot.Release()
		delete(sm.activeSlots, taskID)
	}

	return count
}

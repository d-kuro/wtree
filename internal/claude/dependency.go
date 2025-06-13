package claude

import (
	"fmt"
	"sort"
)

// DependencyGraph manages task dependencies and execution order.
type DependencyGraph struct {
	tasks map[string]*Task
	edges map[string][]string // task_id -> dependencies
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		tasks: make(map[string]*Task),
		edges: make(map[string][]string),
	}
}

// AddTask adds a task to the dependency graph.
func (dg *DependencyGraph) AddTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	if _, exists := dg.tasks[task.ID]; exists {
		return fmt.Errorf("task with ID %s already exists", task.ID)
	}

	dg.tasks[task.ID] = task
	dg.edges[task.ID] = make([]string, len(task.DependsOn))
	copy(dg.edges[task.ID], task.DependsOn)

	// Update blocks field for dependencies
	for _, depID := range task.DependsOn {
		if depTask, exists := dg.tasks[depID]; exists {
			depTask.Blocks = appendUnique(depTask.Blocks, task.ID)
		}
	}

	return nil
}

// ValidateDependencies checks for circular dependencies and missing dependencies.
func (dg *DependencyGraph) ValidateDependencies() error {
	// Check for missing dependencies
	for taskID, deps := range dg.edges {
		for _, depID := range deps {
			if _, exists := dg.tasks[depID]; !exists {
				return fmt.Errorf("task %s depends on non-existent task %s", taskID, depID)
			}
		}
	}

	// Check for circular dependencies using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for taskID := range dg.tasks {
		if !visited[taskID] {
			if dg.hasCycle(taskID, visited, recStack) {
				return fmt.Errorf("circular dependency detected involving task %s", taskID)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles in the dependency graph.
func (dg *DependencyGraph) hasCycle(taskID string, visited, recStack map[string]bool) bool {
	visited[taskID] = true
	recStack[taskID] = true

	// Check all dependencies
	for _, depID := range dg.edges[taskID] {
		if !visited[depID] {
			if dg.hasCycle(depID, visited, recStack) {
				return true
			}
		} else if recStack[depID] {
			return true
		}
	}

	recStack[taskID] = false
	return false
}

// GetExecutableTask returns the highest priority task that is ready to run.
func (dg *DependencyGraph) GetExecutableTask() (*Task, error) {
	readyTasks := dg.getReadyTasks()

	if len(readyTasks) == 0 {
		return nil, fmt.Errorf("no executable tasks available")
	}

	// Sort by priority (highest first), then by creation time (oldest first)
	sort.Slice(readyTasks, func(i, j int) bool {
		if readyTasks[i].Priority == readyTasks[j].Priority {
			return readyTasks[i].CreatedAt.Before(readyTasks[j].CreatedAt)
		}
		return readyTasks[i].Priority > readyTasks[j].Priority
	})

	return readyTasks[0], nil
}

// GetReadyTasks returns all tasks that are ready to execute.
func (dg *DependencyGraph) GetReadyTasks() []*Task {
	return dg.getReadyTasks()
}

// getReadyTasks finds tasks that have no pending dependencies.
func (dg *DependencyGraph) getReadyTasks() []*Task {
	var readyTasks []*Task

	for taskID, task := range dg.tasks {
		if task.Status == StatusPending && dg.areDependenciesCompleted(taskID) {
			readyTasks = append(readyTasks, task)
		}
	}

	return readyTasks
}

// areDependenciesCompleted checks if all dependencies for a task are completed.
func (dg *DependencyGraph) areDependenciesCompleted(taskID string) bool {
	task := dg.tasks[taskID]

	for _, depID := range task.DependsOn {
		depTask, exists := dg.tasks[depID]
		if !exists {
			return false // Dependency doesn't exist
		}

		switch depTask.Status {
		case StatusCompleted:
			continue // OK
		case StatusFailed:
			// Handle based on dependency policy
			switch task.DependencyPolicy {
			case DependencyPolicyFail:
				// Mark this task as failed
				task.Status = StatusFailed
				return false
			case DependencyPolicySkip:
				// Mark this task as skipped
				task.Status = StatusSkipped
				return false
			case DependencyPolicyWait:
				return false // Keep waiting
			}
		default:
			return false // Still pending/running
		}
	}

	return true
}

// GetTopologicalOrder returns tasks in topological order (dependencies first).
func (dg *DependencyGraph) GetTopologicalOrder() ([]*Task, error) {
	if err := dg.ValidateDependencies(); err != nil {
		return nil, err
	}

	// Kahn's algorithm for topological sorting
	inDegree := make(map[string]int)

	// Initialize in-degree count
	for taskID := range dg.tasks {
		inDegree[taskID] = 0
	}

	// Calculate in-degrees
	for taskID, deps := range dg.edges {
		inDegree[taskID] = len(deps)
	}

	// Queue for tasks with no dependencies
	var queue []string
	for taskID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, taskID)
		}
	}

	var result []*Task

	for len(queue) > 0 {
		// Sort queue by priority to maintain priority order among tasks at same level
		sort.Slice(queue, func(i, j int) bool {
			taskI := dg.tasks[queue[i]]
			taskJ := dg.tasks[queue[j]]
			if taskI.Priority == taskJ.Priority {
				return taskI.CreatedAt.Before(taskJ.CreatedAt)
			}
			return taskI.Priority > taskJ.Priority
		})

		// Remove task with highest priority
		current := queue[0]
		queue = queue[1:]
		result = append(result, dg.tasks[current])

		// Update in-degrees of tasks that depend on current task
		for taskID, deps := range dg.edges {
			for _, depID := range deps {
				if depID == current {
					inDegree[taskID]--
					if inDegree[taskID] == 0 {
						queue = append(queue, taskID)
					}
				}
			}
		}
	}

	// Check if all tasks were processed (no cycles)
	if len(result) != len(dg.tasks) {
		return nil, fmt.Errorf("circular dependency detected during topological sort")
	}

	return result, nil
}

// GetDependents returns tasks that depend on the given task.
func (dg *DependencyGraph) GetDependents(taskID string) []*Task {
	var dependents []*Task

	for id, task := range dg.tasks {
		for _, depID := range task.DependsOn {
			if depID == taskID {
				dependents = append(dependents, dg.tasks[id])
				break
			}
		}
	}

	return dependents
}

// GetDependencies returns tasks that the given task depends on.
func (dg *DependencyGraph) GetDependencies(taskID string) []*Task {
	task, exists := dg.tasks[taskID]
	if !exists {
		return nil
	}

	var dependencies []*Task
	for _, depID := range task.DependsOn {
		if depTask, exists := dg.tasks[depID]; exists {
			dependencies = append(dependencies, depTask)
		}
	}

	return dependencies
}

// UpdateTask updates a task in the dependency graph.
func (dg *DependencyGraph) UpdateTask(task *Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID cannot be empty")
	}

	dg.tasks[task.ID] = task
	dg.edges[task.ID] = make([]string, len(task.DependsOn))
	copy(dg.edges[task.ID], task.DependsOn)

	return nil
}

// RemoveTask removes a task from the dependency graph.
func (dg *DependencyGraph) RemoveTask(taskID string) {
	delete(dg.tasks, taskID)
	delete(dg.edges, taskID)

	// Remove this task from other tasks' dependencies
	for id, deps := range dg.edges {
		newDeps := make([]string, 0, len(deps))
		for _, depID := range deps {
			if depID != taskID {
				newDeps = append(newDeps, depID)
			}
		}
		dg.edges[id] = newDeps
		dg.tasks[id].DependsOn = newDeps
	}
}

// GetDependencyDepth returns the maximum dependency depth for the graph.
func (dg *DependencyGraph) GetDependencyDepth() int {
	maxDepth := 0

	for taskID := range dg.tasks {
		depth := dg.calculateDepth(taskID, make(map[string]bool))
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

// calculateDepth calculates the dependency depth for a specific task.
func (dg *DependencyGraph) calculateDepth(taskID string, visited map[string]bool) int {
	if visited[taskID] {
		return 0 // Avoid infinite recursion in case of cycles
	}

	visited[taskID] = true
	maxDepth := 0

	for _, depID := range dg.edges[taskID] {
		depth := dg.calculateDepth(depID, visited)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	delete(visited, taskID)
	return maxDepth + 1
}

// Helper functions

// appendUnique adds an item to a slice if it's not already present.
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

package claude

import (
	"testing"
	"time"
)

func TestNewDependencyGraph(t *testing.T) {
	dg := NewDependencyGraph()
	if dg == nil {
		t.Fatal("NewDependencyGraph() returned nil")
	}
	if dg.tasks == nil {
		t.Error("tasks map should be initialized")
	}
	if dg.edges == nil {
		t.Error("edges map should be initialized")
	}
}

func TestAddTask(t *testing.T) {
	dg := NewDependencyGraph()

	task := &Task{
		ID:        "test-task",
		Name:      "Test Task",
		Status:    StatusPending,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
		DependsOn: []string{},
	}

	err := dg.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask() failed: %v", err)
	}

	// Verify task was added
	if _, exists := dg.tasks[task.ID]; !exists {
		t.Error("Task was not added to tasks map")
	}

	// Verify edges were created
	if edges, exists := dg.edges[task.ID]; !exists {
		t.Error("Edges were not created for task")
	} else if len(edges) != 0 {
		t.Error("Expected empty edges for task with no dependencies")
	}
}

func TestAddTaskWithDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	// Add dependency task first
	depTask := &Task{
		ID:        "dep-task",
		Name:      "Dependency Task",
		Status:    StatusPending,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
		DependsOn: []string{},
		Blocks:    []string{},
	}

	err := dg.AddTask(depTask)
	if err != nil {
		t.Fatalf("AddTask() failed for dependency: %v", err)
	}

	// Add task with dependency
	task := &Task{
		ID:        "main-task",
		Name:      "Main Task",
		Status:    StatusPending,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
		DependsOn: []string{"dep-task"},
		Blocks:    []string{},
	}

	err = dg.AddTask(task)
	if err != nil {
		t.Fatalf("AddTask() failed for main task: %v", err)
	}

	// Verify dependency relationship
	if len(dg.edges[task.ID]) != 1 || dg.edges[task.ID][0] != "dep-task" {
		t.Error("Dependencies not correctly stored in edges")
	}

	// Verify blocks field was updated
	if len(depTask.Blocks) != 1 || depTask.Blocks[0] != "main-task" {
		t.Error("Blocks field not correctly updated in dependency task")
	}
}

func TestAddTaskErrors(t *testing.T) {
	dg := NewDependencyGraph()

	// Test empty ID
	task := &Task{
		ID:   "",
		Name: "Invalid Task",
	}

	err := dg.AddTask(task)
	if err == nil {
		t.Error("Expected error for empty task ID")
	}

	// Test duplicate ID
	validTask := &Task{
		ID:   "test-task",
		Name: "Valid Task",
	}

	err = dg.AddTask(validTask)
	if err != nil {
		t.Fatalf("AddTask() failed: %v", err)
	}

	duplicateTask := &Task{
		ID:   "test-task",
		Name: "Duplicate Task",
	}

	err = dg.AddTask(duplicateTask)
	if err == nil {
		t.Error("Expected error for duplicate task ID")
	}
}

func TestValidateDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	// Add valid tasks with dependencies
	task1 := &Task{ID: "task1", DependsOn: []string{}}
	task2 := &Task{ID: "task2", DependsOn: []string{"task1"}}
	task3 := &Task{ID: "task3", DependsOn: []string{"task2"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}

	err := dg.ValidateDependencies()
	if err != nil {
		t.Errorf("ValidateDependencies() failed for valid dependencies: %v", err)
	}
}

func TestValidateDependenciesCircular(t *testing.T) {
	dg := NewDependencyGraph()

	// Create circular dependency
	task1 := &Task{ID: "task1", DependsOn: []string{"task2"}}
	task2 := &Task{ID: "task2", DependsOn: []string{"task1"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}

	err := dg.ValidateDependencies()
	if err == nil {
		t.Error("Expected error for circular dependencies")
	}
}

func TestValidateDependenciesMissing(t *testing.T) {
	dg := NewDependencyGraph()

	// Create task with non-existent dependency
	task := &Task{ID: "task1", DependsOn: []string{"nonexistent"}}
	if err := dg.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	err := dg.ValidateDependencies()
	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

func TestGetReadyTasks(t *testing.T) {
	dg := NewDependencyGraph()

	// Add tasks with different dependency states
	task1 := &Task{
		ID:        "task1",
		Status:    StatusPending,
		DependsOn: []string{},
		CreatedAt: time.Now(),
		Priority:  PriorityNormal,
	}

	task2 := &Task{
		ID:        "task2",
		Status:    StatusPending,
		DependsOn: []string{"task1"},
		CreatedAt: time.Now().Add(1 * time.Second),
		Priority:  PriorityHigh,
	}

	task3 := &Task{
		ID:        "task3",
		Status:    StatusCompleted,
		DependsOn: []string{},
		CreatedAt: time.Now(),
		Priority:  PriorityLow,
	}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}

	readyTasks := dg.GetReadyTasks()

	// Only task1 should be ready (pending with no dependencies)
	if len(readyTasks) != 1 {
		t.Errorf("Expected 1 ready task, got %d", len(readyTasks))
	}

	if len(readyTasks) > 0 && readyTasks[0].ID != "task1" {
		t.Errorf("Expected task1 to be ready, got %s", readyTasks[0].ID)
	}

	// Complete task1 and check again
	task1.Status = StatusCompleted
	readyTasks = dg.GetReadyTasks()

	// Now task2 should be ready
	if len(readyTasks) != 1 {
		t.Errorf("Expected 1 ready task after completing task1, got %d", len(readyTasks))
	}

	if len(readyTasks) > 0 && readyTasks[0].ID != "task2" {
		t.Errorf("Expected task2 to be ready after completing task1, got %s", readyTasks[0].ID)
	}
}

func TestGetExecutableTask(t *testing.T) {
	dg := NewDependencyGraph()

	// Add tasks with different priorities
	task1 := &Task{
		ID:        "task1",
		Status:    StatusPending,
		DependsOn: []string{},
		CreatedAt: time.Now(),
		Priority:  PriorityLow,
	}

	task2 := &Task{
		ID:        "task2",
		Status:    StatusPending,
		DependsOn: []string{},
		CreatedAt: time.Now().Add(1 * time.Second),
		Priority:  PriorityHigh,
	}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}

	executableTask, err := dg.GetExecutableTask()
	if err != nil {
		t.Fatalf("GetExecutableTask() failed: %v", err)
	}

	// Should return task2 (higher priority)
	if executableTask.ID != "task2" {
		t.Errorf("Expected task2 (higher priority) to be executable, got %s", executableTask.ID)
	}
}

func TestGetExecutableTaskNone(t *testing.T) {
	dg := NewDependencyGraph()

	// Add task that's not ready
	task := &Task{
		ID:        "task1",
		Status:    StatusRunning,
		DependsOn: []string{},
	}

	if err := dg.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	_, err := dg.GetExecutableTask()
	if err == nil {
		t.Error("Expected error when no tasks are executable")
	}
}

func TestGetTopologicalOrder(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a dependency chain: task1 -> task2 -> task3
	task1 := &Task{
		ID:        "task1",
		DependsOn: []string{},
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
	}

	task2 := &Task{
		ID:        "task2",
		DependsOn: []string{"task1"},
		Priority:  PriorityNormal,
		CreatedAt: time.Now().Add(1 * time.Second),
	}

	task3 := &Task{
		ID:        "task3",
		DependsOn: []string{"task2"},
		Priority:  PriorityNormal,
		CreatedAt: time.Now().Add(2 * time.Second),
	}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}

	ordered, err := dg.GetTopologicalOrder()
	if err != nil {
		t.Fatalf("GetTopologicalOrder() failed: %v", err)
	}

	if len(ordered) != 3 {
		t.Errorf("Expected 3 tasks in topological order, got %d", len(ordered))
	}

	// Verify order: task1 should come before task2, task2 before task3
	taskMap := make(map[string]int)
	for i, task := range ordered {
		taskMap[task.ID] = i
	}

	if taskMap["task1"] >= taskMap["task2"] {
		t.Error("task1 should come before task2 in topological order")
	}

	if taskMap["task2"] >= taskMap["task3"] {
		t.Error("task2 should come before task3 in topological order")
	}
}

func TestGetDependencies(t *testing.T) {
	dg := NewDependencyGraph()

	task1 := &Task{ID: "task1", DependsOn: []string{}}
	task2 := &Task{ID: "task2", DependsOn: []string{}}
	task3 := &Task{ID: "task3", DependsOn: []string{"task1", "task2"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}

	deps := dg.GetDependencies("task3")
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies for task3, got %d", len(deps))
	}

	// Check that both dependencies are present
	foundTask1, foundTask2 := false, false
	for _, dep := range deps {
		if dep.ID == "task1" {
			foundTask1 = true
		}
		if dep.ID == "task2" {
			foundTask2 = true
		}
	}

	if !foundTask1 || !foundTask2 {
		t.Error("Not all dependencies were returned")
	}
}

func TestGetDependents(t *testing.T) {
	dg := NewDependencyGraph()

	task1 := &Task{ID: "task1", DependsOn: []string{}}
	task2 := &Task{ID: "task2", DependsOn: []string{"task1"}}
	task3 := &Task{ID: "task3", DependsOn: []string{"task1"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}

	dependents := dg.GetDependents("task1")
	if len(dependents) != 2 {
		t.Errorf("Expected 2 dependents for task1, got %d", len(dependents))
	}

	// Check that both dependents are present
	foundTask2, foundTask3 := false, false
	for _, dep := range dependents {
		if dep.ID == "task2" {
			foundTask2 = true
		}
		if dep.ID == "task3" {
			foundTask3 = true
		}
	}

	if !foundTask2 || !foundTask3 {
		t.Error("Not all dependents were returned")
	}
}

func TestUpdateTask(t *testing.T) {
	dg := NewDependencyGraph()

	task := &Task{
		ID:        "task1",
		Name:      "Original Name",
		DependsOn: []string{},
	}

	if err := dg.AddTask(task); err != nil {
		t.Fatalf("AddTask failed: %v", err)
	}

	// Update task
	updatedTask := &Task{
		ID:        "task1",
		Name:      "Updated Name",
		DependsOn: []string{},
	}

	err := dg.UpdateTask(updatedTask)
	if err != nil {
		t.Fatalf("UpdateTask() failed: %v", err)
	}

	// Verify update
	if dg.tasks["task1"].Name != "Updated Name" {
		t.Error("Task was not updated in dependency graph")
	}
}

func TestUpdateTaskEmptyID(t *testing.T) {
	dg := NewDependencyGraph()

	task := &Task{
		ID:   "",
		Name: "Invalid Task",
	}

	err := dg.UpdateTask(task)
	if err == nil {
		t.Error("Expected error for task with empty ID")
	}
}

func TestRemoveTask(t *testing.T) {
	dg := NewDependencyGraph()

	task1 := &Task{ID: "task1", DependsOn: []string{}}
	task2 := &Task{ID: "task2", DependsOn: []string{"task1"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}

	// Remove task1
	dg.RemoveTask("task1")

	// Verify task1 is removed
	if _, exists := dg.tasks["task1"]; exists {
		t.Error("Task1 should be removed from tasks map")
	}

	if _, exists := dg.edges["task1"]; exists {
		t.Error("Task1 should be removed from edges map")
	}

	// Verify task2's dependencies are updated
	if len(dg.tasks["task2"].DependsOn) != 0 {
		t.Error("Task2's dependencies should be updated after removing task1")
	}

	if len(dg.edges["task2"]) != 0 {
		t.Error("Task2's edges should be updated after removing task1")
	}
}

func TestGetDependencyDepth(t *testing.T) {
	dg := NewDependencyGraph()

	// Create a chain of dependencies
	task1 := &Task{ID: "task1", DependsOn: []string{}}
	task2 := &Task{ID: "task2", DependsOn: []string{"task1"}}
	task3 := &Task{ID: "task3", DependsOn: []string{"task2"}}
	task4 := &Task{ID: "task4", DependsOn: []string{"task3"}}

	if err := dg.AddTask(task1); err != nil {
		t.Fatalf("AddTask(task1) failed: %v", err)
	}
	if err := dg.AddTask(task2); err != nil {
		t.Fatalf("AddTask(task2) failed: %v", err)
	}
	if err := dg.AddTask(task3); err != nil {
		t.Fatalf("AddTask(task3) failed: %v", err)
	}
	if err := dg.AddTask(task4); err != nil {
		t.Fatalf("AddTask(task4) failed: %v", err)
	}

	depth := dg.GetDependencyDepth()
	if depth != 4 {
		t.Errorf("Expected dependency depth of 4, got %d", depth)
	}
}

func TestDependencyPolicyHandling(t *testing.T) {
	dg := NewDependencyGraph()

	// Create dependency task that fails
	depTask := &Task{
		ID:     "dep-task",
		Status: StatusFailed,
	}

	// Create task with different dependency policies
	waitTask := &Task{
		ID:               "wait-task",
		Status:           StatusPending,
		DependsOn:        []string{"dep-task"},
		DependencyPolicy: DependencyPolicyWait,
	}

	skipTask := &Task{
		ID:               "skip-task",
		Status:           StatusPending,
		DependsOn:        []string{"dep-task"},
		DependencyPolicy: DependencyPolicySkip,
	}

	failTask := &Task{
		ID:               "fail-task",
		Status:           StatusPending,
		DependsOn:        []string{"dep-task"},
		DependencyPolicy: DependencyPolicyFail,
	}

	if err := dg.AddTask(depTask); err != nil {
		t.Fatalf("AddTask(depTask) failed: %v", err)
	}
	if err := dg.AddTask(waitTask); err != nil {
		t.Fatalf("AddTask(waitTask) failed: %v", err)
	}
	if err := dg.AddTask(skipTask); err != nil {
		t.Fatalf("AddTask(skipTask) failed: %v", err)
	}
	if err := dg.AddTask(failTask); err != nil {
		t.Fatalf("AddTask(failTask) failed: %v", err)
	}

	// Test dependency completion checking
	waitReady := dg.areDependenciesCompleted("wait-task")
	skipReady := dg.areDependenciesCompleted("skip-task")
	failReady := dg.areDependenciesCompleted("fail-task")

	// All should return false, but status should be updated differently
	if waitReady || skipReady || failReady {
		t.Error("Tasks with failed dependencies should not be ready")
	}

	if waitTask.Status != StatusPending {
		t.Error("Wait task should remain pending")
	}

	if skipTask.Status != StatusSkipped {
		t.Error("Skip task should be marked as skipped")
	}

	if failTask.Status != StatusFailed {
		t.Error("Fail task should be marked as failed")
	}
}

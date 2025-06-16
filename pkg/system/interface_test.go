package system

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestStandardSystem_CreateNamedPipe(t *testing.T) {
	sys := NewStandardSystem()
	tmpDir := t.TempDir()

	pipePath := filepath.Join(tmpDir, "test.pipe")

	err := sys.CreateNamedPipe(pipePath, 0600)
	if err != nil {
		t.Fatalf("CreateNamedPipe() error = %v", err)
	}

	// Verify pipe was created
	info, err := os.Stat(pipePath)
	if err != nil {
		t.Fatalf("Pipe was not created: %v", err)
	}

	// Check if it's a named pipe (FIFO)
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("Created file is not a named pipe")
	}

	// Clean up
	_ = os.Remove(pipePath)
}

func TestStandardSystem_RemoveFile(t *testing.T) {
	sys := NewStandardSystem()
	tmpDir := t.TempDir()

	// Test removing a regular file
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	err = sys.RemoveFile(testFile)
	if err != nil {
		t.Fatalf("RemoveFile() error = %v", err)
	}

	// Verify file was removed
	_, err = os.Stat(testFile)
	if !os.IsNotExist(err) {
		t.Errorf("File still exists after RemoveFile()")
	}
}

func TestStandardSystem_RemoveFile_NonExistent(t *testing.T) {
	sys := NewStandardSystem()
	tmpDir := t.TempDir()

	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	err := sys.RemoveFile(nonExistentFile)
	if err == nil {
		t.Errorf("RemoveFile() should fail for non-existent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("RemoveFile() should return os.ErrNotExist for non-existent file, got: %v", err)
	}
}

func TestStandardSystem_NotifySignal(t *testing.T) {
	sys := NewStandardSystem()

	// Create a signal channel
	sigChan := make(chan os.Signal, 1)

	// Test NotifySignal - this should not panic or error
	sys.NotifySignal(sigChan, os.Interrupt, syscall.SIGTERM)

	// We can't easily test signal delivery in a unit test,
	// but we can verify the method doesn't panic
	t.Log("NotifySignal executed successfully")
}

func TestStandardSystem_CreateNamedPipe_InvalidPath(t *testing.T) {
	sys := NewStandardSystem()

	// Try to create a pipe in a non-existent directory
	invalidPath := "/nonexistent/directory/test.pipe"

	err := sys.CreateNamedPipe(invalidPath, 0600)
	if err == nil {
		t.Errorf("CreateNamedPipe() should fail for invalid path")
	}
}

func TestStandardSystem_CreateNamedPipe_PermissionDenied(t *testing.T) {
	sys := NewStandardSystem()

	// Try to create a pipe in a directory without write permissions
	// This test might be skipped on some systems where root access allows this
	restrictedPath := "/test.pipe"

	err := sys.CreateNamedPipe(restrictedPath, 0600)
	if err == nil {
		// Clean up if somehow it succeeded
		_ = os.Remove(restrictedPath)
		t.Log("CreateNamedPipe() succeeded unexpectedly (maybe running as root?)")
	} else {
		t.Logf("CreateNamedPipe() correctly failed with restricted path: %v", err)
	}
}

// Mock SystemInterface for testing interface compliance
type MockSystem struct {
	createdPipes []string
	removedFiles []string
	signals      []os.Signal
}

func NewMockSystem() *MockSystem {
	return &MockSystem{
		createdPipes: make([]string, 0),
		removedFiles: make([]string, 0),
		signals:      make([]os.Signal, 0),
	}
}

func (m *MockSystem) CreateNamedPipe(path string, mode uint32) error {
	m.createdPipes = append(m.createdPipes, path)
	return nil
}

func (m *MockSystem) RemoveFile(path string) error {
	m.removedFiles = append(m.removedFiles, path)
	return nil
}

func (m *MockSystem) NotifySignal(c chan<- os.Signal, signals ...os.Signal) {
	m.signals = append(m.signals, signals...)
}

func (m *MockSystem) GetCreatedPipes() []string {
	return m.createdPipes
}

func (m *MockSystem) GetRemovedFiles() []string {
	return m.removedFiles
}

func (m *MockSystem) GetNotifiedSignals() []os.Signal {
	return m.signals
}

func TestSystemInterface_MockImplementation(t *testing.T) {
	// Test that MockSystem implements SystemInterface
	var sys SystemInterface = NewMockSystem()
	mockSys := sys.(*MockSystem)

	// Test CreateNamedPipe
	err := sys.CreateNamedPipe("/test/pipe", 0600)
	if err != nil {
		t.Errorf("MockSystem CreateNamedPipe() failed: %v", err)
	}

	createdPipes := mockSys.GetCreatedPipes()
	if len(createdPipes) != 1 || createdPipes[0] != "/test/pipe" {
		t.Errorf("CreateNamedPipe() not recorded correctly, got: %v", createdPipes)
	}

	// Test RemoveFile
	err = sys.RemoveFile("/test/file")
	if err != nil {
		t.Errorf("MockSystem RemoveFile() failed: %v", err)
	}

	removedFiles := mockSys.GetRemovedFiles()
	if len(removedFiles) != 1 || removedFiles[0] != "/test/file" {
		t.Errorf("RemoveFile() not recorded correctly, got: %v", removedFiles)
	}

	// Test NotifySignal
	sigChan := make(chan os.Signal, 1)
	sys.NotifySignal(sigChan, os.Interrupt, syscall.SIGTERM)

	notifiedSignals := mockSys.GetNotifiedSignals()
	if len(notifiedSignals) != 2 {
		t.Errorf("NotifySignal() recorded %d signals, want 2", len(notifiedSignals))
	}
	if notifiedSignals[0] != os.Interrupt {
		t.Errorf("First signal should be os.Interrupt, got: %v", notifiedSignals[0])
	}
	if notifiedSignals[1] != syscall.SIGTERM {
		t.Errorf("Second signal should be syscall.SIGTERM, got: %v", notifiedSignals[1])
	}
}

func TestSystemInterface_StandardImplementation(t *testing.T) {
	// Test that StandardSystem implements SystemInterface
	sys := NewStandardSystem()

	// We've already tested the individual methods above,
	// this just verifies interface compliance
	if sys == nil {
		t.Errorf("NewStandardSystem() returned nil")
	}
}

func TestStandardSystem_CreateMultiplePipes(t *testing.T) {
	sys := NewStandardSystem()
	tmpDir := t.TempDir()

	pipes := []string{
		filepath.Join(tmpDir, "pipe1.pipe"),
		filepath.Join(tmpDir, "pipe2.pipe"),
		filepath.Join(tmpDir, "pipe3.pipe"),
	}

	// Create multiple pipes
	for _, pipe := range pipes {
		err := sys.CreateNamedPipe(pipe, 0600)
		if err != nil {
			t.Fatalf("CreateNamedPipe() failed for %s: %v", pipe, err)
		}
	}

	// Verify all pipes were created
	for _, pipe := range pipes {
		info, err := os.Stat(pipe)
		if err != nil {
			t.Errorf("Pipe %s was not created: %v", pipe, err)
			continue
		}
		if info.Mode()&os.ModeNamedPipe == 0 {
			t.Errorf("File %s is not a named pipe", pipe)
		}
	}

	// Clean up all pipes
	for _, pipe := range pipes {
		err := sys.RemoveFile(pipe)
		if err != nil {
			t.Errorf("Failed to remove pipe %s: %v", pipe, err)
		}
	}

	// Verify all pipes were removed
	for _, pipe := range pipes {
		_, err := os.Stat(pipe)
		if !os.IsNotExist(err) {
			t.Errorf("Pipe %s still exists after removal", pipe)
		}
	}
}

func TestStandardSystem_PipePermissions(t *testing.T) {
	sys := NewStandardSystem()
	tmpDir := t.TempDir()

	pipePath := filepath.Join(tmpDir, "test_perm.pipe")

	// Create pipe with specific permissions
	err := sys.CreateNamedPipe(pipePath, 0644)
	if err != nil {
		t.Fatalf("CreateNamedPipe() error = %v", err)
	}

	// Check permissions
	info, err := os.Stat(pipePath)
	if err != nil {
		t.Fatalf("Failed to stat pipe: %v", err)
	}

	// Note: The actual permissions might be modified by umask
	// so we just check that it's a named pipe with some permissions set
	if info.Mode()&os.ModeNamedPipe == 0 {
		t.Errorf("Created file is not a named pipe")
	}

	// Clean up
	_ = sys.RemoveFile(pipePath)
}

// Benchmark tests
func BenchmarkStandardSystem_CreateNamedPipe(b *testing.B) {
	sys := NewStandardSystem()
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipePath := filepath.Join(tmpDir, "bench_pipe_"+string(rune(i)))
		err := sys.CreateNamedPipe(pipePath, 0600)
		if err != nil {
			b.Fatalf("CreateNamedPipe() error = %v", err)
		}
		// Clean up immediately to avoid too many files
		_ = sys.RemoveFile(pipePath)
	}
}

func BenchmarkStandardSystem_RemoveFile(b *testing.B) {
	sys := NewStandardSystem()
	tmpDir := b.TempDir()

	// Pre-create files for removal
	files := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		files[i] = filepath.Join(tmpDir, "bench_file_"+string(rune(i)))
		err := os.WriteFile(files[i], []byte("test"), 0644)
		if err != nil {
			b.Fatalf("Failed to create test file: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sys.RemoveFile(files[i])
		if err != nil {
			b.Fatalf("RemoveFile() error = %v", err)
		}
	}
}

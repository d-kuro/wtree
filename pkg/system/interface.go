package system

import (
	"os"
	"os/exec"
	"os/signal"
)

// SystemInterface provides an abstraction layer for system calls
// This enables easier testing and mocking of system-level operations
type SystemInterface interface {
	// CreateNamedPipe creates a named pipe (FIFO) with the specified path and mode
	CreateNamedPipe(path string, mode uint32) error

	// RemoveFile removes a file or directory
	RemoveFile(path string) error

	// NotifySignal sets up signal notification for the given signals
	NotifySignal(c chan<- os.Signal, signals ...os.Signal)
}

// StandardSystem implements SystemInterface using standard Go library functions
type StandardSystem struct{}

// NewStandardSystem creates a new StandardSystem instance
func NewStandardSystem() SystemInterface {
	return &StandardSystem{}
}

// CreateNamedPipe creates a named pipe using the mkfifo command
func (s *StandardSystem) CreateNamedPipe(path string, mode uint32) error {
	// Use mkfifo command as a portable solution
	cmd := exec.Command("mkfifo", path)
	return cmd.Run()
}

// RemoveFile removes a file or directory
func (s *StandardSystem) RemoveFile(path string) error {
	return os.Remove(path)
}

// NotifySignal sets up signal notification
func (s *StandardSystem) NotifySignal(c chan<- os.Signal, signals ...os.Signal) {
	signal.Notify(c, signals...)
}

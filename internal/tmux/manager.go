package tmux

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type SessionManager struct {
	config  *SessionConfig
	tmuxCmd *TmuxCommand
}

func NewSessionManager(config *SessionConfig, dataDir string) *SessionManager {
	if config == nil {
		config = DefaultSessionConfig()
	}
	
	return &SessionManager{
		config:  config,
		tmuxCmd: NewTmuxCommand(config.TmuxCommand),
	}
}

func (s *SessionManager) CreateSession(ctx context.Context, opts SessionOptions) (*Session, error) {
	sessionName := fmt.Sprintf("gwq-%s-%s-%s", opts.Context, opts.Identifier, time.Now().Format("20060102150405"))
	
	if err := s.tmuxCmd.NewSessionContext(ctx, sessionName, opts.WorkingDir); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}
	
	if err := s.tmuxCmd.SetOptionContext(ctx, sessionName, "history-limit", s.config.HistoryLimit); err != nil {
		_ = s.tmuxCmd.KillSession(sessionName)
		return nil, fmt.Errorf("failed to set history limit: %w", err)
	}
	
	if opts.Command != "" {
		if err := s.tmuxCmd.SendKeysContext(ctx, sessionName, opts.Command); err != nil {
			_ = s.tmuxCmd.KillSession(sessionName)
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}
	
	session := &Session{
		ID:           generateID(),
		SessionName:  sessionName,
		Context:      opts.Context,
		Identifier:   opts.Identifier,
		WorkingDir:   opts.WorkingDir,
		Command:      opts.Command,
		StartTime:    time.Now(),
		Status:       StatusRunning,
		HistorySize:  s.config.HistoryLimit,
		Metadata:     opts.Metadata,
	}
	
	return session, nil
}

func (s *SessionManager) ListSessions() ([]*Session, error) {
	tmuxSessions, err := s.tmuxCmd.ListSessionsDetailed()
	if err != nil {
		return nil, err
	}
	
	var sessions []*Session
	for _, tmuxSession := range tmuxSessions {
		// Only show gwq-managed sessions
		if !strings.HasPrefix(tmuxSession.Name, "gwq-") {
			continue
		}
		
		session := s.parseSessionFromTmux(tmuxSession)
		if session != nil {
			sessions = append(sessions, session)
		}
	}
	
	return sessions, nil
}

func (s *SessionManager) parseSessionFromTmux(info *SessionInfo) *Session {
	// Parse session name format: gwq-{context}-{identifier}-{timestamp}
	re := regexp.MustCompile(`^gwq-([^-]+)-(.+)-(\d{14})$`)
	matches := re.FindStringSubmatch(info.Name)
	if len(matches) != 4 {
		return nil
	}
	
	context := matches[1]
	identifier := matches[2]
	timestamp := matches[3]
	
	startTime, err := time.Parse("20060102150405", timestamp)
	if err != nil {
		startTime = time.Now()
	}
	
	// Determine command from session name or current command
	command := info.CurrentCommand
	status := StatusRunning
	
	if command == "bash" || command == "zsh" || command == "sh" {
		// If shell is running, the original command likely finished but session is still active
		command = "Shell session (original command completed)"
		// Keep status as running since the session is still active
	}
	
	return &Session{
		ID:          generateShortID(),
		SessionName: info.Name,
		Context:     context,
		Identifier:  identifier,
		WorkingDir:  info.WorkingDir,
		Command:     command,
		StartTime:   startTime,
		Status:      status,
		HistorySize: s.config.HistoryLimit,
		Metadata:    map[string]string{},
	}
}

func (s *SessionManager) GetSession(id string) (*Session, error) {
	sessions, err := s.ListSessions()
	if err != nil {
		return nil, err
	}
	
	for _, session := range sessions {
		if session.ID == id || 
		   strings.Contains(session.SessionName, id) ||
		   session.Identifier == id ||
		   strings.Contains(session.Identifier, id) ||
		   strings.Contains(session.Context, id) {
			return session, nil
		}
	}
	
	return nil, fmt.Errorf("session not found: %s", id)
}

func (s *SessionManager) KillSession(id string) error {
	session, err := s.GetSession(id)
	if err != nil {
		return err
	}
	
	return s.KillSessionDirect(session)
}

func (s *SessionManager) KillSessionDirect(session *Session) error {
	if s.tmuxCmd.HasSession(session.SessionName) {
		if err := s.tmuxCmd.KillSession(session.SessionName); err != nil {
			return fmt.Errorf("failed to kill tmux session: %w", err)
		}
	}
	
	return nil
}

func (s *SessionManager) AttachSession(id string) error {
	session, err := s.GetSession(id)
	if err != nil {
		return err
	}
	
	return s.AttachSessionDirect(session)
}

func (s *SessionManager) AttachSessionDirect(session *Session) error {
	if !s.tmuxCmd.HasSession(session.SessionName) {
		return fmt.Errorf("tmux session %s no longer exists", session.SessionName)
	}
	
	return s.tmuxCmd.AttachSession(session.SessionName)
}


func generateID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func generateShortID() string {
	b := make([]byte, 3)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
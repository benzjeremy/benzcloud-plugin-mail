package mailstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Message represents an internal email.
type Message struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      string    `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	Folder  string    `json:"folder"` // "INBOX", "SENT", "TRASH"
	Date    time.Time `json:"date"`
	Read    bool      `json:"read"`
}

// Store manages mailbox files.
type Store struct {
	dataDir    string
	baseDomain string
	mailboxes  map[string][]*Message // keyed by username
	mu         sync.RWMutex
}

// NewStore initializes the mail store.
func NewStore(dataDir, baseDomain string) (*Store, error) {
	mbDir := filepath.Join(dataDir, "mailboxes")
	if err := os.MkdirAll(mbDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create mailboxes dir: %w", err)
	}

	s := &Store{
		dataDir:    mbDir,
		baseDomain: strings.ToLower(strings.Trim(baseDomain, ".")),
		mailboxes:  make(map[string][]*Message),
	}
	_ = s.loadAll()
	return s, nil
}

func (s *Store) userFile(username string) string {
	clean := strings.ToLower(strings.TrimSpace(username))
	clean = strings.ReplaceAll(clean, "/", "_")
	return filepath.Join(s.dataDir, clean+".json")
}

func (s *Store) loadAll() error {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			user := strings.TrimSuffix(e.Name(), ".json")
			data, err := os.ReadFile(filepath.Join(s.dataDir, e.Name()))
			if err == nil {
				var msgs []*Message
				if err := json.Unmarshal(data, &msgs); err == nil {
					s.mailboxes[user] = msgs
				}
			}
		}
	}
	return nil
}

func (s *Store) saveUser(username string) error {
	msgs := s.mailboxes[username]
	data, err := json.MarshalIndent(msgs, "", "  ")
	if err != nil {
		return err
	}
	uFile := s.userFile(username)
	tmp := uFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, uFile)
}

// Deliver stores an email in the recipient's INBOX and optionally sender's SENT folder.
func (s *Store) Deliver(from, to, subject, body string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	msgID := fmt.Sprintf("msg_%d_%d", time.Now().UnixNano(), len(body))
	now := time.Now().UTC()

	// Extract username from to address
	toUser := strings.ToLower(to)
	if idx := strings.Index(toUser, "@"); idx != -1 {
		toUser = toUser[:idx]
	}

	fromUser := strings.ToLower(from)
	if idx := strings.Index(fromUser, "@"); idx != -1 {
		fromUser = fromUser[:idx]
	}

	inboxMsg := &Message{
		ID:      msgID,
		From:    from,
		To:      to,
		Subject: subject,
		Body:    body,
		Folder:  "INBOX",
		Date:    now,
		Read:    false,
	}

	s.mailboxes[toUser] = append(s.mailboxes[toUser], inboxMsg)
	_ = s.saveUser(toUser)

	// Also record in sender's SENT folder
	sentMsg := &Message{
		ID:      msgID + "_sent",
		From:    from,
		To:      to,
		Subject: subject,
		Body:    body,
		Folder:  "SENT",
		Date:    now,
		Read:    true,
	}
	s.mailboxes[fromUser] = append(s.mailboxes[fromUser], sentMsg)
	_ = s.saveUser(fromUser)

	return inboxMsg, nil
}

// GetMessages returns messages for a user in a given folder.
func (s *Store) GetMessages(username, folder string) []*Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cleanUser := strings.ToLower(strings.TrimSpace(username))
	cleanFolder := strings.ToUpper(strings.TrimSpace(folder))

	var res []*Message
	for _, m := range s.mailboxes[cleanUser] {
		if m.Folder == cleanFolder || cleanFolder == "ALL" || cleanFolder == "" {
			copyM := *m
			res = append(res, &copyM)
		}
	}
	return res
}

// MarkAsRead marks a message as read.
func (s *Store) MarkAsRead(username, msgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanUser := strings.ToLower(strings.TrimSpace(username))
	for _, m := range s.mailboxes[cleanUser] {
		if m.ID == msgID {
			m.Read = true
			return s.saveUser(cleanUser)
		}
	}
	return fmt.Errorf("message %s not found", msgID)
}

// DeleteMessage moves message to TRASH or deletes permanently if already in TRASH.
func (s *Store) DeleteMessage(username, msgID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cleanUser := strings.ToLower(strings.TrimSpace(username))
	msgs := s.mailboxes[cleanUser]

	var updated []*Message
	for _, m := range msgs {
		if m.ID == msgID {
			if m.Folder != "TRASH" {
				m.Folder = "TRASH"
				updated = append(updated, m)
			}
			// If already in trash, drop it (permanent delete)
		} else {
			updated = append(updated, m)
		}
	}
	s.mailboxes[cleanUser] = updated
	return s.saveUser(cleanUser)
}

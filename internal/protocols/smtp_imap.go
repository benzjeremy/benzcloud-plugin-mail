package protocols

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/benzjeremy/benzcloud-plugin-mail/internal/mailstore"
)

// SMTPServer handles inbound SMTP email traffic.
type SMTPServer struct {
	port     int
	store    *mailstore.Store
	listener net.Listener
	running  bool
	mu       sync.RWMutex
}

// NewSMTPServer creates a new internal SMTP server.
func NewSMTPServer(port int, store *mailstore.Store) *SMTPServer {
	return &SMTPServer{
		port:  port,
		store: store,
	}
}

// Start launches the SMTP listener.
func (s *SMTPServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.listener = l
	s.running = true
	s.mu.Unlock()

	log.Printf("[BenzCloud Mail] SMTP Server listening on port %d\n", s.port)
	go s.acceptLoop()
	return nil
}

func (s *SMTPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *SMTPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleSMTPConnection(conn)
	}
}

func (s *SMTPServer) handleSMTPConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	_, _ = writer.WriteString("220 BenzCloud SMTP Service Ready\r\n")
	_ = writer.Flush()

	var from, to string
	var inData bool
	var dataLines []string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")

		if inData {
			if line == "." {
				inData = false
				subject := "No Subject"
				body := strings.Join(dataLines, "\n")
				// Simple subject parse
				for _, dl := range dataLines {
					if strings.HasPrefix(strings.ToLower(dl), "subject:") {
						subject = strings.TrimSpace(dl[8:])
						break
					}
				}
				_, _ = s.store.Deliver(from, to, subject, body)
				_, _ = writer.WriteString("250 2.0.0 OK: message queued\r\n")
				_ = writer.Flush()
				dataLines = nil
				continue
			}
			dataLines = append(dataLines, line)
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		cmd := strings.ToUpper(parts[0])

		switch cmd {
		case "HELO", "EHLO":
			_, _ = writer.WriteString("250-BenzCloud Hello\r\n250-8BITMIME\r\n250 OK\r\n")
		case "MAIL":
			if len(parts) > 1 && strings.HasPrefix(strings.ToUpper(parts[1]), "FROM:") {
				from = extractEmail(parts[1][5:])
				_, _ = writer.WriteString("250 2.1.0 Sender OK\r\n")
			} else {
				_, _ = writer.WriteString("501 Syntax error in parameters\r\n")
			}
		case "RCPT":
			if len(parts) > 1 && strings.HasPrefix(strings.ToUpper(parts[1]), "TO:") {
				to = extractEmail(parts[1][3:])
				_, _ = writer.WriteString("250 2.1.5 Recipient OK\r\n")
			} else {
				_, _ = writer.WriteString("501 Syntax error in parameters\r\n")
			}
		case "DATA":
			inData = true
			dataLines = nil
			_, _ = writer.WriteString("354 Start mail input; end with <CRLF>.<CRLF>\r\n")
		case "QUIT":
			_, _ = writer.WriteString("221 2.0.0 BenzCloud service closing transmission channel\r\n")
			_ = writer.Flush()
			return
		case "NOOP", "RSET":
			_, _ = writer.WriteString("250 OK\r\n")
		default:
			_, _ = writer.WriteString("500 5.5.1 Command unrecognized\r\n")
		}
		_ = writer.Flush()
	}
}

// IMAPServer provides standard IMAP4rev1 mailbox synchronization.
type IMAPServer struct {
	port     int
	store    *mailstore.Store
	listener net.Listener
	running  bool
	mu       sync.RWMutex
}

// NewIMAPServer creates a new internal IMAP server.
func NewIMAPServer(port int, store *mailstore.Store) *IMAPServer {
	return &IMAPServer{
		port:  port,
		store: store,
	}
}

// Start launches the IMAP server.
func (s *IMAPServer) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}
	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		s.mu.Unlock()
		return err
	}
	s.listener = l
	s.running = true
	s.mu.Unlock()

	log.Printf("[BenzCloud Mail] IMAP Server listening on port %d (Thunderbird compatible)\n", s.port)
	go s.acceptLoop()
	return nil
}

func (s *IMAPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.running = false
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *IMAPServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleIMAPConnection(conn)
	}
}

func (s *IMAPServer) handleIMAPConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	_, _ = writer.WriteString("* OK [CAPABILITY IMAP4rev1 AUTH=PLAIN] BenzCloud IMAP4rev1 Ready\r\n")
	_ = writer.Flush()

	var currentUser string

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			continue
		}

		tag := parts[0]
		cmd := strings.ToUpper(parts[1])

		switch cmd {
		case "CAPABILITY":
			_, _ = writer.WriteString("* CAPABILITY IMAP4rev1 AUTH=PLAIN\r\n")
			_, _ = writer.WriteString(fmt.Sprintf("%s OK CAPABILITY completed\r\n", tag))
		case "LOGIN":
			if len(parts) >= 3 {
				currentUser = strings.Trim(parts[2], "\"")
			}
			_, _ = writer.WriteString(fmt.Sprintf("%s OK [READ-WRITE] LOGIN completed\r\n", tag))
		case "SELECT":
			msgs := s.store.GetMessages(currentUser, "INBOX")
			_, _ = writer.WriteString(fmt.Sprintf("* %d EXISTS\r\n", len(msgs)))
			_, _ = writer.WriteString("* 0 RECENT\r\n")
			_, _ = writer.WriteString("* OK [PERMANENTFLAGS (\\Seen \\Deleted)] Flags permitted\r\n")
			_, _ = writer.WriteString(fmt.Sprintf("%s OK [READ-WRITE] SELECT completed\r\n", tag))
		case "FETCH":
			msgs := s.store.GetMessages(currentUser, "INBOX")
			for i, m := range msgs {
				bodyContent := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\n\r\n%s",
					m.From, m.To, m.Subject, m.Date.Format(time.RFC1123Z), m.Body)
				_, _ = writer.WriteString(fmt.Sprintf("* %d FETCH (FLAGS () RFC822.SIZE %d BODY[] {%d}\r\n%s)\r\n",
					i+1, len(bodyContent), len(bodyContent), bodyContent))
			}
			_, _ = writer.WriteString(fmt.Sprintf("%s OK FETCH completed\r\n", tag))
		case "LOGOUT":
			_, _ = writer.WriteString("* BYE BenzCloud IMAP server logging out\r\n")
			_, _ = writer.WriteString(fmt.Sprintf("%s OK LOGOUT completed\r\n", tag))
			_ = writer.Flush()
			return
		case "NOOP":
			_, _ = writer.WriteString(fmt.Sprintf("%s OK NOOP completed\r\n", tag))
		default:
			_, _ = writer.WriteString(fmt.Sprintf("%s OK %s completed\r\n", tag, cmd))
		}
		_ = writer.Flush()
	}
}

func extractEmail(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "<")
	s = strings.TrimSuffix(s, ">")
	return strings.ToLower(s)
}

package protocols

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/benzjeremy/benzcloud-plugin-mail/internal/mailstore"
)

func TestSMTPServer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "smtp_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := mailstore.NewStore(tmpDir, "local.benzcloud")
	if err != nil {
		t.Fatal(err)
	}

	smtpServer := NewSMTPServer(21025, store)
	if err := smtpServer.Start(); err != nil {
		t.Fatal(err)
	}
	defer smtpServer.Stop()

	// Connect to SMTP server
	conn, err := net.DialTimeout("tcp", "127.0.0.1:21025", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(banner, "220") {
		t.Fatalf("unexpected banner: %v (%s)", err, banner)
	}

	// Send EHLO
	fmt.Fprintf(conn, "EHLO client.test\r\n")
	for {
		line, _ := reader.ReadString('\n')
		if strings.HasPrefix(line, "250 ") {
			break
		}
	}

	// Send MAIL FROM
	fmt.Fprintf(conn, "MAIL FROM:<alice@local.benzcloud>\r\n")
	line, _ := reader.ReadString('\n')
	if !strings.HasPrefix(line, "250") {
		t.Fatalf("unexpected response to MAIL FROM: %s", line)
	}

	// Send RCPT TO
	fmt.Fprintf(conn, "RCPT TO:<bob@local.benzcloud>\r\n")
	line, _ = reader.ReadString('\n')
	if !strings.HasPrefix(line, "250") {
		t.Fatalf("unexpected response to RCPT TO: %s", line)
	}

	// Send DATA
	fmt.Fprintf(conn, "DATA\r\n")
	line, _ = reader.ReadString('\n')
	if !strings.HasPrefix(line, "354") {
		t.Fatalf("unexpected response to DATA: %s", line)
	}

	// Send body
	fmt.Fprintf(conn, "Subject: Hello Bob\r\n\r\nThis is a test message.\r\n.\r\n")
	line, _ = reader.ReadString('\n')
	if !strings.HasPrefix(line, "250") {
		t.Fatalf("unexpected response to message completion: %s", line)
	}

	// Send QUIT
	fmt.Fprintf(conn, "QUIT\r\n")

	// Check that bob received it
	msgs := store.GetMessages("bob", "INBOX")
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message in bob's inbox, got %d", len(msgs))
	}
	if msgs[0].Subject != "Hello Bob" {
		t.Fatalf("expected subject 'Hello Bob', got '%s'", msgs[0].Subject)
	}
}

func TestIMAPServer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "imap_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	store, err := mailstore.NewStore(tmpDir, "local.benzcloud")
	if err != nil {
		t.Fatal(err)
	}

	// Preload a message
	_, err = store.Deliver("alice@local.benzcloud", "bob@local.benzcloud", "Meeting Tomorrow", "Don't forget the sync.")
	if err != nil {
		t.Fatal(err)
	}

	imapServer := NewIMAPServer(21143, store)
	if err := imapServer.Start(); err != nil {
		t.Fatal(err)
	}
	defer imapServer.Stop()

	// Connect to IMAP server
	conn, err := net.DialTimeout("tcp", "127.0.0.1:21143", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil || !strings.Contains(banner, "* OK") {
		t.Fatalf("unexpected IMAP banner: %v (%s)", err, banner)
	}

	// Login
	fmt.Fprintf(conn, "A001 LOGIN bob password123\r\n")
	line, _ := reader.ReadString('\n')
	if !strings.Contains(line, "A001 OK") {
		t.Fatalf("unexpected response to LOGIN: %s", line)
	}

	// Select INBOX
	fmt.Fprintf(conn, "A002 SELECT INBOX\r\n")
	for {
		l, _ := reader.ReadString('\n')
		if strings.Contains(l, "A002 OK") {
			break
		}
	}

	// Fetch
	fmt.Fprintf(conn, "A003 FETCH 1 BODY[]\r\n")
	var fetchFound bool
	for {
		l, _ := reader.ReadString('\n')
		if strings.Contains(l, "Meeting Tomorrow") {
			fetchFound = true
		}
		if strings.Contains(l, "A003 OK") {
			break
		}
	}

	if !fetchFound {
		t.Fatalf("expected to find 'Meeting Tomorrow' in FETCH response")
	}

	// Logout
	fmt.Fprintf(conn, "A004 LOGOUT\r\n")
}

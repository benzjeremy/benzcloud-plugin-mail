package mailstore

import (
	"os"
	"testing"
)

func TestMailStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "benzcloud_mailstore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewStore(tempDir, "benzjeremy.de")
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// 1. Deliver message
	msg, err := store.Deliver("alice@benzjeremy.de", "bob@benzjeremy.de", "Project BenzCloud Kickoff", "Hi Bob, welcome to BenzCloud!")
	if err != nil {
		t.Fatalf("Deliver failed: %v", err)
	}

	// 2. Check Bob's inbox
	bobsInbox := store.GetMessages("bob", "INBOX")
	if len(bobsInbox) != 1 {
		t.Fatalf("Expected 1 message in Bob's inbox, got %d", len(bobsInbox))
	}
	if bobsInbox[0].Subject != "Project BenzCloud Kickoff" {
		t.Fatalf("Subject mismatch: %s", bobsInbox[0].Subject)
	}

	// 3. Check Alice's sent folder
	alicesSent := store.GetMessages("alice", "SENT")
	if len(alicesSent) != 1 {
		t.Fatalf("Expected 1 message in Alice's sent folder, got %d", len(alicesSent))
	}

	// 4. Mark as read
	if err := store.MarkAsRead("bob", msg.ID); err != nil {
		t.Fatalf("MarkAsRead failed: %v", err)
	}
	bobsInbox = store.GetMessages("bob", "INBOX")
	if !bobsInbox[0].Read {
		t.Fatal("Expected message to be read")
	}

	// 5. Move to TRASH and delete
	if err := store.DeleteMessage("bob", msg.ID); err != nil {
		t.Fatalf("DeleteMessage failed: %v", err)
	}
	bobsInbox = store.GetMessages("bob", "INBOX")
	if len(bobsInbox) != 0 {
		t.Fatal("Expected inbox to be empty after moving to trash")
	}
	bobsTrash := store.GetMessages("bob", "TRASH")
	if len(bobsTrash) != 1 {
		t.Fatal("Expected 1 message in trash")
	}
}

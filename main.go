package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/benzjeremy/benzcloud-plugin-mail/internal/mailstore"
	"github.com/benzjeremy/benzcloud-plugin-mail/internal/protocols"
)

//go:embed web/*
var webFS embed.FS

type SendRequest struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

func main() {
	httpPort := flag.Int("port", 8092, "HTTP webmail & API port")
	smtpPort := flag.Int("smtp", 1025, "SMTP server port")
	imapPort := flag.Int("imap", 1143, "IMAP server port")
	dataDir := flag.String("data", "./data", "Directory for mailboxes")
	baseDomain := flag.String("domain", "benzcloud.local", "Base domain for mail")
	flag.Parse()

	log.Printf("[BenzCloud Mail] Initializing Mail Plugin v1.0 on domain: %s\n", *baseDomain)

	store, err := mailstore.NewStore(*dataDir, *baseDomain)
	if err != nil {
		log.Fatalf("Failed to initialize mailstore: %v", err)
	}

	// Seed welcome email for admin if empty
	existing := store.GetMessages("admin", "ALL")
	if len(existing) == 0 {
		_, _ = store.Deliver(
			"system@"+*baseDomain,
			"admin@"+*baseDomain,
			"Welcome to BenzCloud Mail!",
			"Welcome to your private, self-hosted BenzCloud Mail service!\n\n"+
				"Features:\n"+
				"- Webmail UI with Inbox, Sent, and Trash\n"+
				"- Full SMTP (port "+fmt.Sprintf("%d", *smtpPort)+") and IMAP (port "+fmt.Sprintf("%d", *imapPort)+") support\n"+
				"- Out-of-the-box support for Thunderbird, K-9 Mail, and Apple Mail\n"+
				"- 100% private and decentralized within your Mesh VPN\n\n"+
				"Enjoy complete digital sovereignty!",
		)
	}

	// Launch SMTP server
	smtpServer := protocols.NewSMTPServer(*smtpPort, store)
	if err := smtpServer.Start(); err != nil {
		log.Printf("[BenzCloud Mail] Warning: SMTP server failed on port %d: %v", *smtpPort, err)
	}
	defer smtpServer.Stop()

	// Launch IMAP server
	imapServer := protocols.NewIMAPServer(*imapPort, store)
	if err := imapServer.Start(); err != nil {
		log.Printf("[BenzCloud Mail] Warning: IMAP server failed on port %d: %v", *imapPort, err)
	}
	defer imapServer.Stop()

	// HTTP Router
	mux := http.NewServeMux()

	// Health endpoint for BenzCloud Supervisor / Router
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"plugin":    "mail",
			"version":   "v1.0",
			"subdomain": "mail",
			"smtp_port": *smtpPort,
			"imap_port": *imapPort,
		})
	})

	// API: Users list
	mux.HandleFunc("/api/mail/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		users := []string{"admin"}
		_ = json.NewEncoder(w).Encode(users)
	})

	// API: Get messages
	mux.HandleFunc("/api/mail/messages", func(w http.ResponseWriter, r *http.Request) {
		user := r.URL.Query().Get("user")
		if user == "" {
			user = "admin"
		}
		folder := r.URL.Query().Get("folder")
		if folder == "" {
			folder = "INBOX"
		}
		msgs := store.GetMessages(user, folder)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msgs)
	})

	// API: Send message
	mux.HandleFunc("/api/mail/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req SendRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.To == "" {
			http.Error(w, "Recipient is required", http.StatusBadRequest)
			return
		}
		msg, err := store.Deliver(req.From, req.To, req.Subject, req.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(msg)
	})

	// API: Mark as read
	mux.HandleFunc("/api/mail/read", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		user := r.URL.Query().Get("user")
		id := r.URL.Query().Get("id")
		if user == "" || id == "" {
			http.Error(w, "Missing user or id", http.StatusBadRequest)
			return
		}
		if err := store.MarkAsRead(user, id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// API: Delete message
	mux.HandleFunc("/api/mail/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		user := r.URL.Query().Get("user")
		id := r.URL.Query().Get("id")
		if user == "" || id == "" {
			http.Error(w, "Missing user or id", http.StatusBadRequest)
			return
		}
		if err := store.DeleteMessage(user, id); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	// Static Webmail Files
	subWeb, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	fileServer := http.FileServer(http.FS(subWeb))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/health" {
			fileServer.ServeHTTP(w, r)
			return
		}
	})

	server := &http.Server{
		Addr:    fmt.Sprintf("0.0.0.0:%d", *httpPort),
		Handler: mux,
	}

	go func() {
		log.Printf("[BenzCloud Mail] Webmail UI & API listening on http://0.0.0.0:%d\n", *httpPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("[BenzCloud Mail] Shutting down gracefully...")
	_ = server.Close()
}

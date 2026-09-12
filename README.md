# BenzCloud Mail Plugin (`benzcloud-plugin-mail`)

[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://golang.org)
[![Version](https://img.shields.io/badge/Version-v1.0-green.svg)](https://github.com/benzjeremy/benzcloud-plugin-mail)

The **BenzCloud Mail Plugin** delivers an autonomous, privacy-respecting email subsystem for the BenzCloud decentralized mesh ecosystem. It provides integrated standard email protocol servers alongside an embedded modern webmail interface.

---

## 🌟 Key Features

- **Built-in SMTP Server (Port 1025):** RFC 5321 compliant internal mail submission and delivery engine.
- **Built-in IMAP Server (Port 1143):** RFC 3501 compliant mailbox synchronization engine supporting desktop and mobile clients (Mozilla Thunderbird, K-9 Mail, Apple Mail, Microsoft Outlook).
- **Embedded Webmail Client:** Ultra-fast, zero-dependency browser client with Cybernetic Dark UI, dual-language toggle (DE / EN), folder navigation (Inbox, Sent, Trash), and instant compose/reply workflow.
- **BenzCloud Service Integration:** Auto-registers via `/health` endpoint with `benzcloud-server` reverse proxy and DNS resolution (`mail.<domain>`).
- **Decentralized Storage:** Zero third-party cloud dependency; mail storage is kept in local encrypted JSON stores.

---

## 🚀 Quick Start

### Running the Plugin
```bash
./benzcloud-plugin-mail -port=8092 -smtp=1025 -imap=1143 -domain=benzcloud.local
```

### Command-line Options
- `-port <number>`: Webmail HTTP UI and REST API port (default: `8092`).
- `-smtp <number>`: Internal SMTP server port (default: `1025`).
- `-imap <number>`: Internal IMAP server port (default: `1143`).
- `-data <path>`: Directory for user mailbox storage (default: `./data`).
- `-domain <name>`: Base domain for email addresses (default: `benzcloud.local`).

---

## 📧 Client Configuration (Thunderbird / K-9 / Apple Mail)

| Setting | Value |
|---|---|
| **Incoming Server (IMAP)** | `mail.<your-domain>` or `127.0.0.1` |
| **IMAP Port** | `1143` |
| **Security** | None / STARTTLS |
| **Outgoing Server (SMTP)** | `mail.<your-domain>` or `127.0.0.1` |
| **SMTP Port** | `1025` |
| **Username** | Your username (e.g. `admin`) |

---

## 🛡️ License

This project is licensed under the [GNU General Public License v3.0 (GPL-3.0)](LICENSE).

const i18n = {
  de: {
    badge_local: "E-Mail Service",
    label_user: "Benutzer:",
    compose_btn: "Neue E-Mail",
    folder_inbox: "Posteingang",
    folder_sent: "Gesendet",
    folder_trash: "Papierkorb",
    proto_title: "Client-Verbindung",
    proto_desc: "Für Thunderbird, Outlook & K-9 Mail:",
    loading: "Lädt Nachrichten...",
    select_message_prompt: "Wähle eine E-Mail aus, um sie zu lesen.",
    btn_reply: "Antworten",
    btn_delete: "Löschen",
    label_from: "Von:",
    label_to: "An:",
    label_date: "Datum:",
    label_subject: "Betreff:",
    label_body: "Nachricht:",
    btn_cancel: "Abbrechen",
    btn_send: "Senden",
    modal_compose_title: "Neue Nachricht verfassen",
    no_messages: "Keine Nachrichten in diesem Ordner",
    send_success: "E-Mail erfolgreich gesendet!",
    send_error: "Fehler beim Senden der E-Mail.",
    reply_prefix: "Re: "
  },
  en: {
    badge_local: "Email Service",
    label_user: "User:",
    compose_btn: "New Email",
    folder_inbox: "Inbox",
    folder_sent: "Sent",
    folder_trash: "Trash",
    proto_title: "Client Connection",
    proto_desc: "For Thunderbird, Outlook & K-9 Mail:",
    loading: "Loading messages...",
    select_message_prompt: "Select an email to view its contents.",
    btn_reply: "Reply",
    btn_delete: "Delete",
    label_from: "From:",
    label_to: "To:",
    label_date: "Date:",
    label_subject: "Subject:",
    label_body: "Message:",
    btn_cancel: "Cancel",
    btn_send: "Send",
    modal_compose_title: "Compose New Message",
    no_messages: "No messages in this folder",
    send_success: "Email sent successfully!",
    send_error: "Failed to send email.",
    reply_prefix: "Re: "
  }
};

let currentLang = localStorage.getItem("benzcloud_lang") || "de";
let currentFolder = "INBOX";
let currentUser = "admin";
let currentMessages = [];
let selectedMessageId = null;

function setLang(lang) {
  currentLang = lang;
  localStorage.setItem("benzcloud_lang", lang);
  document.documentElement.lang = lang;

  document.querySelectorAll("[data-i18n]").forEach(el => {
    const key = el.getAttribute("data-i18n");
    if (i18n[lang] && i18n[lang][key]) {
      el.textContent = i18n[lang][key];
    }
  });

  document.getElementById("btnDe").classList.toggle("active", lang === "de");
  document.getElementById("btnEn").classList.toggle("active", lang === "en");

  updateFolderTitle();
}

function updateFolderTitle() {
  const titleEl = document.getElementById("currentFolderName");
  if (currentFolder === "INBOX") titleEl.textContent = i18n[currentLang].folder_inbox;
  else if (currentFolder === "SENT") titleEl.textContent = i18n[currentLang].folder_sent;
  else if (currentFolder === "TRASH") titleEl.textContent = i18n[currentLang].folder_trash;
}

// Fetch users
async function loadUsers() {
  try {
    const res = await fetch("/api/mail/users");
    if (res.ok) {
      const users = await res.json();
      const select = document.getElementById("userSelect");
      select.innerHTML = "";
      users.forEach(u => {
        const opt = document.createElement("option");
        opt.value = u;
        opt.textContent = u;
        select.appendChild(opt);
      });
      if (users.length > 0) {
        currentUser = users[0];
      }
    }
  } catch (e) {
    console.warn("Could not load users list, using admin default", e);
  }
}

// Fetch messages
async function loadMessages() {
  const listEl = document.getElementById("messageList");
  listEl.innerHTML = `<div class="empty-state">${i18n[currentLang].loading}</div>`;

  try {
    const res = await fetch(`/api/mail/messages?user=${encodeURIComponent(currentUser)}&folder=${currentFolder}`);
    if (!res.ok) throw new Error("HTTP error " + res.status);

    const msgs = await res.json();
    currentMessages = msgs || [];

    renderMessageList();
    updateUnreadCount();
  } catch (err) {
    listEl.innerHTML = `<div class="empty-state">Error loading messages: ${err.message}</div>`;
  }
}

function updateUnreadCount() {
  const unread = currentMessages.filter(m => !m.read && m.folder === "INBOX").length;
  const badge = document.getElementById("unreadBadge");
  badge.textContent = unread;
  badge.setAttribute("data-count", unread);
}

function renderMessageList() {
  const listEl = document.getElementById("messageList");
  listEl.innerHTML = "";

  if (currentMessages.length === 0) {
    listEl.innerHTML = `<div class="empty-state">${i18n[currentLang].no_messages}</div>`;
    return;
  }

  currentMessages.sort((a, b) => new Date(b.date) - new Date(a.date));

  currentMessages.forEach(msg => {
    const item = document.createElement("div");
    item.className = `mail-item ${!msg.read ? "unread" : ""} ${msg.id === selectedMessageId ? "selected" : ""}`;

    const dateStr = new Date(msg.date).toLocaleDateString(currentLang === "de" ? "de-DE" : "en-US", {
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit"
    });

    const displayContact = currentFolder === "SENT" ? `To: ${msg.to}` : msg.from;

    item.innerHTML = `
      <div class="mail-item-top">
        <span class="mail-sender">${escapeHtml(displayContact)}</span>
        <span class="mail-time">${dateStr}</span>
      </div>
      <div class="mail-subject">${escapeHtml(msg.subject || "(No Subject)")}</div>
      <div class="mail-preview">${escapeHtml(msg.body.substring(0, 80))}</div>
    `;

    item.addEventListener("click", () => selectMessage(msg));
    listEl.appendChild(item);
  });
}

function selectMessage(msg) {
  selectedMessageId = msg.id;
  msg.read = true;

  // API call to mark as read
  fetch(`/api/mail/read?user=${encodeURIComponent(currentUser)}&id=${encodeURIComponent(msg.id)}`, { method: "POST" });

  document.getElementById("emptyDetail").style.display = "none";
  document.getElementById("activeDetail").style.display = "flex";

  document.getElementById("detailSubject").textContent = msg.subject || "(No Subject)";
  document.getElementById("detailFrom").textContent = msg.from;
  document.getElementById("detailTo").textContent = msg.to;
  document.getElementById("detailDate").textContent = new Date(msg.date).toLocaleString(currentLang === "de" ? "de-DE" : "en-US");
  document.getElementById("detailBody").textContent = msg.body;

  renderMessageList();
  updateUnreadCount();
}

async function deleteSelectedMessage() {
  if (!selectedMessageId) return;
  try {
    await fetch(`/api/mail/delete?user=${encodeURIComponent(currentUser)}&id=${encodeURIComponent(selectedMessageId)}`, { method: "POST" });
    selectedMessageId = null;
    document.getElementById("emptyDetail").style.display = "flex";
    document.getElementById("activeDetail").style.display = "none";
    await loadMessages();
  } catch (err) {
    alert("Delete failed: " + err.message);
  }
}

function openCompose(replyTo = null) {
  const modal = document.getElementById("composeModal");
  modal.style.display = "flex";

  if (replyTo) {
    document.getElementById("composeTo").value = replyTo.from;
    const prefix = i18n[currentLang].reply_prefix;
    document.getElementById("composeSubject").value = replyTo.subject.startsWith("Re:") ? replyTo.subject : `${prefix}${replyTo.subject}`;
    document.getElementById("composeBody").value = `\n\n--- Original Message ---\nFrom: ${replyTo.from}\nDate: ${replyTo.date}\n\n${replyTo.body}`;
  } else {
    document.getElementById("composeTo").value = "";
    document.getElementById("composeSubject").value = "";
    document.getElementById("composeBody").value = "";
  }
}

function closeCompose() {
  document.getElementById("composeModal").style.display = "none";
}

async function handleSend(e) {
  e.preventDefault();
  const to = document.getElementById("composeTo").value.trim();
  const subject = document.getElementById("composeSubject").value.trim();
  const body = document.getElementById("composeBody").value.trim();

  try {
    const res = await fetch("/api/mail/send", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        from: `${currentUser}@benzcloud.local`,
        to: to,
        subject: subject,
        body: body
      })
    });

    if (!res.ok) throw new Error("Failed to deliver message");

    closeCompose();
    await loadMessages();
  } catch (err) {
    alert(i18n[currentLang].send_error + ": " + err.message);
  }
}

function escapeHtml(str) {
  return (str || "").replace(/[&<>"']/g, function (m) {
    return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[m];
  });
}

// Event Listeners
document.addEventListener("DOMContentLoaded", async () => {
  setLang(currentLang);

  document.getElementById("btnDe").addEventListener("click", () => setLang("de"));
  document.getElementById("btnEn").addEventListener("click", () => setLang("en"));

  document.querySelectorAll(".nav-item").forEach(btn => {
    btn.addEventListener("click", () => {
      document.querySelectorAll(".nav-item").forEach(b => b.classList.remove("active"));
      btn.classList.add("active");
      currentFolder = btn.getAttribute("data-folder");
      selectedMessageId = null;
      document.getElementById("emptyDetail").style.display = "flex";
      document.getElementById("activeDetail").style.display = "none";
      updateFolderTitle();
      loadMessages();
    });
  });

  document.getElementById("userSelect").addEventListener("change", (e) => {
    currentUser = e.target.value;
    loadMessages();
  });

  document.getElementById("btnRefresh").addEventListener("click", () => loadMessages());
  document.getElementById("btnCompose").addEventListener("click", () => openCompose());
  document.getElementById("btnModalClose").addEventListener("click", closeCompose);
  document.getElementById("btnCancelCompose").addEventListener("click", closeCompose);
  document.getElementById("composeForm").addEventListener("submit", handleSend);

  document.getElementById("btnDeleteMsg").addEventListener("click", deleteSelectedMessage);
  document.getElementById("btnReply").addEventListener("click", () => {
    const msg = currentMessages.find(m => m.id === selectedMessageId);
    if (msg) openCompose(msg);
  });

  await loadUsers();
  await loadMessages();
});

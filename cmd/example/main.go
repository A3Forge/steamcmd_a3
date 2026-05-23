package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/A3Forge/steamcmd_a3/steamcmd"
)

var (
	scmd        *steamcmd.SteamCmd
	broadcaster *sseBroadcaster
)

// --- SSE broadcaster ---

type sseBroadcaster struct {
	mu      sync.Mutex
	clients map[chan steamcmd.SteamCmdStatus]struct{}
}

func (b *sseBroadcaster) subscribe() chan steamcmd.SteamCmdStatus {
	ch := make(chan steamcmd.SteamCmdStatus, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *sseBroadcaster) unsubscribe(ch chan steamcmd.SteamCmdStatus) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *sseBroadcaster) broadcast(s steamcmd.SteamCmdStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- s:
		default:
		}
	}
}

// --- JSON helpers ---

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func decodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// --- Handlers ---

func handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, scmd.Status())
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Send current status immediately
	data, _ := json.Marshal(scmd.Status())
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	ch := broadcaster.subscribe()
	defer broadcaster.unsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case s, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(s)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// GET /api/users — returns users list + activeUserId
func handleGetUsers(w http.ResponseWriter, r *http.Request) {
	type safeUser struct {
		UserID   int    `json:"userId"`
		Username string `json:"username"`
	}
	type response struct {
		Users        []safeUser `json:"users"`
		ActiveUserID int        `json:"activeUserId"`
	}

	all := scmd.AllUsers()
	safe := make([]safeUser, len(all))
	for i, u := range all {
		safe[i] = safeUser{UserID: u.UserID, Username: u.Username}
	}

	writeJSON(w, response{
		Users:        safe,
		ActiveUserID: scmd.GetActiveUser().UserID,
	})
}

// POST /api/users
func handleAddUser(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Username == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	id, err := scmd.AddUser(body.Username, body.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]int{"id": id})
}

// DELETE /api/users/{id}
func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := scmd.DeleteUser(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// POST /api/users/{id}/activate
func handleActivateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := scmd.SetActiveUser(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// POST /api/login
func handleLogin(w http.ResponseWriter, r *http.Request) {
	err := scmd.TryLogin()
	if err == nil {
		writeJSON(w, map[string]bool{"ok": true})
		return
	}
	if err == steamcmd.ErrSteamCmdAlreadyActive {
		writeError(w, http.StatusConflict, "SteamCMD is busy")
		return
	}
	if err.Error() == steamcmd.STATUS_STEAMGUARD_CODE {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": steamcmd.STATUS_STEAMGUARD_CODE})
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

// POST /api/login/guard
func handleLoginGuard(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Code == "" {
		writeError(w, http.StatusBadRequest, "code required")
		return
	}
	if err := scmd.TryLoginWithSteamGuardCode(body.Code); err != nil {
		if err == steamcmd.ErrSteamCmdAlreadyActive {
			writeError(w, http.StatusConflict, "SteamCMD is busy")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// POST /api/server/update — {path: string}
func handleServerUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := decodeJSON(r, &body); err != nil || body.Path == "" {
		writeError(w, http.StatusBadRequest, "path required")
		return
	}
	if err := scmd.ValidateArma3Server(body.Path); err != nil {
		if err == steamcmd.ErrSteamCmdAlreadyActive {
			writeError(w, http.StatusConflict, "SteamCMD is busy")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// GET /api/mods/dir
func handleModsDir(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"path": scmd.ModsDir()})
}

// GET /api/mods
func handleGetMods(w http.ResponseWriter, r *http.Request) {
	mods, err := scmd.InstalledMods()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, mods)
}

// POST /api/mods/update — {ids: [string]}
func handleModsUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []string `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil || len(body.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "ids required")
		return
	}
	if err := scmd.ValidateMods(body.IDs); err != nil {
		if err == steamcmd.ErrSteamCmdAlreadyActive {
			writeError(w, http.StatusConflict, "SteamCMD is busy")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// GET /api/config
func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"steamCmdDir": scmd.GetSteamCmdDir()})
}

// POST /api/config — {steamCmdDir: string}
func handleSetConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SteamCmdDir string `json:"steamCmdDir"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.SteamCmdDir != "" {
		scmd.SetSteamCmdDir(body.SteamCmdDir)
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	scmd = steamcmd.NewSteamCmd()
	broadcaster = &sseBroadcaster{clients: make(map[chan steamcmd.SteamCmdStatus]struct{})}
	scmd.OnStatusChange = broadcaster.broadcast

	defer scmd.Close()
	scmd.CloseOnInterrupt()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/status", handleStatus)
	mux.HandleFunc("GET /api/events", handleEvents)

	mux.HandleFunc("GET /api/users", handleGetUsers)
	mux.HandleFunc("POST /api/users", handleAddUser)
	mux.HandleFunc("DELETE /api/users/{id}", handleDeleteUser)
	mux.HandleFunc("POST /api/users/{id}/activate", handleActivateUser)

	mux.HandleFunc("POST /api/login", handleLogin)
	mux.HandleFunc("POST /api/login/guard", handleLoginGuard)

	mux.HandleFunc("POST /api/server/update", handleServerUpdate)

	mux.HandleFunc("GET /api/mods/dir", handleModsDir)
	mux.HandleFunc("GET /api/mods", handleGetMods)
	mux.HandleFunc("POST /api/mods/update", handleModsUpdate)

	mux.HandleFunc("GET /api/config", handleGetConfig)
	mux.HandleFunc("POST /api/config", handleSetConfig)

	// catch-all: serve index.html
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	addr := ":8080"
	fmt.Printf("Server running at http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(mux)))
}

// Command teams-adapter is the outbound channel adapter for Microsoft Teams.
// It exposes POST /post, which the orchestrator calls to publish an approved
// brief/report. If TEAMS_WEBHOOK_URL is unset (P0 default) it logs a preview and
// returns {"status":"logged"} so the whole pipeline runs with zero credentials.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/aaraminds/scrum-master-agent/teams-adapter/internal/teams"
)

func main() {
	poster := teams.NewPoster(os.Getenv("TEAMS_WEBHOOK_URL"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /post", func(w http.ResponseWriter, r *http.Request) {
		var msg teams.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		delivered, err := poster.Post(msg)
		if err != nil {
			log.Printf("teams post error: %v", err)
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
			return
		}
		status := "delivered"
		if !delivered {
			status = "logged"
			log.Printf("[TEAMS STUB] no TEAMS_WEBHOOK_URL set — would post:\n--- %s ---\n%s\n---end---", msg.Title, msg.Markdown)
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": status})
	})

	addr := ":" + getenv("PORT", "8090")
	log.Printf("teams-adapter listening on %s (POST /post, GET /healthz)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("teams-adapter error: %v", err)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

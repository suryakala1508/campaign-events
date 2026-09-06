package main

import (
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"sort"
)

func main() {
	s := newStorage()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", s.handlePostEvents)
	mux.HandleFunc("GET /campaigns/{campaignID}/stats", s.handleGetStats)

	addr := ":8080"
	fmt.Printf("listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// handlePostEvents ingests a JSON array of events.
func (s *storage) handlePostEvents(w http.ResponseWriter, r *http.Request) {
	mediatype, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediatype != "application/json" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Content-Type must be application/json"})
		return
	}
	defer r.Body.Close()

	var events []Event
	if err := json.NewDecoder(r.Body).Decode(&events); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	if len(events) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "events array must not be empty"})
		return
	}

	var accepted, duplicates, rejected int
	for _, ev := range events {
		switch s.ingest(ev) {
		case ingestAccepted:
			accepted++
		case ingestDuplicate:
			duplicates++
		case ingestRejected:
			rejected++
		}
	}

	writeJSON(w, http.StatusOK, map[string]int{
		"received":   len(events),
		"accepted":   accepted,
		"duplicates": duplicates,
		"rejected":   rejected,
	})
}

// handleGetStats returns aggregated stats for one campaign.
func (s *storage) handleGetStats(w http.ResponseWriter, r *http.Request) {
	campaignID := r.PathValue("campaignID")
	if campaignID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "campaignID is required"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.stats[campaignID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "campaign not found"})
		return
	}

	daily := make([]DailyDelivered, 0, len(s.deliveredByDay[campaignID]))
	for date, count := range s.deliveredByDay[campaignID] {
		daily = append(daily, DailyDelivered{Date: date, Count: count})
	}
	sort.Slice(daily, func(i, j int) bool {
		return daily[i].Date < daily[j].Date
	})

	writeJSON(w, http.StatusOK, CampaignStatsResponse{
		CampaignID:     campaignID,
		Sent:           c.Sent,
		Delivered:      c.Delivered,
		Opened:         c.Opened,
		Clicked:        c.Clicked,
		UniqueOpens:    c.UniqueOpens,
		DailyDelivered: daily,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

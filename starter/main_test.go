package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func doPost(t *testing.T, s *storage, contentType, body string) (int, map[string]int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("POST /events", s.handlePostEvents)
	mux.ServeHTTP(rec, req)

	var resp map[string]int
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("non-JSON success body: %v (%s)", err, rec.Body.String())
		}
	}
	return rec.Code, resp
}

func TestPostEventsValid(t *testing.T) {
	s := newStorage()
	code, resp := doPost(t, s, "application/json", `[
		{"event_id":"evt_s1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"},
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:30:00Z"},
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_c1","campaign_id":"cmpA","contact_id":"ct_1","type":"clicked","timestamp":"2026-08-10T07:05:00Z","metadata":{"url":"https://example.com"}}
	]`)

	if code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if resp["received"] != 4 || resp["accepted"] != 4 || resp["duplicates"] != 0 || resp["rejected"] != 0 {
		t.Fatalf("unexpected counts: %v", resp)
	}

	c := s.stats["cmpA"]
	if c == nil {
		t.Fatal("no stats recorded for cmpA")
	}
	if c.Sent != 1 || c.Delivered != 1 || c.Opened != 1 || c.Clicked != 1 || c.UniqueOpens != 1 {
		t.Fatalf("unexpected stats: sent=%d delivered=%d opened=%d clicked=%d unique=%d",
			c.Sent, c.Delivered, c.Opened, c.Clicked, c.UniqueOpens)
	}
}

func TestPostEventsEmptyArray(t *testing.T) {
	code, _ := doPost(t, newStorage(), "application/json", `[]`)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestPostEventsEmptyBody(t *testing.T) {
	code, _ := doPost(t, newStorage(), "application/json", ``)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestPostEventsMalformedJSON(t *testing.T) {
	code, _ := doPost(t, newStorage(), "application/json", `[{"event_id":`)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestPostEventsMissingContentType(t *testing.T) {
	code, _ := doPost(t, newStorage(), "", `[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestPostEventsContentTypeWithCharset(t *testing.T) {
	// RFC 7231 allows parameters such as charset on a media type;
	// "application/json; charset=utf-8" is still JSON.
	code, resp := doPost(t, newStorage(), "application/json; charset=utf-8",
		`[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`)
	if code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if resp["accepted"] != 1 {
		t.Fatalf("unexpected counts: %v", resp)
	}
}

func TestPostEventsWrongContentType(t *testing.T) {
	code, _ := doPost(t, newStorage(), "text/plain",
		`[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`)
	if code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", code)
	}
}

func TestPostEventsBodyNotArray(t *testing.T) {
	for _, body := range []string{`{}`, `"not-an-array"`, `42`, `null`} {
		code, _ := doPost(t, newStorage(), "application/json", body)
		if code != http.StatusBadRequest {
			t.Fatalf("body %s: code = %d, want 400", body, code)
		}
	}
}

func TestPostEventsInvalidEvent(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"unsupported type", `[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"spam_report","timestamp":"2026-08-10T06:01:00Z"}]`},
		{"uppercase type", `[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"OPENED","timestamp":"2026-08-10T06:01:00Z"}]`},
		{"missing event_id", `[{"campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`},
		{"missing contact_id", `[{"event_id":"evt_1","campaign_id":"cmpA","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`},
		{"bad timestamp", `[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"not-a-time"}]`},
		{"non-rfc3339 timestamp", `[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"10-08-2026 09:00"}]`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newStorage()
			code, resp := doPost(t, s, "application/json", tc.body)
			if code != http.StatusOK {
				t.Fatalf("code = %d, want 200 (batch processed)", code)
			}
			if resp["received"] != 1 || resp["accepted"] != 0 || resp["rejected"] != 1 {
				t.Fatalf("unexpected counts: %v", resp)
			}
			if len(s.stats) != 0 {
				t.Fatalf("invalid event updated stats: %v", s.stats)
			}
		})
	}
}

func TestPostEventsInvalidDoesNotMarkSeen(t *testing.T) {
	s := newStorage()
	code, resp := doPost(t, s, "application/json",
		`[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"bogus","timestamp":"not-a-time"}]`)
	if code != http.StatusOK || resp["rejected"] != 1 {
		t.Fatalf("first invalid batch: code=%d resp=%v", code, resp)
	}

	code, resp = doPost(t, s, "application/json",
		`[{"event_id":"evt_1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`)
	if code != http.StatusOK || resp["accepted"] != 1 {
		t.Fatalf("corrected retry should be accepted: code=%d resp=%v", code, resp)
	}
}

func TestPostEventsDuplicateSameRequest(t *testing.T) {
	s := newStorage()
	code, resp := doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"}
	]`)

	if code != http.StatusOK {
		t.Fatalf("code = %d, want 200", code)
	}
	if resp["accepted"] != 1 || resp["duplicates"] != 1 {
		t.Fatalf("unexpected counts: %v", resp)
	}
	if c := s.stats["cmpA"]; c.Opened != 1 || c.UniqueOpens != 1 {
		t.Fatalf("duplicate affected stats: %+v", c)
	}
}

func TestPostEventsDuplicateAcrossRequests(t *testing.T) {
	s := newStorage()
	body := `[{"event_id":"evt_s1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`

	code, resp := doPost(t, s, "application/json", body)
	if code != http.StatusOK || resp["accepted"] != 1 {
		t.Fatalf("first request: code=%d resp=%v", code, resp)
	}

	code, resp = doPost(t, s, "application/json", body)
	if code != http.StatusOK || resp["accepted"] != 0 || resp["duplicates"] != 1 {
		t.Fatalf("second request: code=%d resp=%v", code, resp)
	}

	if c := s.stats["cmpA"]; c.Sent != 1 {
		t.Fatalf("event counted twice: %+v", c)
	}
}

func TestPostEventsConflictFirstWins(t *testing.T) {
	s := newStorage()
	// Same event_id, conflicting types: only the first occurrence counts.
	code, resp := doPost(t, s, "application/json",
		`[{"event_id":"evt_x","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"},
		   {"event_id":"evt_x","campaign_id":"cmpA","contact_id":"ct_1","type":"clicked","timestamp":"2026-08-10T06:02:00Z"}]`)
	if code != http.StatusOK || resp["accepted"] != 1 || resp["duplicates"] != 1 {
		t.Fatalf("unexpected counts: %v", resp)
	}
	if c := s.stats["cmpA"]; c.Sent != 1 || c.Clicked != 0 {
		t.Fatalf("conflict not resolved first-wins: %+v", c)
	}
}

func TestPostEventsConflictAcrossCampaigns(t *testing.T) {
	s := newStorage()
	// First request accepts evt_1 for cmp_A.
	code, resp := doPost(t, s, "application/json",
		`[{"event_id":"evt_1","campaign_id":"cmp_A","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`)
	if code != http.StatusOK || resp["accepted"] != 1 {
		t.Fatalf("first request: code=%d resp=%v", code, resp)
	}

	// Second request reuses evt_1 for a *different* campaign: it is a
	// duplicate and must not modify cmp_B at all.
	code, resp = doPost(t, s, "application/json",
		`[{"event_id":"evt_1","campaign_id":"cmp_B","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:30:00Z"}]`)
	if code != http.StatusOK || resp["accepted"] != 0 || resp["duplicates"] != 1 {
		t.Fatalf("second request: code=%d resp=%v", code, resp)
	}

	if b, ok := s.stats["cmp_B"]; ok {
		t.Fatalf("duplicate event modified cmp_B: %+v", b)
	}
	if b, ok := s.deliveredByDay["cmp_B"]; ok {
		t.Fatalf("duplicate event created cmp_B daily buckets: %+v", b)
	}
	if a := s.stats["cmp_A"]; a.Sent != 1 || a.Delivered != 0 {
		t.Fatalf("cmp_A stats changed: %+v", a)
	}
}

func TestPostEventsRepeatedOpenSameContact(t *testing.T) {
	s := newStorage()
	code, resp := doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o2","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T09:00:00Z"}
	]`)

	if code != http.StatusOK || resp["accepted"] != 2 {
		t.Fatalf("unexpected counts: %v", resp)
	}
	if c := s.stats["cmpA"]; c.Opened != 2 || c.UniqueOpens != 1 {
		t.Fatalf("wanted opened=2 unique=1, got %+v", c)
	}
}

func TestPostEventsSameContactDifferentCampaigns(t *testing.T) {
	s := newStorage()
	// From the spec:
	// cmp_A+ct_1+opened, cmp_A+ct_1+opened, cmp_A+ct_2+opened, cmp_B+ct_1+opened
	// => cmp_A: opened=3 unique_opens=2, cmp_B: opened=1 unique_opens=1
	code, resp := doPost(t, s, "application/json", `[
		{"event_id":"evt_1","campaign_id":"cmp_A","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_2","campaign_id":"cmp_A","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T08:00:00Z"},
		{"event_id":"evt_3","campaign_id":"cmp_A","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T09:00:00Z"},
		{"event_id":"evt_4","campaign_id":"cmp_B","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"}
	]`)

	if code != http.StatusOK || resp["accepted"] != 4 {
		t.Fatalf("unexpected counts: %v", resp)
	}

	a := s.stats["cmp_A"]
	if a == nil || a.Opened != 3 || a.UniqueOpens != 2 {
		t.Fatalf("cmp_A want opened=3 unique=2, got %+v", a)
	}
	b := s.stats["cmp_B"]
	if b == nil || b.Opened != 1 || b.UniqueOpens != 1 {
		t.Fatalf("cmp_B want opened=1 unique=1, got %+v", b)
	}
}

func doGetStats(t *testing.T, s *storage, campaignID string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/campaigns/"+campaignID+"/stats", nil)
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /campaigns/{campaignID}/stats", s.handleGetStats)
	mux.ServeHTTP(rec, req)
	return rec
}

func decodeStats(t *testing.T, rec *httptest.ResponseRecorder) CampaignStatsResponse {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("code = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp CampaignStatsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("non-JSON stats body: %v (%s)", err, rec.Body.String())
	}
	return resp
}

func TestGetStatsExistingCampaign(t *testing.T) {
	s := newStorage()
	code, _ := doPost(t, s, "application/json", `[
		{"event_id":"evt_s1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"},
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:30:00Z"},
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_c1","campaign_id":"cmpA","contact_id":"ct_1","type":"clicked","timestamp":"2026-08-10T07:05:00Z"}
	]`)
	if code != http.StatusOK {
		t.Fatalf("post code = %d, want 200", code)
	}

	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.CampaignID != "cmpA" {
		t.Fatalf("campaign_id = %q, want cmpA", resp.CampaignID)
	}
	if resp.Sent != 1 || resp.Delivered != 1 || resp.Opened != 1 || resp.Clicked != 1 || resp.UniqueOpens != 1 {
		t.Fatalf("unexpected stats: %+v", resp)
	}
}

func TestGetStatsUnknownCampaign(t *testing.T) {
	s := newStorage()
	rec := doGetStats(t, s, "does_not_exist")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want 404", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("non-JSON 404 body: %v (%s)", err, rec.Body.String())
	}
	if body["error"] != "campaign not found" {
		t.Fatalf("error = %q, want %q", body["error"], "campaign not found")
	}
}

func TestGetStatsEmptyCampaignID(t *testing.T) {
	s := newStorage()
	// Called directly (no ServeMux) so PathValue is unset/empty.
	req := httptest.NewRequest(http.MethodGet, "/campaigns/cmpA/stats", nil)
	rec := httptest.NewRecorder()
	s.handleGetStats(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code = %d, want 400", rec.Code)
	}
}

func TestGetStatsSentCount(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_s1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"},
		{"event_id":"evt_s2","campaign_id":"cmpA","contact_id":"ct_2","type":"sent","timestamp":"2026-08-10T06:02:00Z"},
		{"event_id":"evt_s3","campaign_id":"cmpA","contact_id":"ct_3","type":"sent","timestamp":"2026-08-10T06:03:00Z"}
	]`)
	if got := decodeStats(t, doGetStats(t, s, "cmpA")).Sent; got != 3 {
		t.Fatalf("sent = %d, want 3", got)
	}
}

func TestGetStatsDeliveredCount(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:10:00Z"},
		{"event_id":"evt_d2","campaign_id":"cmpA","contact_id":"ct_2","type":"delivered","timestamp":"2026-08-10T06:11:00Z"},
		{"event_id":"evt_d3","campaign_id":"cmpA","contact_id":"ct_3","type":"delivered","timestamp":"2026-08-11T06:12:00Z"}
	]`)
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Delivered != 3 {
		t.Fatalf("delivered = %d, want 3", resp.Delivered)
	}
}

func TestGetStatsOpenedCount(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o2","campaign_id":"cmpA","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T07:01:00Z"}
	]`)
	if got := decodeStats(t, doGetStats(t, s, "cmpA")).Opened; got != 2 {
		t.Fatalf("opened = %d, want 2", got)
	}
}

func TestGetStatsClickedCount(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_c1","campaign_id":"cmpA","contact_id":"ct_1","type":"clicked","timestamp":"2026-08-10T07:05:00Z"},
		{"event_id":"evt_c2","campaign_id":"cmpA","contact_id":"ct_2","type":"clicked","timestamp":"2026-08-10T07:06:00Z"}
	]`)
	if got := decodeStats(t, doGetStats(t, s, "cmpA")).Clicked; got != 2 {
		t.Fatalf("clicked = %d, want 2", got)
	}
}

func TestGetStatsUniqueOpens(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o2","campaign_id":"cmpA","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T07:01:00Z"},
		{"event_id":"evt_o3","campaign_id":"cmpA","contact_id":"ct_3","type":"opened","timestamp":"2026-08-10T07:02:00Z"}
	]`)
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Opened != 3 || resp.UniqueOpens != 3 {
		t.Fatalf("opened = %d unique = %d, want 3 and 3", resp.Opened, resp.UniqueOpens)
	}
}

func TestGetStatsRepeatedOpensSameContact(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o2","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T09:00:00Z"},
		{"event_id":"evt_o3","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T11:00:00Z"}
	]`)
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Opened != 3 || resp.UniqueOpens != 1 {
		t.Fatalf("opened = %d unique = %d, want 3 and 1", resp.Opened, resp.UniqueOpens)
	}
}

func TestGetStatsUniqueOpensMixedContacts(t *testing.T) {
	// contact 1 opens 3 times, contact 2 opens 2 times:
	// opened = 5, unique_opens = 2.
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_o2","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T08:00:00Z"},
		{"event_id":"evt_o3","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T09:00:00Z"},
		{"event_id":"evt_o4","campaign_id":"cmpA","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T10:00:00Z"},
		{"event_id":"evt_o5","campaign_id":"cmpA","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T11:00:00Z"}
	]`)
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Opened != 5 || resp.UniqueOpens != 2 {
		t.Fatalf("opened = %d unique = %d, want 5 and 2", resp.Opened, resp.UniqueOpens)
	}
}

func TestGetStatsSameContactDifferentCampaigns(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_1","campaign_id":"cmp_A","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_2","campaign_id":"cmp_A","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T08:00:00Z"},
		{"event_id":"evt_3","campaign_id":"cmp_A","contact_id":"ct_2","type":"opened","timestamp":"2026-08-10T09:00:00Z"},
		{"event_id":"evt_4","campaign_id":"cmp_B","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"}
	]`)

	a := decodeStats(t, doGetStats(t, s, "cmp_A"))
	if a.Opened != 3 || a.UniqueOpens != 2 {
		t.Fatalf("cmp_A opened = %d unique = %d, want 3 and 2", a.Opened, a.UniqueOpens)
	}
	b := decodeStats(t, doGetStats(t, s, "cmp_B"))
	if b.Opened != 1 || b.UniqueOpens != 1 {
		t.Fatalf("cmp_B opened = %d unique = %d, want 1 and 1", b.Opened, b.UniqueOpens)
	}
}

func TestGetStatsDailyDeliveredUTCDates(t *testing.T) {
	s := newStorage()
	// Event timestamps are parsed as RFC 3339 and bucketed by their UTC date:
	//   2026-08-10T23:30:00Z   -> 2026-08-10 (same UTC day)
	//   2026-08-11T00:30:00Z   -> 2026-08-11 (crosses UTC midnight)
	//   2026-08-10T20:00:00-05:00 -> 2026-08-11 (offset converted to UTC)
	doPost(t, s, "application/json", `[
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T23:30:00Z"},
		{"event_id":"evt_d2","campaign_id":"cmpA","contact_id":"ct_2","type":"delivered","timestamp":"2026-08-11T00:30:00Z"},
		{"event_id":"evt_d3","campaign_id":"cmpA","contact_id":"ct_3","type":"delivered","timestamp":"2026-08-10T20:00:00-05:00"},
		{"event_id":"evt_d4","campaign_id":"cmpA","contact_id":"ct_4","type":"delivered","timestamp":"2026-08-10T23:45:00Z"}
	]`)

	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Delivered != 4 {
		t.Fatalf("delivered = %d, want 4", resp.Delivered)
	}
	want := []DailyDelivered{
		{Date: "2026-08-10", Count: 2},
		{Date: "2026-08-11", Count: 2},
	}
	if len(resp.DailyDelivered) != len(want) {
		t.Fatalf("daily buckets = %+v, want %+v", resp.DailyDelivered, want)
	}
	for i := range want {
		if resp.DailyDelivered[i] != want[i] {
			t.Fatalf("daily buckets = %+v, want %+v", resp.DailyDelivered, want)
		}
	}
}

func TestGetStatsDailyDeliveredOnlyDelivered(t *testing.T) {
	// sent/opened/clicked events must not create daily_delivered buckets.
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_s1","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"},
		{"event_id":"evt_o1","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"},
		{"event_id":"evt_c1","campaign_id":"cmpA","contact_id":"ct_1","type":"clicked","timestamp":"2026-08-10T07:05:00Z"}
	]`)
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if resp.Sent != 1 || resp.Opened != 1 || resp.Clicked != 1 {
		t.Fatalf("unexpected counters: %+v", resp)
	}
	if len(resp.DailyDelivered) != 0 {
		t.Fatalf("daily_delivered non-empty for non-delivered events: %+v", resp.DailyDelivered)
	}
}

func TestGetStatsDailyDeliveredDuplicate(t *testing.T) {
	s := newStorage()
	body := `[{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:10:00Z"}]`
	if code, _ := doPost(t, s, "application/json", body); code != http.StatusOK {
		t.Fatalf("first post code = %d, want 200", code)
	}
	if code, resp := doPost(t, s, "application/json", body); code != http.StatusOK || resp["duplicates"] != 1 {
		t.Fatalf("second post: code=%d resp=%v", code, resp)
	}
	if resp := decodeStats(t, doGetStats(t, s, "cmpA")); resp.Delivered != 1 {
		t.Fatalf("delivered = %d, want 1", resp.Delivered)
	}
	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	want := []DailyDelivered{{Date: "2026-08-10", Count: 1}}
	if len(resp.DailyDelivered) != 1 || resp.DailyDelivered[0] != want[0] {
		t.Fatalf("daily buckets = %+v, want %+v", resp.DailyDelivered, want)
	}
}

func TestGetStatsDailyDeliveredInvalid(t *testing.T) {
	// A delivered event that fails validation must not be counted at all.
	s := newStorage()
	for _, body := range []string{
		`[{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"not-a-time"}]`,
		`[{"event_id":"evt_d2","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"10-08-2026 09:00"}]`,
	} {
		code, resp := doPost(t, s, "application/json", body)
		if code != http.StatusOK || resp["rejected"] != 1 {
			t.Fatalf("body %s: code=%d resp=%v", body, code, resp)
		}
	}
	if rec := doGetStats(t, s, "cmpA"); rec.Code != http.StatusNotFound {
		t.Fatalf("invalid events created a campaign: code=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGetStatsDailyDeliveredCampaignIsolation(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:10:00Z"},
		{"event_id":"evt_d2","campaign_id":"cmpA","contact_id":"ct_2","type":"delivered","timestamp":"2026-08-10T07:10:00Z"},
		{"event_id":"evt_d3","campaign_id":"cmpB","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:10:00Z"},
		{"event_id":"evt_d4","campaign_id":"cmpB","contact_id":"ct_2","type":"delivered","timestamp":"2026-08-11T06:10:00Z"}
	]`)

	a := decodeStats(t, doGetStats(t, s, "cmpA"))
	wantA := []DailyDelivered{{Date: "2026-08-10", Count: 2}}
	if len(a.DailyDelivered) != 1 || a.DailyDelivered[0] != wantA[0] {
		t.Fatalf("cmpA daily = %+v, want %+v", a.DailyDelivered, wantA)
	}

	b := decodeStats(t, doGetStats(t, s, "cmpB"))
	wantB := []DailyDelivered{
		{Date: "2026-08-10", Count: 1},
		{Date: "2026-08-11", Count: 1},
	}
	if len(b.DailyDelivered) != 2 {
		t.Fatalf("cmpB daily = %+v, want %+v", b.DailyDelivered, wantB)
	}
	for i := range wantB {
		if b.DailyDelivered[i] != wantB[i] {
			t.Fatalf("cmpB daily = %+v, want %+v", b.DailyDelivered, wantB)
		}
	}
}

func TestGetStatsDailyDeliveredDeterministic(t *testing.T) {
	s := newStorage()
	doPost(t, s, "application/json", `[
		{"event_id":"evt_d1","campaign_id":"cmpA","contact_id":"ct_1","type":"delivered","timestamp":"2026-08-10T06:10:00Z"},
		{"event_id":"evt_d2","campaign_id":"cmpA","contact_id":"ct_2","type":"delivered","timestamp":"2026-08-12T06:11:00Z"},
		{"event_id":"evt_d3","campaign_id":"cmpA","contact_id":"ct_3","type":"delivered","timestamp":"2026-08-11T06:12:00Z"},
		{"event_id":"evt_d4","campaign_id":"cmpA","contact_id":"ct_4","type":"delivered","timestamp":"2026-08-30T06:13:00Z"},
		{"event_id":"evt_d5","campaign_id":"cmpA","contact_id":"ct_5","type":"delivered","timestamp":"2026-08-02T06:14:00Z"}
	]`)

	first := doGetStats(t, s, "cmpA").Body.String()
	for i := 0; i < 5; i++ {
		if got := doGetStats(t, s, "cmpA").Body.String(); got != first {
			t.Fatalf("non-deterministic response: run %d differs:\n%s\nvs\n%s", i, got, first)
		}
	}

	resp := decodeStats(t, doGetStats(t, s, "cmpA"))
	if len(resp.DailyDelivered) != 5 {
		t.Fatalf("got %d daily buckets, want 5", len(resp.DailyDelivered))
	}
	last := ""
	for i, d := range resp.DailyDelivered {
		if d.Date <= last {
			t.Fatalf("bucket %d date %q not strictly ascending after %q: %+v", i, d.Date, last, resp.DailyDelivered)
		}
		last = d.Date
	}
}

func TestPostEventsConcurrentDifferentEvents(t *testing.T) {
	s := newStorage()

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted, duplicates := 0, 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`[{"event_id":"evt_%d","campaign_id":"cmpA","contact_id":"ct_1","type":"opened","timestamp":"2026-08-10T07:00:00Z"}]`, i)
			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux := http.NewServeMux()
			mux.HandleFunc("POST /events", s.handlePostEvents)
			mux.ServeHTTP(rec, req)

			var resp map[string]int
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Errorf("bad response body: %v", err)
				return
			}
			mu.Lock()
			accepted += resp["accepted"]
			duplicates += resp["duplicates"]
			mu.Unlock()
		}(i)
	}
	wg.Wait()

	if accepted != 100 || duplicates != 0 {
		t.Fatalf("accepted=%d duplicates=%d, want 100 and 0", accepted, duplicates)
	}
	if c := s.stats["cmpA"]; c.Opened != 100 || c.UniqueOpens != 1 {
		t.Fatalf("want opened=100 unique=1, got %+v", c)
	}
}

func TestPostEventsConcurrentSameEventID(t *testing.T) {
	s := newStorage()
	const n = 100
	body := `[{"event_id":"evt_solo","campaign_id":"cmpA","contact_id":"ct_1","type":"sent","timestamp":"2026-08-10T06:01:00Z"}]`

	var wg sync.WaitGroup
	var mu sync.Mutex
	accepted, duplicates, failed := 0, 0, 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			mux := http.NewServeMux()
			mux.HandleFunc("POST /events", s.handlePostEvents)
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				mu.Lock()
				failed++
				mu.Unlock()
				return
			}
			var resp map[string]int
			if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
				t.Errorf("bad response body: %v", err)
				return
			}
			mu.Lock()
			accepted += resp["accepted"]
			duplicates += resp["duplicates"]
			mu.Unlock()
		}()
	}
	wg.Wait()

	if failed != 0 {
		t.Fatalf("%d requests failed", failed)
	}
	if accepted != 1 || duplicates != n-1 {
		t.Fatalf("accepted=%d duplicates=%d, want 1 and %d", accepted, duplicates, n-1)
	}
	c := s.stats["cmpA"]
	if c == nil || c.Sent != 1 {
		t.Fatalf("event counted wrong number of times: %+v", c)
	}
}

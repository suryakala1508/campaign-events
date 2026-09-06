package main

import (
	"sync"
	"time"
)

// CampaignStats holds the counters for a single campaign.
type CampaignStats struct {
	Sent        int
	Delivered   int
	Opened      int
	Clicked     int
	UniqueOpens int
}

// ingestResult says how one event in a batch was handled.
type ingestResult int

const (
	ingestAccepted ingestResult = iota
	ingestDuplicate
	ingestRejected
)

// storage is the in-memory event store. All fields are guarded by mu.
type storage struct {
	mu             sync.Mutex
	seen           map[string]bool            // event IDs that have already been accepted
	stats          map[string]*CampaignStats  // campaignID -> stats
	opens          map[string]map[string]bool // campaignID -> set of contact IDs that opened
	deliveredByDay map[string]map[string]int  // campaignID -> UTC date (YYYY-MM-DD) -> delivered count
}

func newStorage() *storage {
	return &storage{
		seen:           make(map[string]bool),
		stats:          make(map[string]*CampaignStats),
		opens:          make(map[string]map[string]bool),
		deliveredByDay: make(map[string]map[string]int),
	}
}

// ingest validates and records a single event, returning how it was handled.
func (s *storage) ingest(ev Event) ingestResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !validEvent(ev) {
		return ingestRejected
	}
	if s.seen[ev.EventID] {
		return ingestDuplicate
	}
	s.seen[ev.EventID] = true

	c := s.stats[ev.CampaignID]
	if c == nil {
		c = &CampaignStats{}
		s.stats[ev.CampaignID] = c
	}

	switch ev.Type {
	case "sent":
		c.Sent++
	case "delivered":
		c.Delivered++
		if t, err := time.Parse(time.RFC3339, ev.Timestamp); err == nil {
			date := t.UTC().Format(time.DateOnly)
			days := s.deliveredByDay[ev.CampaignID]
			if days == nil {
				days = make(map[string]int)
				s.deliveredByDay[ev.CampaignID] = days
			}
			days[date]++
		}
	case "opened":
		c.Opened++
		contacts := s.opens[ev.CampaignID]
		if contacts == nil {
			contacts = make(map[string]bool)
			s.opens[ev.CampaignID] = contacts
		}
		if !contacts[ev.ContactID] {
			contacts[ev.ContactID] = true
			c.UniqueOpens++
		}
	case "clicked":
		c.Clicked++
	}
	return ingestAccepted
}

func validEvent(ev Event) bool {
	switch ev.Type {
	case "sent", "delivered", "opened", "clicked":
	default:
		return false
	}
	return ev.EventID != "" &&
		ev.CampaignID != "" &&
		ev.ContactID != "" &&
		validTimestamp(ev.Timestamp)
}

func validTimestamp(ts string) bool {
	_, err := time.Parse(time.RFC3339, ts)
	return err == nil
}

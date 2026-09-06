package main

// Event is one webhook event from a message provider.
type Event struct {
	EventID    string            `json:"event_id"`
	CampaignID string            `json:"campaign_id"`
	ContactID  string            `json:"contact_id"`
	Type       string            `json:"type"` // sent | delivered | opened | clicked
	Timestamp  string            `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// CampaignStatsResponse is the response for GET /campaigns/{campaignID}/stats.
type CampaignStatsResponse struct {
	CampaignID     string           `json:"campaign_id"`
	Sent           int              `json:"sent"`
	Delivered      int              `json:"delivered"`
	Opened         int              `json:"opened"`
	Clicked        int              `json:"clicked"`
	UniqueOpens    int              `json:"unique_opens"`
	DailyDelivered []DailyDelivered `json:"daily_delivered"`
}

// DailyDelivered is one UTC-date bucket of delivered events.
type DailyDelivered struct {
	Date  string `json:"date"` // YYYY-MM-DD in UTC
	Count int    `json:"count"`
}

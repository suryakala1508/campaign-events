## Test
Run the tests with:
    go test ./...
The expected result is that all tests pass.



## Race Detector
If CGO and GCC are configures :
      go run -race .
The race detector should complete without any `DATA RACE` warnnigs.



# campaign-events starter

A minimal skeleton. You can change anything in it — the routing, the types,
the layout. It exists so you don't spend your first twenty minutes on
boilerplate.

## Requirements

- Go 1.22 or later (the `"POST /events"` routing pattern needs it).
  Check with `go version`. On older Go, replace the patterns with manual
  method/path handling — that's fine too.
- No third-party dependencies are required. You may add them
  (e.g. an SQLite driver) if you want; keep `go.mod` tidy.

## Run

    go run .

Then, in another terminal:

    curl -s -X POST localhost:8080/events \
      -H 'Content-Type: application/json' \
      -d '[{"event_id":"evt_1","campaign_id":"cmp_summer_sale","contact_id":"ct_001","type":"delivered","timestamp":"2026-08-10T06:15:00Z"}]'

    curl -s localhost:8080/campaigns/cmp_summer_sale/stats

## Seed data

`seed/events.json` is one day of realistic provider traffic: 235 lines of events
including retries, conflicts, and malformed records. Your service should
survive all of it. Surviving it is not the same as accepting all of it — what
you accept, reject, and normalise is yours to decide and document.

One practical note: `[]Event` with strict field types means one bad element
can fail the whole array decode. Whether that is acceptable behaviour for a
batch endpoint is one of the decisions we are interested in.

To load the seed:

    curl -s -X POST localhost:8080/events \
      -H 'Content-Type: application/json' \
      --data-binary @seed/events.json

## Ingestion behavior (`POST /events`)

- `Content-Type` must identify the body as JSON: it is parsed with
  `mime.ParseMediaType` and accepted when the media type is `application/json`,
  so a header like `application/json; charset=utf-8` is fine. A missing or
  non-JSON `Content-Type` is rejected with `400`.
- The body must be a JSON array of `Event`. Structurally malformed JSON (or an
  empty array) is rejected with `400`.
- Events are processed one at a time, so a single bad element does not discard
  the rest of the batch (a deliberate choice: the seed data mixes valid and
  malformed records, and a batch endpoint that rejects everything on one bad
  record is fragile).
- A valid event: non-empty `event_id`, `campaign_id`, `contact_id`, and
  `timestamp`; `type` must be one of `sent`, `delivered`, `opened`, `clicked`;
  `timestamp` must parse as RFC 3339.
- Invalid events are skipped and never recorded, so a corrected retry with the
  same `event_id` is still accepted. The response reports what happened, e.g.:

      {"received": 5, "accepted": 4, "duplicates": 1, "rejected": 0}

- `event_id` is the idempotency key: an already-accepted `event_id` is counted
  as a duplicate and never affects statistics again (across requests and within
  the same request). If the same `event_id` arrives with conflicting content —
  including content that names a *different* campaign — the first occurrence
  wins; a duplicate `event_id` can never modify another campaign's counters.
- `unique_opens` counts each distinct contact that opened a campaign, so the
  same contact opening the same campaign repeatedly counts once per campaign.
- State is in memory, guarded by a mutex, so concurrent requests are safe.

`GET /campaigns/{campaignID}/stats` returns aggregated statistics for one campaign:

    {
      "campaign_id": "cmp_summer_sale",
      "sent": 12,
      "delivered": 11,
      "opened": 9,
      "clicked": 4,
      "unique_opens": 8,
      "daily_delivered": [
        {"date": "2026-08-10", "count": 11}
      ]
    }

Behavior:

- The campaign ID comes from the path (`r.PathValue("campaignID")`). An empty
  ID is rejected with `400`.
- `sent`, `delivered`, `opened`, and `clicked` count the accepted events of
  that type; `unique_opens` counts distinct contacts that opened the campaign.
- `daily_delivered` bucketing uses each event timestamp's **UTC date**
  (`YYYY-MM-DD`, computed via `time.Parse(RFC3339)` then `.UTC()`), so a
  timestamp like `2026-08-10T23:30:00Z` belongs to `2026-08-10` and an offset
  timestamp is converted to UTC first. `time.Local()` is never used.
- `daily_delivered` is returned as a *sorted, ascending* array by date, so the
  output is deterministic and does not depend on Go map iteration order.
- Unknown campaigns return `404` with `{"error":"campaign not found"}`. Status
  `404` (rather than `200` with zeroed stats) is chosen because the requested
  resource does not exist; it also keeps the case "campaign exists but has no
  events" distinguishable, should that ever arise.
- The endpoint is read-only and, like ingestion, is guarded by the storage
  mutex, so it is safe under concurrent requests. No `/campaigns/{id}/events`
  endpoint exists.

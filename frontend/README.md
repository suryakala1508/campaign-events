# Campaign Events Dashboard (frontend)

A tiny React + Vite dashboard that reads statistics from the Go backend
(`GET /campaigns/{campaignID}/stats`).

It is intentionally minimal: one page, one API call, no routing, no state
management library, no CSS framework.

## Requirements

- Node.js 18 or later (and npm). Check with `node --version`.
- The Go backend must already be running on `http://localhost:8080`.

## How the frontend talks to the Go API

The dashboard calls a **relative** URL:

```
GET /campaigns/{campaignID}/stats
```

Taken from `src/api.js`:

```js
fetch(`/campaigns/${encodeURIComponent(campaignId)}/stats`)
```

The browser resolves that relative URL against the page origin
(`http://localhost:5173`), so the request goes to the **Vite dev server**.
A small proxy rule in `vite.config.js` forwards any `/campaigns/*` request to
the Go backend on `http://localhost:8080`:

```js
proxy: {
  "/campaigns": {
    target: "http://localhost:8080",
    changeOrigin: true
  }
}
```

This is why no CORS headers were added to the Go server: the browser only ever
sees same-origin requests (5173 -> 5173 via the proxy), so CORS never applies.

## Install dependencies

```bash
npm install
```

## Start the backend first

In another terminal, from the `starter/` backend directory:

```bash
go run .
```

The backend prints `listening on :8080`.

To get some data into the in-memory store (optional but useful), load the seed
file once the backend is running:

```bash
curl -s -X POST localhost:8080/events \
  -H 'Content-Type: application/json' \
  --data-binary @seed/events.json
```

## Start the frontend

```bash
npm run dev
```

Open the dashboard at:

```
http://localhost:5173
```

The page loads `cmp_summer_sale` automatically. Type any campaign ID and press
**Load Stats** to refresh.

## Production build (optional)

```bash
npm run build
```

The build goes to `frontend/dist/`. Note: the dev proxy only exists in the Vite
dev server, so a static build served on its own will not proxy API calls. The
`npm run dev` flow above is the intended way to run this dashboard.

## What you should see

- With the backend running and seeded: four cards (Sent, Delivered, Opened,
  Clicked), a Unique Opens card, and a Daily Delivered table.
- Loading state while the request is in flight.
- `Campaign "xxx" not found.` when the backend returns 404.
- `Could not reach the backend...` when the Go server is not running.
- `Please enter a campaign ID.` when the input is empty (no request is sent).
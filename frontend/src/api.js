// api.js is the only file that talks to the Go backend.
//
// The URL below is *relative* ("/campaigns/..."), so the browser sends it to
// the Vite dev server on localhost:5173. The Vite proxy configured in
// vite.config.js then forwards it to the Go backend on localhost:8080.
// Because the browser only ever talks to localhost:5173, no CORS headers are
// needed on the Go side.

export async function fetchCampaignStats(campaignId) {
  let response

  try {
    response = await fetch(`/campaigns/${encodeURIComponent(campaignId)}/stats`)
  } catch {
    // fetch() throws (a TypeError) when the network request itself fails,
    // for example when the Go backend is not running.
    throw new Error(
      "Could not reach the backend. Is the Go server running on http://localhost:8080?"
    )
  }

  if (response.status === 404) {
    throw new Error(`Campaign "${campaignId}" not found.`)
  }

  if (!response.ok) {
    throw new Error(`The request failed (HTTP ${response.status}).`)
  }

  return await response.json()
}
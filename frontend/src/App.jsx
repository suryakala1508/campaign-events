import { useState } from "react"
import { fetchCampaignStats } from "./api.js"

const CAMPAIGNS = [
  { id: "cmp_summer_sale", label: "Summer Sale" },
  { id: "cmp_winback", label: "Winback" },
  { id: "cmp_welcome", label: "Welcome" },
]

const DEFAULT_CAMPAIGN_ID = CAMPAIGNS[0].id

function App() {
  const [campaignId, setCampaignId] = useState(DEFAULT_CAMPAIGN_ID)
  const [stats, setStats] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const loadStats = async (id) => {
    const trimmed = id.trim()

    if (trimmed === "") {
      setError("Please enter a campaign ID.")
      setStats(null)
      return
    }

    setLoading(true)
    setError(null)

    try {
      const data = await fetchCampaignStats(trimmed)
      setStats(data)
    } catch (err) {
      setStats(null)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = (event) => {
    event.preventDefault()
    loadStats(campaignId)
  }

  const hasStats = stats !== null
  const showEmptyState = !stats && !loading && !error
  const showBadge = stats && stats.campaign_id === "cmp_summer_sale"

  return (
    <div className="app">
      <Header loading={loading} />

      <CampaignSelector
        campaignId={campaignId}
        onChange={setCampaignId}
        onSubmit={handleSubmit}
        disabled={loading}
      />

      {loading && !hasStats && <LoadingPanel />}

      {error && !loading && <ErrorAlert message={error} />}

      {showEmptyState && <EmptyState />}

      {hasStats && (
        <div className={loading ? "content content-overlay" : "content"}>
          {loading && (
            <div className="loading-panel">
              <span className="spinner" />
              Loading campaign statistics…
            </div>
          )}

          <CampaignSummary campaignId={stats.campaign_id} showBadge={showBadge} />

          <MetricGrid stats={stats} />

          <UniqueOpens stats={stats} />

          <DailyDelivered stats={stats} />
        </div>
      )}

      <Footer />
    </div>
  )
}

function Header({ loading }) {
  return (
    <header className="dashboard-header">
      <div>
        <h1 className="brand-title">
          <span className="brand-name">RELAY</span>
          <span>Campaign Analytics</span>
        </h1>
        <p className="header-subtitle">Campaign performance overview</p>
      </div>
      <span className={loading ? "header-status loading" : "header-status"}>
        <span className="status-dot" />
        {loading ? "Refreshing…" : "Data loaded"}
      </span>
    </header>
  )
}

function CampaignSelector({ campaignId, onChange, onSubmit, disabled }) {
  return (
    <form className="campaign-card" onSubmit={onSubmit}>
      <label className="selector-label" htmlFor="campaign-input">
        Campaign
      </label>
      <div className="campaign-controls">
        <select
          id="campaign-input"
          className="campaign-input"
          value={campaignId}
          onChange={(event) => onChange(event.target.value)}
          disabled={disabled}
        >
          {CAMPAIGNS.map((campaign) => (
            <option key={campaign.id} value={campaign.id}>
              {campaign.label}
            </option>
          ))}
        </select>
        <button className="load-button" type="submit" disabled={disabled}>
          {disabled ? "Loading…" : "Load stats"}
        </button>
      </div>
    </form>
  )
}

function LoadingPanel() {
  return (
    <div className="loading-panel standalone">
      <span className="spinner" />
      Loading campaign statistics…
    </div>
  )
}

function EmptyState() {
  return (
    <div className="alert alert-info">
      <span className="alert-round">i</span>
      <div>
        <p className="alert-message" style={{ margin: 0 }}>
          No campaign loaded yet
        </p>
        <p className="alert-hint" style={{ margin: 0 }}>
          Select a campaign above to view its performance.
        </p>
      </div>
    </div>
  )
}

function ErrorAlert({ message }) {
  const isNotFound = message.toLowerCase().includes("not found")
  const isEmptyId = message === "Please enter a campaign ID."

  const title = isNotFound
    ? "Campaign not found"
    : isEmptyId
      ? "Enter a campaign ID"
      : "Unable to load campaign statistics"

  const hint = isEmptyId
    ? "Type a campaign ID and press Load stats to see its performance."
    : !isNotFound
      ? "Make sure the backend is running and try again."
      : "Check the campaign ID and try again."

  return (
    <div className="alert alert-error">
      <span className="alert-round">!</span>
      <div>
        <p className="alert-message" style={{ margin: 0 }}>
          {title}
        </p>
        <p className="alert-hint" style={{ margin: 0 }}>
          {hint}
        </p>
      </div>
    </div>
  )
}

function CampaignSummary({ campaignId, showBadge }) {
  return (
    <div className="campaign-summary">
      <h2>Campaign performance</h2>
      <span className="campaign-id">{campaignId}</span>
      {showBadge && <span className="campaign-badge">Active campaign</span>}
    </div>
  )
}

function MetricGrid({ stats }) {
  const metricCards = [
    { label: "Sent", value: stats.sent, note: "Messages sent", tone: "sent" },
    {
      label: "Delivered",
      value: stats.delivered,
      note: percent(stats.delivered, stats.sent) + " of sent",
      tone: "delivered",
    },
    {
      label: "Opened",
      value: stats.opened,
      note: percent(stats.opened, stats.delivered) + " of delivered",
      tone: "opened",
    },
    {
      label: "Clicked",
      value: stats.clicked,
      note: percent(stats.clicked, stats.opened) + " of opened",
      tone: "clicked",
    },
  ]

  return (
    <div className="metric-grid">
      {metricCards.map((card) => (
        <MetricCard key={card.label} {...card} />
      ))}
    </div>
  )
}

function MetricCard({ label, value, note, tone }) {
  return (
    <div className={`metric-card metric-${tone}`}>
      <span className="metric-label">{label}</span>
      <span className="metric-value">{value}</span>
      <span className="metric-note">{note}</span>
    </div>
  )
}

function UniqueOpens({ stats }) {
  return (
    <div className="unique-opens-card">
      <div className="unique-opens-text">
        <h3 className="section-title">Unique Opens</h3>
        <p className="section-subtitle" style={{ marginBottom: 0 }}>
          Distinct contacts who opened this campaign
        </p>
      </div>
      <span className="unique-opens-value">{stats.unique_opens}</span>
    </div>
  )
}

function DailyDelivered({ stats }) {
  const days = stats.daily_delivered || []
  const total = days.reduce((sum, day) => sum + day.count, 0)
  const max = Math.max(...days.map((day) => day.count), 0)
  const chartHeight = Math.max(64, Math.min(96, 48 + days.length * 16))

  return (
    <section className="daily-card">
      <h3 className="section-title">Daily Delivered</h3>
      <p className="section-subtitle">Delivered events by UTC date</p>

      {days.length === 0 ? (
        <p className="table-empty">No delivered events recorded for this campaign.</p>
      ) : (
        <>
          <BarChart days={days} max={max} height={chartHeight} />
          <table className="daily-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Delivered</th>
              </tr>
            </thead>
            <tbody>
              {days.map((day) => (
                <tr key={day.date}>
                  <td>{day.date}</td>
                  <td>{day.count}</td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr>
                <td>Total</td>
                <td>{total}</td>
              </tr>
            </tfoot>
          </table>
        </>
      )}
    </section>
  )
}

function BarChart({ days, max, height }) {
  const barArea = Math.max(4, height - 50)

  return (
    <div className="bar-chart" style={{ height }} aria-label="Daily delivered bar chart">
      {days.map((day) => (
        <div className="bar-col" key={day.date} title={`${day.date}: ${day.count} delivered`}>
          <span className="bar-value">{day.count}</span>
          <div
            className="bar"
            style={{ height: max > 0 ? Math.max(3, Math.round((day.count / max) * barArea)) : 3 }}
          />
          <span className="bar-label">{day.date.slice(5)}</span>
        </div>
      ))}
    </div>
  )
}

function Footer() {
  return (
    <footer className="dashboard-footer">
      Relay Campaign Analytics — internal tool
    </footer>
  )
}

function percent(part, whole) {
  if (!Number.isFinite(part) || !Number.isFinite(whole) || whole <= 0) {
    return "—"
  }
  return Math.round((part / whole) * 1000) / 10 + "%"
}

export default App
import { useState } from 'react'
import type { SubmissionAnalytics, Widget } from './lib/api'

interface Props {
  analytics: SubmissionAnalytics | null
  widgets: Widget[]
  loading: boolean
  error: string
  onLoadAnalytics: (widgetId?: string) => Promise<SubmissionAnalytics>
}

function countLabel(value: number | undefined): string {
  return typeof value === 'number' ? value.toLocaleString() : '—'
}

function chartDayLabel(value: string): string {
  const date = new Date(`${value.slice(0, 10)}T12:00:00`)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat('en-US', {
    weekday: 'short',
  }).format(date)
}

export default function AnalyticsPage({
  analytics,
  widgets,
  loading,
  error,
  onLoadAnalytics,
}: Props) {
  const [widgetId, setWidgetId] = useState('all')
  const [widgetAnalytics, setWidgetAnalytics] =
    useState<SubmissionAnalytics | null>(null)
  const [workspaceAnalytics, setWorkspaceAnalytics] =
    useState<SubmissionAnalytics | null>(null)
  const [requestLoading, setRequestLoading] = useState(false)
  const [requestError, setRequestError] = useState('')

  const shownAnalytics = widgetId === 'all'
    ? (workspaceAnalytics ?? analytics)
    : widgetAnalytics
  const chartDays = (shownAnalytics?.daily ?? []).slice(-7)
  const maximum = Math.max(1, ...chartDays.map((day) => day.count))
  const selectedWidget = widgets.find((widget) => widget.id === widgetId)

  async function selectWidget(nextId: string) {
    setWidgetId(nextId)
    setWidgetAnalytics(null)
    setRequestError('')

    if (nextId === 'all') return

    setRequestLoading(true)
    try {
      const result = await onLoadAnalytics(nextId)
      setWidgetAnalytics(result)
    } catch (caught) {
      setRequestError(
        caught instanceof Error ? caught.message : 'Could not load widget analytics.',
      )
    } finally {
      setRequestLoading(false)
    }
  }

  async function refreshAnalytics() {
    setRequestError('')
    setRequestLoading(true)

    try {
      const result = await onLoadAnalytics(
        widgetId === 'all' ? undefined : widgetId,
      )
      if (widgetId !== 'all') {
        setWidgetAnalytics(result)
      } else {
        setWorkspaceAnalytics(result)
      }
    } catch (caught) {
      setRequestError(
        caught instanceof Error ? caught.message : 'Could not refresh analytics.',
      )
    } finally {
      setRequestLoading(false)
    }
  }

  const busy = loading || requestLoading
  const visibleError = requestError || error

  return (
    <div className="analytics-workspace">
      <section className="analytics-page-heading reveal">
        <div>
          <div className="eyebrow">
            <span className="eyebrow-dot" /> WORKSPACE INSIGHTS
          </div>
          <h1>Make sense <span>of your leads.</span></h1>
          <p className="page-subtitle">
            Understand submission volume and activity across your widgets.
          </p>
        </div>

        <div className="analytics-controls">
          <label className="analytics-widget-filter">
            <span>Widget</span>
            <select
              aria-label="Filter analytics by widget"
              value={widgetId}
              disabled={requestLoading}
              onChange={(event) => { void selectWidget(event.target.value) }}
            >
              <option value="all">All widgets</option>
              {widgets.map((widget) => (
                <option key={widget.id} value={widget.id}>
                  {widget.title}
                </option>
              ))}
            </select>
          </label>

          <button
            type="button"
            className="quiet-button"
            disabled={busy}
            onClick={() => { void refreshAnalytics() }}
          >
            {busy ? 'Updating…' : 'Refresh analytics'}
          </button>
        </div>
      </section>

      {visibleError && (
        <div className="data-alert" role="alert">{visibleError}</div>
      )}

      <section className="analytics-metrics" aria-label="Submission analytics">
        <article className="analytics-metric-card">
          <span>Total submissions</span>
          <strong>{busy && !shownAnalytics ? '…' : countLabel(shownAnalytics?.total_submissions)}</strong>
          <small>{selectedWidget ? selectedWidget.title : 'Across this workspace'}</small>
        </article>

        <article className="analytics-metric-card">
          <span>Last 7 days</span>
          <strong>{busy && !shownAnalytics ? '…' : countLabel(shownAnalytics?.submissions_last_7_days)}</strong>
          <small>Recent submission activity</small>
        </article>

        <article className="analytics-metric-card">
          <span>Last 30 days</span>
          <strong>{busy && !shownAnalytics ? '…' : countLabel(shownAnalytics?.submissions_last_30_days)}</strong>
          <small>Monthly activity</small>
        </article>

        <article className="analytics-metric-card">
          <span>Widgets</span>
          <strong>{widgets.length.toLocaleString()}</strong>
          <small>In this workspace</small>
        </article>
      </section>

      <section className="panel analytics-chart-panel reveal">
        <div className="panel-header">
          <div>
            <h2>Submission activity</h2>
            <p>
              {selectedWidget
                ? `Daily activity for ${selectedWidget.title}.`
                : 'Daily activity across all your widgets.'}
            </p>
          </div>
          <span className="analytics-period-label">Last 7 available days</span>
        </div>

        {busy && !shownAnalytics ? (
          <div className="analytics-empty">
            <strong>Loading analytics…</strong>
            <p>Fetching the latest figures from your API.</p>
          </div>
        ) : chartDays.length > 0 ? (
          <div className="analytics-chart" aria-label="Submissions by day">
            {chartDays.map((day) => (
              <div
                className="analytics-chart-column"
                key={day.date}
                title={`${day.count} submissions on ${day.date}`}
              >
                <span className="analytics-chart-count">{day.count}</span>
                <div className="analytics-chart-track">
                  <span
                    className="analytics-chart-bar"
                    style={{
                      height: `${day.count > 0
                        ? Math.max(6, (day.count / maximum) * 100)
                        : 0}%`,
                    }}
                  />
                </div>
                <span className="analytics-chart-day">{chartDayLabel(day.date)}</span>
              </div>
            ))}
          </div>
        ) : (
          <div className="analytics-empty">
            <strong>{visibleError ? 'Analytics unavailable' : 'No activity to display yet'}</strong>
            <p>
              {visibleError
                ? 'Check the API connection and refresh analytics.'
                : 'Submission activity will appear here when visitors send your forms.'}
            </p>
          </div>
        )}

        <div className="analytics-chart-footer">
          <span><i aria-hidden="true" /> Submissions per day</span>
          <span>{busy ? 'Updating…' : 'Live API data'}</span>
        </div>
      </section>

      <section className="panel analytics-insight-panel reveal">
        <div className="panel-header">
          <div>
            <h2>How to read these numbers</h2>
            <p>These figures come from your authenticated analytics endpoint.</p>
          </div>
        </div>
        <div className="analytics-insight-grid">
          <div>
            <strong>All-time submissions</strong>
            <p>The total recorded submissions for the selected scope.</p>
          </div>
          <div>
            <strong>Recent activity</strong>
            <p>Compare the seven-day and 30-day counts to understand recent lead volume.</p>
          </div>
        </div>
      </section>
    </div>
  )
}

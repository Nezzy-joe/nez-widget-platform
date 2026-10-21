import { useMemo, useState } from 'react'
import type { Submission, Widget } from './lib/api'

interface Props {
  submissions: Submission[]
  widgets: Widget[]
  loading: boolean
  error: string
  onRefresh: () => Promise<void>
}

function textValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value) ?? ''
  return String(value)
}

function leadInfo(submission: Submission, index: number) {
  const entries = Object.entries(submission.data ?? {})
  const normalise = (key: string) =>
    key.toLowerCase().replace(/[^a-z0-9]/g, '')
  const find = (keys: string[]) =>
    entries.find(([key, value]) =>
      keys.includes(normalise(key)) &&
      value !== null &&
      value !== undefined &&
      textValue(value).trim() !== '',
    )?.[1]

  const name = textValue(find(['name', 'fullname', 'contactname'])) ||
    [
      textValue(find(['firstname'])),
      textValue(find(['lastname'])),
    ].filter(Boolean).join(' ') ||
    `Submission ${index + 1}`

  const contact = textValue(find(['email', 'emailaddress'])) ||
    textValue(find(['phone', 'phonenumber', 'telephone', 'tel'])) ||
    'Form response'

  return { name, contact }
}

function dateLabel(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Date unavailable'

  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(date)
}

export default function SubmissionsPage({
  submissions,
  widgets,
  loading,
  error,
  onRefresh,
}: Props) {
  const [query, setQuery] = useState('')
  const [widgetId, setWidgetId] = useState('all')
  const [page, setPage] = useState(1)
  const [refreshing, setRefreshing] = useState(false)
  const [refreshError, setRefreshError] = useState('')
  const pageSize = 10

  const widgetNames = useMemo(
    () => new Map(widgets.map((widget) => [widget.id, widget.title])),
    [widgets],
  )

  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase()

    return submissions.filter((submission) => {
      if (widgetId !== 'all' && submission.widget_id !== widgetId) {
        return false
      }

      if (!term) return true

      const widgetName = widgetNames.get(submission.widget_id) ?? ''
      const searchable = [
        submission.id,
        submission.created_at,
        submission.city ?? '',
        submission.country ?? '',
        widgetName,
        JSON.stringify(submission.data ?? {}),
      ].join(' ').toLowerCase()

      return searchable.includes(term)
    })
  }, [submissions, widgetId, query, widgetNames])

  const totalPages = Math.max(1, Math.ceil(filtered.length / pageSize))
  const currentPage = Math.min(page, totalPages)
  const visible = filtered.slice(
    (currentPage - 1) * pageSize,
    currentPage * pageSize,
  )

  async function refresh() {
    setRefreshing(true)
    setRefreshError('')

    try {
      await onRefresh()
    } catch (caught) {
      setRefreshError(
        caught instanceof Error ? caught.message : 'Could not refresh submissions.',
      )
    } finally {
      setRefreshing(false)
    }
  }

  function exportCsv() {
    const quote = (value: unknown) =>
      `"${String(value ?? '').replace(/"/g, '""')}"`

    const rows = [
      ['Submitted at', 'Lead', 'Contact', 'Widget', 'Location', 'Submission data'],
      ...filtered.map((submission, index) => {
        const lead = leadInfo(submission, index)

        return [
          submission.created_at,
          lead.name,
          lead.contact,
          widgetNames.get(submission.widget_id) ?? submission.widget_id,
          [submission.city, submission.country].filter(Boolean).join(', ') || 'Unavailable',
          JSON.stringify(submission.data ?? {}),
        ]
      }),
    ]

    const csv = rows.map((row) => row.map(quote).join(',')).join('\r\n')
    const url = URL.createObjectURL(
      new Blob(['\uFEFF', csv], { type: 'text/csv;charset=utf-8' }),
    )
    const link = document.createElement('a')
    link.href = url
    link.download = 'nez-widget-submissions.csv'
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="submissions-workspace">
      <section className="submissions-page-heading reveal">
        <div>
          <div className="eyebrow">
            <span className="eyebrow-dot" /> LEAD INBOX
          </div>
          <h1>Every lead, <span>in one place.</span></h1>
          <p className="page-subtitle">
            Review responses from visitors who submit your embedded widgets.
          </p>
        </div>

        <button
          type="button"
          className="quiet-button"
          disabled={refreshing}
          onClick={() => { void refresh() }}
        >
          {refreshing ? 'Refreshing…' : 'Refresh submissions'}
        </button>
      </section>

      {(error || refreshError) && (
        <div className="data-alert" role="alert">
          {refreshError || error}
        </div>
      )}

      <section className="submission-stat-row">
        <article className="submission-stat-card">
          <span>Loaded submissions</span>
          <strong>{submissions.length}</strong>
          <small>Latest records returned by the API</small>
        </article>
        <article className="submission-stat-card">
          <span>Matching results</span>
          <strong>{filtered.length}</strong>
          <small>After current filters</small>
        </article>
        <article className="submission-stat-card">
          <span>Workspace widgets</span>
          <strong>{widgets.length}</strong>
          <small>Available for filtering</small>
        </article>
      </section>

      <section className="panel submissions-panel reveal">
        <div className="submissions-toolbar">
          <label className="submission-search">
            <span aria-hidden="true">⌕</span>
            <input
              type="search"
              value={query}
              placeholder="Search leads or submitted data"
              aria-label="Search submissions"
              onChange={(event) => {
                setQuery(event.target.value)
                setPage(1)
              }}
            />
          </label>

          <label className="submission-widget-filter">
            <span>Widget</span>
            <select
              value={widgetId}
              aria-label="Filter submissions by widget"
              onChange={(event) => {
                setWidgetId(event.target.value)
                setPage(1)
              }}
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
            className="quiet-button submission-export-button"
            disabled={filtered.length === 0}
            onClick={exportCsv}
          >
            Export CSV
          </button>
        </div>

        <p className="submission-results-note">
          {loading
            ? 'Updating submissions…'
            : 'Search and export cover the records currently loaded from the API.'}
        </p>

        {loading && submissions.length === 0 ? (
          <div className="submissions-empty">
            <strong>Loading submissions…</strong>
            <p>Fetching records for your workspace.</p>
          </div>
        ) : visible.length === 0 ? (
          <div className="submissions-empty">
            <strong>
              {submissions.length === 0
                ? 'No submissions yet'
                : 'No matching submissions'}
            </strong>
            <p>
              {submissions.length === 0
                ? 'New visitor responses will appear here.'
                : 'Try another search or select a different widget.'}
            </p>
            {(query || widgetId !== 'all') && (
              <button
                type="button"
                className="quiet-button"
                onClick={() => {
                  setQuery('')
                  setWidgetId('all')
                  setPage(1)
                }}
              >
                Clear filters
              </button>
            )}
          </div>
        ) : (
          <div className="submission-list">
            {visible.map((submission) => {
              const index = submissions.findIndex(
                (item) => item.id === submission.id,
              )
              const lead = leadInfo(submission, index)
              const location = [
                submission.city,
                submission.country,
              ].filter(Boolean).join(', ')

              return (
                <article className="submission-record" key={submission.id}>
                  <div className="submission-record-avatar">
                    {lead.name.charAt(0).toUpperCase()}
                  </div>

                  <div className="submission-record-main">
                    <div className="submission-record-heading">
                      <strong>{lead.name}</strong>
                      <time dateTime={submission.created_at}>
                        {dateLabel(submission.created_at)}
                      </time>
                    </div>

                    <div className="submission-record-meta">
                      <span>{lead.contact}</span>
                      <span>
                        {widgetNames.get(submission.widget_id) ?? 'Widget unavailable'}
                      </span>
                      <span>{location || 'Location unavailable'}</span>
                    </div>

                    <details className="submission-record-details">
                      <summary>View submitted fields</summary>
                      <dl>
                        {Object.entries(submission.data ?? {}).map(([key, value]) => (
                          <div className="submission-detail-row" key={key}>
                            <dt>{key.replace(/[_-]+/g, ' ')}</dt>
                            <dd>{textValue(value) || '—'}</dd>
                          </div>
                        ))}
                      </dl>
                    </details>
                  </div>
                </article>
              )
            })}
          </div>
        )}

        {filtered.length > 0 && (
          <footer className="submissions-pagination">
            <span>
              Showing {(currentPage - 1) * pageSize + 1}–
              {Math.min(currentPage * pageSize, filtered.length)} of {filtered.length}
            </span>

            <div>
              <button
                type="button"
                className="quiet-button"
                disabled={currentPage <= 1}
                onClick={() => setPage((value) => Math.max(1, value - 1))}
              >
                Previous
              </button>
              <span>Page {currentPage} of {totalPages}</span>
              <button
                type="button"
                className="quiet-button"
                disabled={currentPage >= totalPages}
                onClick={() => setPage((value) => Math.min(totalPages, value + 1))}
              >
                Next
              </button>
            </div>
          </footer>
        )}
      </section>
    </div>
  )
}

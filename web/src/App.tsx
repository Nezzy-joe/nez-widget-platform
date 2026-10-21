import { useEffect, useRef, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { gsap } from 'gsap'
import { useGSAP } from '@gsap/react'
import Lenis from 'lenis'
import {
  ApiError,
  clearAccessToken,
  getAccessToken,
  getMe,
  createWidget,
  deleteWidget,
  updateWidget,
  getAnalytics,
  getSubmissions,
  getWidgets,
  login,
  register,
} from './lib/api'
import type { CreateWidgetInput, Submission, SubmissionAnalytics, User, Widget } from './lib/api'
import SubmissionsPage from './SubmissionsPage'
import AnalyticsPage from './AnalyticsPage'
import './App.css'

type PageId = 'Overview' | 'Widgets' | 'Submissions' | 'Analytics' | 'Settings'
type Theme = 'light' | 'dark'

type IconName =
  | 'overview'
  | 'widgets'
  | 'submissions'
  | 'analytics'
  | 'settings'
  | 'plus'
  | 'bell'
  | 'menu'
  | 'search'
  | 'arrow'
  | 'sparkles'
  | 'check'
  | 'clock'
  | 'external'
  | 'layers'
  | 'shield'
  | 'close'
  | 'logout'
  | 'sun'
  | 'moon'

const iconArtwork: Record<IconName, ReactNode> = {
  overview: (
    <>
      <rect x="3" y="3" width="7" height="7" rx="2" />
      <rect x="14" y="3" width="7" height="7" rx="2" />
      <rect x="3" y="14" width="7" height="7" rx="2" />
      <rect x="14" y="14" width="7" height="7" rx="2" />
    </>
  ),
  widgets: (
    <>
      <rect x="3" y="3" width="18" height="18" rx="4" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </>
  ),
  submissions: (
    <>
      <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3h11A2.5 2.5 0 0 1 20 5.5v13a2.5 2.5 0 0 1-2.5 2.5h-11A2.5 2.5 0 0 1 4 18.5z" />
      <path d="M8 8h8M8 12h8M8 16h5" />
    </>
  ),
  analytics: (
    <>
      <path d="M3 3v18h18" />
      <path d="m7 14 4-4 3 3 6-7" />
      <path d="M15.5 6H20v4.5" />
    </>
  ),
  settings: (
    <>
      <circle cx="12" cy="12" r="3" />
      <path d="m19.4 15 .1.1 1.4 1.1-1.5 2.6-1.7-.6a8 8 0 0 1-1.5.9l-.3 1.9h-3l-.3-1.9a8 8 0 0 1-1.5-.9l-1.7.6-1.5-2.6 1.5-1.1a8 8 0 0 1 0-1.8L6.4 12l1.5-2.6 1.7.6a8 8 0 0 1 1.5-.9l.3-1.9h3l.3 1.9a8 8 0 0 1 1.5.9l1.7-.6 1.5 2.6-1.5 1.1a8 8 0 0 1 0 1.9z" transform="translate(-1 -1) scale(1.08)" />
    </>
  ),
  plus: <path d="M12 5v14M5 12h14" />,
  bell: (
    <>
      <path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" />
      <path d="M10 21h4" />
    </>
  ),
  menu: <path d="M4 7h16M4 12h16M4 17h16" />,
  search: (
    <>
      <circle cx="10.8" cy="10.8" r="6.8" />
      <path d="m16 16 5 5" />
    </>
  ),
  arrow: <path d="M5 12h14m-6-6 6 6-6 6" />,
  sparkles: (
    <>
      <path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" />
      <path d="m19 14 .9 2.1L22 17l-2.1.9L19 20l-.9-2.1L16 17l2.1-.9z" />
    </>
  ),
  check: <path d="m5 12 4 4L19 6" />,
  clock: (
    <>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </>
  ),
  external: (
    <>
      <path d="M14 4h6v6M20 4l-9 9" />
      <path d="M18 13v6a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1h6" />
    </>
  ),
  layers: (
    <>
      <path d="m12 3 9 5-9 5-9-5z" />
      <path d="m3 12 9 5 9-5M3 16l9 5 9-5" />
    </>
  ),
  shield: (
    <>
      <path d="M12 3 20 6v5c0 5-3.5 8-8 10-4.5-2-8-5-8-10V6z" />
      <path d="m9 12 2 2 4-4" />
    </>
  ),
  close: <path d="m6 6 12 12M18 6 6 18" />,
  logout: (
    <>
      <path d="M10 17l5-5-5-5M15 12H3" />
      <path d="M12 3h6a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2h-6" />
    </>
  ),
  sun: (
    <>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.93 4.93l1.42 1.42M17.65 17.65l1.42 1.42M2 12h2M20 12h2M4.93 19.07l1.42-1.42M17.65 6.35l1.42-1.42" />
    </>
  ),
  moon: <path d="M20.9 13A9 9 0 0 1 11 3.1 9 9 0 1 0 20.9 13Z" />,
}

function Icon({
  name,
  size = 18,
}: {
  name: IconName
  size?: number
}) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth={1.7}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {iconArtwork[name]}
    </svg>
  )
}

const navigation: { label: PageId; icon: IconName }[] = [
  { label: 'Overview', icon: 'overview' },
  { label: 'Widgets', icon: 'widgets' },
  { label: 'Submissions', icon: 'submissions' },
  { label: 'Analytics', icon: 'analytics' },
]

const pageDescriptions: Record<
  PageId,
  { subtitle: string; description: string }
> = {
  Overview: {
    subtitle: 'Your workspace, at a glance.',
    description: 'Everything you need to understand your lead capture activity.',
  },
  Widgets: {
    subtitle: 'Your forms, your way.',
    description: 'Create and manage the widgets you embed on your website.',
  },
  Submissions: {
    subtitle: 'Every lead, in one place.',
    description: 'Review the people who have submitted your forms.',
  },
  Analytics: {
    subtitle: 'Understand what is working.',
    description: 'Explore submission volume and activity across your widgets.',
  },
  Settings: {
    subtitle: 'Make this workspace yours.',
    description: 'Account and workspace preferences will live here.',
  },
}

function Sidebar({
  activePage,
  onNavigate,
  mobileOpen,
  onClose,
  userEmail,
  onLogout,
}: {
  activePage: PageId
  onNavigate: (page: PageId) => void
  mobileOpen: boolean
  onClose: () => void
  userEmail: string
  onLogout: () => void
}) {
  return (
    <>
      {mobileOpen && (
        <button
          type="button"
          className="sidebar-scrim"
          aria-label="Close navigation"
          onClick={onClose}
        />
      )}

      <aside className={`sidebar ${mobileOpen ? 'sidebar--open' : ''}`}>
        <div className="brand-row">
          <div className="brand-symbol" aria-hidden="true">
            <span />
            <span />
            <span />
          </div>
          <span className="brand-name">nez<span>widget</span></span>
          <button
            type="button"
            className="icon-button sidebar-close"
            aria-label="Close navigation"
            onClick={onClose}
          >
            <Icon name="close" />
          </button>
        </div>

        <button type="button" className="workspace-switcher">
          <span className="workspace-avatar">N</span>
          <span className="workspace-text">
            <strong>My workspace</strong>
            <small>Free plan</small>
          </span>
          <span className="workspace-chevron">⌄</span>
        </button>

        <div className="sidebar-section-label">WORKSPACE</div>
        <nav className="primary-nav" aria-label="Main navigation">
          {navigation.map((item) => (
            <button
              type="button"
              key={item.label}
              className={`nav-item ${activePage === item.label ? 'nav-item--active' : ''}`}
              aria-current={activePage === item.label ? 'page' : undefined}
              onClick={() => {
                onNavigate(item.label)
                onClose()
              }}
            >
              <Icon name={item.icon} size={18} />
              <span>{item.label}</span>
              {item.label === 'Submissions' && (
                <span className="nav-item-badge">Inbox</span>
              )}
            </button>
          ))}
        </nav>

        <div className="sidebar-section-label sidebar-section-label--lower">
          PREFERENCES
        </div>
        <nav className="primary-nav" aria-label="Preferences">
          <button
            type="button"
            className={`nav-item ${activePage === 'Settings' ? 'nav-item--active' : ''}`}
            aria-current={activePage === 'Settings' ? 'page' : undefined}
            onClick={() => {
              onNavigate('Settings')
              onClose()
            }}
          >
            <Icon name="settings" size={18} />
            <span>Settings</span>
          </button>
        </nav>

        <div className="sidebar-spacer" />

        <div className="sidebar-help-card">
          <div className="help-icon"><Icon name="sparkles" size={17} /></div>
          <strong>A little help goes a long way.</strong>
          <p>Make the most of your lead capture setup.</p>
          <button type="button" onClick={() => onNavigate('Widgets')}>
            Explore widgets <Icon name="arrow" size={14} />
          </button>
        </div>

        <div className="sidebar-user">
          <div className="user-avatar">{userEmail.charAt(0).toUpperCase()}</div>
          <div className="sidebar-user-info">
            <strong title={userEmail}>{userEmail}</strong>
            <span>Signed in</span>
          </div>
          <button
            type="button"
            className="icon-button user-settings-button"
            aria-label="Sign out"
            title="Sign out"
            onClick={onLogout}
          >
            <Icon name="logout" size={17} />
          </button>
        </div>
      </aside>
    </>
  )
}

const dashboardDateLabel = new Intl.DateTimeFormat('en-US', {
  weekday: 'short',
  month: 'short',
  day: 'numeric',
}).format(new Date())

function Header({
  activePage,
  onCreateWidget,
  onToggleSidebar,
  userEmail,
  theme,
  onToggleTheme,
}: {
  activePage: PageId
  onCreateWidget: () => void
  onToggleSidebar: () => void
  userEmail: string
  theme: Theme
  onToggleTheme: () => void
}) {
  const [notificationsOpen, setNotificationsOpen] = useState(false)
  return (
    <header className="topbar">
      <div className="topbar-left">
        <button
          type="button"
          className="icon-button mobile-menu-button"
          aria-label="Open navigation"
          onClick={onToggleSidebar}
        >
          <Icon name="menu" size={20} />
        </button>
        <div className="breadcrumb">
          <span>Workspace</span>
          <span className="breadcrumb-slash">/</span>
          <strong>{activePage}</strong>
        </div>
      </div>

      <div className="topbar-right">
        <div className="date-label">
          <Icon name="clock" size={15} />
          <span>{dashboardDateLabel}</span>
        </div>
        <button
          type="button"
          className="icon-button theme-toggle"
          aria-label={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          aria-pressed={theme === 'dark'}
          title={theme === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}
          onClick={onToggleTheme}
        >
          <Icon name={theme === 'dark' ? 'sun' : 'moon'} size={18} />
        </button>
        <div className="notification-wrap">
          <button
            type="button"
            className={`icon-button notification-button ${notificationsOpen ? 'icon-button--selected' : ''}`}
            aria-label="Notifications"
            aria-expanded={notificationsOpen}
            onClick={() => setNotificationsOpen((open) => !open)}
          >
            <Icon name="bell" size={18} />
            <span className="notification-dot" />
          </button>
          {notificationsOpen && (
            <div className="notification-popover">
              <div className="notification-popover-title">Notifications</div>
              <div className="notification-empty">
                <span className="notification-empty-icon">
                  <Icon name="check" size={19} />
                </span>
                <strong>You're all caught up</strong>
                <p>New activity will show up here.</p>
              </div>
            </div>
          )}
        </div>
        <div className="topbar-divider" />
        <div className="topbar-avatar" title={userEmail}>
          {userEmail.charAt(0).toUpperCase()}
        </div>
        <button type="button" className="primary-button topbar-cta" onClick={onCreateWidget}>
          <Icon name="plus" size={17} />
          <span>New widget</span>
        </button>
      </div>
    </header>
  )
}

function MetricCard({
  label,
  value,
  icon,
  iconClass,
  helper,
}: {
  label: string
  value: string
  icon: IconName
  iconClass: string
  helper: string
}) {
  return (
    <article className="metric-card reveal">
      <div className="metric-card-top">
        <span className="metric-label">{label}</span>
        <span className={`metric-icon ${iconClass}`}>
          <Icon name={icon} size={18} />
        </span>
      </div>
      <div className="metric-value">{value}</div>
      <div className="metric-helper">
        <span className="metric-pending-dot" />
        {helper}
      </div>
    </article>
  )
}

function displayValue(value: unknown): string {
  if (value === null || value === undefined) return ''
  if (typeof value === 'object') return JSON.stringify(value) ?? ''
  return String(value)
}

function submissionIdentity(submission: Submission, index: number) {
  const entries = Object.entries(submission.data ?? {}).filter(
    ([, value]) => value !== null && value !== undefined &&
      displayValue(value).trim() !== '',
  )
  const normalize = (key: string) => key.toLowerCase().replace(/[^a-z0-9]/g, '')
  const findValue = (keys: string[]) =>
    entries.find(([key]) => keys.includes(normalize(key)))?.[1]

  const fullName = displayValue(findValue(['name', 'fullname', 'contactname']))
  const firstName = displayValue(findValue(['firstname']))
  const lastName = displayValue(findValue(['lastname']))
  const email = displayValue(findValue(['email', 'emailaddress']))
  const phone = displayValue(findValue(['phone', 'phonenumber', 'telephone', 'tel']))
  const fallback = entries.find(
    ([key]) => ![
      'email', 'emailaddress', 'phone', 'phonenumber', 'telephone',
      'tel', 'message', 'comment',
    ].includes(normalize(key)),
  )?.[1]

  return {
    name: fullName || [firstName, lastName].filter(Boolean).join(' ') ||
      displayValue(fallback) || `Submission ${index + 1}`,
    detail: email || phone || 'Form response',
  }
}

function formatSubmissionDate(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return 'Date unavailable'
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  }).format(date)
}

function formatChartDate(value: string): string {
  const date = new Date(`${value.slice(0, 10)}T12:00:00`)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat('en-US', { weekday: 'short' }).format(date)
}

function Overview({
  onNavigate,
  analytics,
  widgetCount,
  submissions,
  loading,
  error,
}: {
  onNavigate: (page: PageId) => void
  analytics: SubmissionAnalytics | null
  widgetCount: number
  submissions: Submission[]
  loading: boolean
  error: string
}) {
  const daily = (analytics?.daily ?? []).slice(-7)
  const maxDailyCount = Math.max(1, ...daily.map((day) => day.count))
  const recentSubmissions = submissions.slice(0, 5)
  return (
    <>
      <section className="page-heading reveal">
        <div>
          <div className="eyebrow">
            <span className="eyebrow-dot" /> WORKSPACE OVERVIEW
          </div>
          <h1>Your leads, <span>in perspective.</span></h1>
          <p className="page-subtitle">
            A clear view of your widgets, submissions, and what happens next.
          </p>
        </div>
        <button type="button" className="primary-button heading-cta" onClick={() => onNavigate('Widgets')}>
          <Icon name="plus" size={17} />
          Create a widget
        </button>
      </section>

      {error && <div className="data-alert" role="status">{error}</div>}

      <section className="metrics-grid" aria-label="Submission metrics">
        <MetricCard
          label="Total submissions"
          value={loading ? '…' : analytics ? analytics.total_submissions.toLocaleString() : '—'}
          icon="submissions"
          iconClass="metric-icon--blue"
          helper={loading ? 'Syncing live data' : error ? 'Unable to sync data' : 'All-time submissions'}
        />
        <MetricCard
          label="Last 7 days"
          value={loading ? '…' : analytics ? analytics.submissions_last_7_days.toLocaleString() : '—'}
          icon="analytics"
          iconClass="metric-icon--violet"
          helper="Recent lead activity"
        />
        <MetricCard
          label="Last 30 days"
          value={loading ? '…' : analytics ? analytics.submissions_last_30_days.toLocaleString() : '—'}
          icon="clock"
          iconClass="metric-icon--amber"
          helper="Recent lead activity"
        />
        <MetricCard
          label="Your widgets"
          value={loading ? '…' : error && !analytics ? '—' : widgetCount.toLocaleString()}
          icon="widgets"
          iconClass="metric-icon--green"
          helper="In this workspace"
        />
      </section>

      <section className="dashboard-panels">
        <article className="panel activity-panel reveal">
          <div className="panel-header">
            <div>
              <h2>Submission activity</h2>
              <p>Your lead capture, over time.</p>
            </div>
            <button
              type="button"
              className="text-button"
              onClick={() => onNavigate('Analytics')}
            >
              View analytics <Icon name="arrow" size={15} />
            </button>
          </div>

          {daily.length > 0 ? (
            <div className="live-chart" role="img" aria-label="Daily submission counts">
              {daily.map((day) => (
                <div
                  className="live-chart-column"
                  key={day.date}
                  title={`${day.count} submissions on ${day.date}`}
                >
                  <span className="live-chart-count">{day.count}</span>
                  <div className="live-chart-track">
                    <span
                      className="live-chart-bar"
                      style={{
                        height: `${day.count > 0
                          ? Math.max(6, (day.count / maxDailyCount) * 100)
                          : 0}%`,
                      }}
                    />
                  </div>
                  <span className="live-chart-label">{formatChartDate(day.date)}</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="chart-empty">
              <div className="chart-grid-lines" aria-hidden="true">
                <span /><span /><span /><span />
              </div>
              <div className="chart-empty-content">
                <div className="chart-empty-icon">
                  <Icon name="analytics" size={22} />
                </div>
                <strong>
                  {loading
                    ? 'Loading your activity…'
                    : error
                      ? 'Activity is temporarily unavailable'
                      : 'No submission activity yet'}
                </strong>
                <p>
                  {loading
                    ? 'Fetching the latest analytics from your workspace.'
                    : error
                      ? 'Check the API connection and try again.'
                      : 'When visitors submit one of your forms, activity will appear here.'}
                </p>
                <button type="button" className="quiet-button" onClick={() => onNavigate('Analytics')}>
                  Explore analytics <Icon name="arrow" size={14} />
                </button>
              </div>
            </div>
          )}
          <div className="chart-footer">
            <span><span className="legend-dot" /> Submissions per day</span>
            <span className="chart-footnote">
              {loading ? 'Updating…' : error ? 'Sync issue' : 'Connected to live data'}
            </span>
          </div>
        </article>

        <article className="panel readiness-panel reveal">
          <div className="panel-header">
            <div>
              <h2>Getting started</h2>
              <p>The pieces of your workflow.</p>
            </div>
            <span className="small-progress-label">3 of 3 ready</span>
          </div>

          <div className="progress-track"><span style={{ width: "100%" }} /></div>

          <div className="setup-list">
            <div className="setup-item">
              <span className="setup-check setup-check--done">
                <Icon name="check" size={14} />
              </span>
              <div className="setup-copy">
                <strong>Widget API</strong>
                <span>Your backend can create and manage widgets.</span>
              </div>
              <span className="setup-status setup-status--ready">Ready</span>
            </div>
            <div className="setup-item">
              <span className="setup-check setup-check--done">
                <Icon name="check" size={14} />
              </span>
              <div className="setup-copy">
                <strong>Embed script</strong>
                <span>Your versioned form renderer is available.</span>
              </div>
              <span className="setup-status setup-status--ready">Ready</span>
            </div>
            <div className="setup-item">
              <span className="setup-check setup-check--done">
                <Icon name="check" size={14} />
              </span>
              <div className="setup-copy">
                <strong>Live dashboard data</strong>
                <span>Metrics and recent submissions load from your API.</span>
              </div>
              <span className="setup-status setup-status--ready">Ready</span>
            </div>
          </div>

          <button type="button" className="setup-footer-button" onClick={() => onNavigate('Widgets')}>
            Continue setup <Icon name="arrow" size={15} />
          </button>
        </article>
      </section>

      <section className="panel recent-panel reveal">
        <div className="panel-header recent-header">
          <div>
            <h2>Recent submissions</h2>
            <p>Your latest leads will be listed here.</p>
          </div>
          <button type="button" className="text-button" onClick={() => onNavigate('Submissions')}>
            View submissions <Icon name="arrow" size={15} />
          </button>
        </div>

        {recentSubmissions.length > 0 ? (
          <div className="recent-submission-list">
            {recentSubmissions.map((submission, index) => {
              const identity = submissionIdentity(submission, index)
              return (
                <div className="recent-submission-row" key={submission.id}>
                  <div className="recent-lead-avatar">
                    {identity.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="recent-lead-copy">
                    <strong>{identity.name}</strong>
                    <span>{identity.detail}</span>
                  </div>
                  <span className="recent-lead-location">
                    {[submission.city, submission.country].filter(Boolean).join(', ') || '—'}
                  </span>
                  <time className="recent-lead-date" dateTime={submission.created_at}>
                    {formatSubmissionDate(submission.created_at)}
                  </time>
                </div>
              )
            })}
          </div>
        ) : (
          <div className="recent-empty">
            <span className="recent-empty-icon"><Icon name="submissions" size={22} /></span>
            <strong>
              {loading
                ? 'Loading recent submissions…'
                : error
                  ? 'Submissions are unavailable'
                  : 'No submissions yet'}
            </strong>
            <p>
              {loading
                ? 'Fetching the latest responses from your workspace.'
                : error
                  ? 'Check the API connection and try again.'
                  : 'When a visitor submits a widget, their response will appear here.'}
            </p>
            <button type="button" className="quiet-button" onClick={() => onNavigate('Submissions')}>
              Open submissions <Icon name="arrow" size={14} />
            </button>
          </div>
        )}
      </section>

      <footer className="page-footer">
        <span>Nez Widget Platform</span>
        <span><Icon name="shield" size={13} /> Built with tenant privacy in mind</span>
      </footer>
    </>
  )
}


function widgetApiOrigin(): string {
  const configured = import.meta.env.VITE_WIDGET_API_BASE_URL?.trim()
  if (configured) return configured.replace(/\/+$/, '')

  const hostname = window.location.hostname
  if (hostname === 'localhost' || hostname === '127.0.0.1') {
    return `${window.location.protocol}//${hostname}:8080`
  }

  return window.location.origin
}

type DraftWidgetField = CreateWidgetInput['fields'][number]

function WidgetsPage({
  widgets,
  loading,
  error,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
}: {
  widgets: Widget[]
  loading: boolean
  error: string
  onRefresh: () => Promise<void>
  onCreate: (input: CreateWidgetInput) => Promise<void>
  onUpdate: (widgetId: string, input: CreateWidgetInput) => Promise<void>
  onDelete: (widgetId: string) => Promise<void>
}) {
  const [type, setType] = useState<'contact' | 'signup'>('contact')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('We would love to hear from you.')
  const [buttonText, setButtonText] = useState('Send message')
  const [fields, setFields] = useState<DraftWidgetField[]>([
    { name: 'name', label: 'Full name', field_type: 'text', required: true },
    { name: 'email', label: 'Email address', field_type: 'email', required: true },
    { name: 'message', label: 'Message', field_type: 'textarea', required: false },
  ])
  const [creating, setCreating] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [actionError, setActionError] = useState('')
  const [successMessage, setSuccessMessage] = useState('')
  const [copiedWidgetId, setCopiedWidgetId] = useState('')
  const [copyErrorWidgetId, setCopyErrorWidgetId] = useState('')
  const [editingWidgetId, setEditingWidgetId] = useState<string | null>(null)
  const [deletingWidgetId, setDeletingWidgetId] = useState('')

  function resetForm() {
    setEditingWidgetId(null)
    setType('contact')
    setTitle('')
    setDescription('We would love to hear from you.')
    setButtonText('Send message')
    setFields([
      { name: 'name', label: 'Full name', field_type: 'text', required: true },
      { name: 'email', label: 'Email address', field_type: 'email', required: true },
      { name: 'message', label: 'Message', field_type: 'textarea', required: false },
    ])
  }

  function startEditing(widget: Widget) {
    setEditingWidgetId(widget.id)
    setType(widget.type === 'signup' ? 'signup' : 'contact')
    setTitle(widget.title)
    setDescription(widget.description ?? '')
    setButtonText(widget.button_text || 'Submit')
    setFields((widget.fields ?? []).map((field) => ({
      name: field.name,
      label: field.label,
      field_type: field.field_type,
      required: field.required,
    })))
    setActionError('')
    setSuccessMessage('')
  }

  async function handleDelete(widget: Widget) {
    if (!window.confirm(
      `Delete "${widget.title}"? This cannot be undone.`,
    )) return

    setActionError('')
    setSuccessMessage('')
    setDeletingWidgetId(widget.id)

    try {
      await onDelete(widget.id)
      if (editingWidgetId === widget.id) resetForm()
      setSuccessMessage(`"${widget.title}" was deleted.`)
    } catch (caught) {
      setActionError(
        caught instanceof Error ? caught.message : 'Could not delete the widget.',
      )
    } finally {
      setDeletingWidgetId('')
    }
  }

  function updateField(index: number, patch: Partial<DraftWidgetField>) {
    setFields((current) =>
      current.map((field, i) => i === index ? { ...field, ...patch } : field),
    )
  }

  function addField() {
    setFields((current) => {
      let number = current.length + 1
      while (current.some((field) => field.name === `field_${number}`)) {
        number += 1
      }
      return [
        ...current,
        {
          name: `field_${number}`,
          label: `Field ${number}`,
          field_type: 'text',
          required: false,
        },
      ]
    })
  }

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setActionError('')
    setSuccessMessage('')

    const names = fields.map((field) => field.name.trim().toLowerCase())
    if (new Set(names).size !== names.length) {
      setActionError('Each field needs a unique field name.')
      return
    }

    const wasEditing = editingWidgetId !== null
    setCreating(true)

    try {
      const input: CreateWidgetInput = {
        type,
        title: title.trim(),
        description: description.trim(),
        button_text: buttonText.trim() || 'Submit',
        display_options: wasEditing
          ? widgets.find((widget) => widget.id === editingWidgetId)?.display_options ?? {}
          : {},
        fields: fields.map((field) => ({
          name: field.name.trim().toLowerCase(),
          label: field.label.trim(),
          field_type: field.field_type,
          required: field.required,
        })),
      }

      if (editingWidgetId) {
        await onUpdate(editingWidgetId, input)
      } else {
        await onCreate(input)
      }

      resetForm()
      setSuccessMessage(
        wasEditing ? 'Widget updated successfully.' :
          'Widget created. Its embed code is ready below.',
      )
    } catch (caught) {
      setActionError(
        caught instanceof Error ? caught.message : 'Could not save the widget.',
      )
    } finally {
      setCreating(false)
    }
  }

  async function refreshList() {
    setActionError('')
    setRefreshing(true)
    try {
      await onRefresh()
    } catch (caught) {
      setActionError(
        caught instanceof Error ? caught.message : 'Could not refresh widgets.',
      )
    } finally {
      setRefreshing(false)
    }
  }

  async function copyEmbed(widget: Widget) {
    const origin = widgetApiOrigin()
    const snippet =
      `<script src="${origin}/widget/v1/widget.js?widget_id=${encodeURIComponent(widget.public_id)}" defer></script>`

    setCopyErrorWidgetId('')
    try {
      await navigator.clipboard.writeText(snippet)
      setCopiedWidgetId(widget.id)
    } catch {
      setCopiedWidgetId('')
      setCopyErrorWidgetId(widget.id)
    }
  }

  return (
    <div className="widgets-workspace">
      <section className="widgets-page-heading reveal">
        <div>
          <div className="eyebrow">
            <span className="eyebrow-dot" /> WIDGET WORKSPACE
          </div>
          <h1>Your forms, <span>your way.</span></h1>
          <p className="page-subtitle">
            Create a form, customise its fields, and embed it on your website.
          </p>
        </div>
        <button
          type="button"
          className="quiet-button widget-refresh-button"
          onClick={() => { void refreshList() }}
          disabled={refreshing}
        >
          {refreshing ? 'Refreshing…' : 'Refresh widgets'}
        </button>
      </section>

      {(error || actionError) && (
        <div className="data-alert" role="alert">{actionError || error}</div>
      )}
      {successMessage && (
        <div className="widget-success" role="status">{successMessage}</div>
      )}

      <div className="widgets-layout">
        <section className="widget-create-card reveal">
          <div className="widget-section-heading">
            <span className="widget-section-icon"><Icon name="plus" size={18} /></span>
            <div>
              <h2>{editingWidgetId ? 'Edit widget' : 'Create a widget'}</h2>
              <p>
                {editingWidgetId
                  ? 'Update your form details and fields.'
                  : 'Start with the basics. You can embed it after saving.'}
              </p>
            </div>
          </div>

          <form className="widget-create-form" onSubmit={(event) => { void handleSubmit(event) }}>
            <label className="widget-form-label">
              <span>Widget name</span>
              <input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                placeholder="e.g. Contact form"
                maxLength={120}
                required
              />
            </label>

            <label className="widget-form-label">
              <span>Form type</span>
              <select
                value={type}
                onChange={(event) => {
                  const next = event.target.value as 'contact' | 'signup'
                  setType(next)
                  setButtonText(next === 'signup' ? 'Sign up' : 'Send message')
                }}
              >
                <option value="contact">Contact form</option>
                <option value="signup">Sign-up form</option>
              </select>
            </label>

            <label className="widget-form-label">
              <span>Description</span>
              <textarea
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                placeholder="Tell visitors what this form is for"
                maxLength={2000}
                rows={3}
              />
            </label>

            <label className="widget-form-label">
              <span>Submit button text</span>
              <input
                value={buttonText}
                onChange={(event) => setButtonText(event.target.value)}
                placeholder="Submit"
                maxLength={50}
              />
            </label>

            <div className="widget-fields-heading">
              <div>
                <h3>Form fields</h3>
                <p>Use a unique field name for each field.</p>
              </div>
              <button type="button" className="widget-add-field" onClick={addField}>
                <Icon name="plus" size={14} /> Add field
              </button>
            </div>

            <div className="widget-draft-fields">
              {fields.map((field, index) => (
                <div className="widget-draft-field" key={`${index}-${field.name}`}>
                  <div className="widget-draft-field-top">
                    <span>FIELD {index + 1}</span>
                    <button
                      type="button"
                      className="widget-remove-field"
                      onClick={() => setFields((current) => current.filter((_, i) => i !== index))}
                      disabled={fields.length <= 1}
                      aria-label={`Remove ${field.label || `field ${index + 1}`}`}
                    >
                      Remove
                    </button>
                  </div>

                  <div className="widget-field-grid">
                    <label className="widget-form-label">
                      <span>Label shown to visitors</span>
                      <input
                        value={field.label}
                        onChange={(event) => updateField(index, { label: event.target.value })}
                        placeholder="Full name"
                        maxLength={80}
                        required
                      />
                    </label>

                    <label className="widget-form-label">
                      <span>Field name</span>
                      <input
                        value={field.name}
                        onChange={(event) => updateField(index, {
                          name: event.target.value.toLowerCase().replace(/[^a-z0-9_]/g, '_').replace(/_+/g, '_'),
                        })}
                        placeholder="full_name"
                        pattern="[a-z][a-z0-9_]*"
                        maxLength={64}
                        required
                      />
                    </label>
                  </div>

                  <div className="widget-field-options">
                    <label className="widget-form-label">
                      <span>Field type</span>
                      <select
                        value={field.field_type}
                        onChange={(event) => updateField(index, { field_type: event.target.value })}
                      >
                        <option value="text">Text</option>
                        <option value="email">Email</option>
                        <option value="textarea">Long text</option>
                        <option value="tel">Phone</option>
                        <option value="number">Number</option>
                        <option value="url">Website URL</option>
                        <option value="checkbox">Checkbox</option>
                        <option value="select">Select (text input in current renderer)</option>
                      </select>
                    </label>

                    <label className="widget-required-toggle">
                      <input
                        type="checkbox"
                        checked={field.required}
                        onChange={(event) => updateField(index, { required: event.target.checked })}
                      />
                      Required
                    </label>
                  </div>
                </div>
              ))}
            </div>

            <div className="widget-form-actions">
              <button className="primary-button widget-create-submit" type="submit" disabled={creating}>
                <Icon name={editingWidgetId ? 'check' : 'plus'} size={16} />
                {creating
                  ? (editingWidgetId ? 'Saving changes…' : 'Creating widget…')
                  : (editingWidgetId ? 'Save changes' : 'Create widget')}
              </button>
              {editingWidgetId && (
                <button
                  type="button"
                  className="quiet-button widget-cancel-edit"
                  disabled={creating}
                  onClick={resetForm}
                >
                  Cancel
                </button>
              )}
            </div>
          </form>
        </section>

        <section className="widget-list-section reveal">
          <div className="widget-section-heading widget-list-heading">
            <span className="widget-section-icon"><Icon name="layers" size={18} /></span>
            <div>
              <h2>Your widgets</h2>
              <p>{widgets.length} {widgets.length === 1 ? 'widget' : 'widgets'} in this workspace</p>
            </div>
          </div>

          {loading && widgets.length === 0 ? (
            <div className="widget-list-empty">
              <strong>Loading your widgets…</strong>
              <p>Fetching widgets for the signed-in workspace.</p>
            </div>
          ) : widgets.length === 0 ? (
            <div className="widget-list-empty">
              <span className="widget-list-empty-icon"><Icon name="widgets" size={22} /></span>
              <strong>No widgets yet</strong>
              <p>Create your first form, then copy its embed code into your website.</p>
            </div>
          ) : (
            <div className="widget-card-list">
              {widgets.map((widget) => {
                const snippet =
                  `<script src="${widgetApiOrigin()}/widget/v1/widget.js?widget_id=${encodeURIComponent(widget.public_id)}" defer></script>`

                return (
                  <article className="widget-list-card" key={widget.id}>
                    <div className="widget-list-card-top">
                      <span className="widget-card-icon"><Icon name="widgets" size={17} /></span>
                      <span className="widget-type-label">{widget.type}</span>
                    </div>
                    <h3>{widget.title}</h3>
                    <p>{widget.description || 'No description provided.'}</p>
                    <div className="widget-card-meta">
                      <span>{(widget.fields ?? []).length} fields</span>
                      <span className="widget-id-label" title={widget.public_id}>
                        ID · {widget.public_id}
                      </span>
                    </div>
                    <div className="widget-card-actions">
                      <button
                        type="button"
                        className="quiet-button widget-edit-button"
                        onClick={() => startEditing(widget)}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="quiet-button widget-delete-button"
                        disabled={deletingWidgetId === widget.id}
                        onClick={() => { void handleDelete(widget) }}
                      >
                        {deletingWidgetId === widget.id ? 'Deleting…' : 'Delete'}
                      </button>
                    </div>
                    <details className="widget-embed-details">
                      <summary>View embed code</summary>
                      <pre>{snippet}</pre>
                      <button
                        type="button"
                        className="primary-button widget-copy-button"
                        onClick={() => { void copyEmbed(widget) }}
                      >
                        {copiedWidgetId === widget.id ? 'Copied!' : 'Copy embed code'}
                      </button>
                      {copyErrorWidgetId === widget.id && (
                        <p className="widget-copy-help" role="status">
                          Clipboard access was unavailable. Select the code above and copy it manually.
                        </p>
                      )}
                    </details>
                  </article>
                )
              })}
            </div>
          )}
        </section>
      </div>
    </div>
  )
}

function PlaceholderPage({
  page,
  onBack,
}: {
  page: Exclude<PageId, 'Overview'>
  onBack: () => void
}) {
  const content = pageDescriptions[page]

  return (
    <section className="section-placeholder reveal">
      <div className="placeholder-symbol">
        <Icon
          name={
            page === 'Widgets'
              ? 'widgets'
              : page === 'Submissions'
                ? 'submissions'
                : page === 'Analytics'
                  ? 'analytics'
                  : 'settings'
          }
          size={25}
        />
      </div>
      <div className="eyebrow"><span className="eyebrow-dot" /> {page.toUpperCase()}</div>
      <h1>{content.subtitle}</h1>
      <p>{content.description}</p>
      <div className="placeholder-divider" />
      <div className="placeholder-note">
        <Icon name="sparkles" size={18} />
        <div>
          <strong>This screen is taking shape.</strong>
          <span>We'll connect this section to the existing Go API in the next implementation step.</span>
        </div>
      </div>
      <button type="button" className="primary-button" onClick={onBack}>
        <Icon name="arrow" size={16} />
        Back to overview
      </button>
    </section>
  )
}

function authErrorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return 'Something went wrong. Please try again.'
}

function LoginScreen({
  mode,
  setMode,
  email,
  setEmail,
  password,
  setPassword,
  tenantName,
  setTenantName,
  busy,
  error,
  onSubmit,
}: {
  mode: 'login' | 'register'
  setMode: (mode: 'login' | 'register') => void
  email: string
  setEmail: (value: string) => void
  password: string
  setPassword: (value: string) => void
  tenantName: string
  setTenantName: (value: string) => void
  busy: boolean
  error: string
  onSubmit: () => void
}) {
  const registering = mode === 'register'

  return (
    <main className="auth-shell">
      <section className="auth-card">
        <div className="auth-brand">
          <div className="brand-symbol" aria-hidden="true">
            <span /><span /><span />
          </div>
          <span className="brand-name">nez<span>widget</span></span>
        </div>

        <div className="auth-eyebrow">
          <span className="eyebrow-dot" />
          WORKSPACE ACCESS
        </div>

        <h1>{registering ? 'Create your workspace.' : 'Good to see you.'}</h1>
        <p className="auth-description">
          {registering
            ? 'Set up your workspace and start collecting leads.'
            : 'Sign in to access your widgets and lead activity.'}
        </p>

        <form
          className="auth-form"
          onSubmit={(event) => {
            event.preventDefault()
            onSubmit()
          }}
        >
          {registering && (
            <label>
              <span>Workspace name</span>
              <input
                type="text"
                value={tenantName}
                onChange={(event) => setTenantName(event.target.value)}
                placeholder="Your workspace"
                maxLength={100}
                autoComplete="organization"
                required
              />
            </label>
          )}

          <label>
            <span>Email address</span>
            <input
              type="email"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
              placeholder="you@example.com"
              autoComplete="email"
              required
            />
          </label>

          <label>
            <span>Password</span>
            <input
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder={registering ? 'At least 8 characters' : 'Your password'}
              minLength={registering ? 8 : undefined}
              maxLength={72}
              autoComplete={registering ? 'new-password' : 'current-password'}
              required
            />
          </label>

          {error && <div className="auth-error" role="alert">{error}</div>}

          <button className="auth-submit" type="submit" disabled={busy}>
            {busy
              ? (registering ? 'Creating workspace…' : 'Signing in…')
              : (registering ? 'Create workspace' : 'Sign in')}
            {!busy && <Icon name="arrow" size={17} />}
          </button>
        </form>

        <div className="auth-switch">
          <span>{registering ? 'Already have an account?' : 'New to Nez Widget?'}</span>
          <button
            type="button"
            onClick={() => setMode(registering ? 'login' : 'register')}
          >
            {registering ? 'Sign in' : 'Create a workspace'}
          </button>
        </div>

        <div className="auth-security-note">
          <Icon name="shield" size={15} />
          <span>Secure, authenticated access to your workspace.</span>
        </div>
      </section>
    </main>
  )
}

function getInitialTheme(): Theme {
  try {
    const saved = window.localStorage.getItem('nez-widget-theme')
    if (saved === 'light' || saved === 'dark') return saved
    return window.matchMedia('(prefers-color-scheme: dark)').matches
      ? 'dark'
      : 'light'
  } catch {
    return 'light'
  }
}

function App() {
  const [activePage, setActivePage] = useState<PageId>('Overview')
  const [theme, setTheme] = useState<Theme>(getInitialTheme)
  const [mobileOpen, setMobileOpen] = useState(false)
  const [user, setUser] = useState<User | null>(null)
  const [authChecking, setAuthChecking] = useState(true)
  const [authMode, setAuthMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [tenantName, setTenantName] = useState('')
  const [authBusy, setAuthBusy] = useState(false)
  const [authError, setAuthError] = useState('')
  const [widgets, setWidgets] = useState<Widget[]>([])
  const [analytics, setAnalytics] = useState<SubmissionAnalytics | null>(null)
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [dataLoading, setDataLoading] = useState(true)
  const [dataError, setDataError] = useState('')
  const contentRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    document.documentElement.style.colorScheme = theme

    try {
      window.localStorage.setItem('nez-widget-theme', theme)
    } catch {
      // The theme still works for this session if storage is unavailable.
    }
  }, [theme])

  useEffect(() => {
    let active = true

    async function restoreSession() {
      if (!getAccessToken()) {
        if (active) setAuthChecking(false)
        return
      }

      try {
        const result = await getMe()
        if (active) setUser(result.user)
      } catch (error) {
        if (error instanceof ApiError && error.status === 401) {
          clearAccessToken()
        } else if (active) {
          setAuthError(authErrorMessage(error))
        }

        if (active) setUser(null)
      } finally {
        if (active) setAuthChecking(false)
      }
    }

    void restoreSession()
    return () => {
      active = false
    }
  }, [])

  useEffect(() => {
    if (!user) return

    let active = true

    Promise.all([
      getWidgets(),
      getAnalytics(),
      getSubmissions({ limit: 50, offset: 0 }),
    ])
      .then(([widgetResult, analyticsResult, submissionResult]) => {
        if (!active) return
        setWidgets(widgetResult.widgets ?? [])
        setAnalytics(analyticsResult)
        setSubmissions(submissionResult.submissions ?? [])
      })
      .catch((error: unknown) => {
        if (!active) return
        setDataError(authErrorMessage(error))
        if (error instanceof ApiError && error.status === 401) {
          clearAccessToken()
          setWidgets([])
          setAnalytics(null)
          setSubmissions([])
          setUser(null)
        }
      })
      .finally(() => {
        if (active) setDataLoading(false)
      })

    return () => {
      active = false
    }
  }, [user])

  useEffect(() => {
    const prefersReducedMotion = window.matchMedia(
      '(prefers-reduced-motion: reduce)',
    ).matches

    if (prefersReducedMotion) return

    const lenis = new Lenis({ autoRaf: true, anchors: true })
    return () => lenis.destroy()
  }, [])

  useGSAP(
    () => {
      const prefersReducedMotion = window.matchMedia(
        '(prefers-reduced-motion: reduce)',
      ).matches

      if (prefersReducedMotion) return

      gsap.fromTo(
        '.reveal',
        { autoAlpha: 0, y: 12 },
        {
          autoAlpha: 1,
          y: 0,
          duration: 0.48,
          stagger: 0.055,
          ease: 'power2.out',
          clearProps: 'all',
        },
      )
    },
    {
      scope: contentRef,
      dependencies: [activePage, user],
      revertOnUpdate: true,
    },
  )

  async function handleAuthSubmit() {
    setAuthError('')
    setAuthBusy(true)

    try {
      if (authMode === 'register') {
        await register(tenantName.trim(), email.trim(), password)
      }

      const result = await login(email.trim(), password)
      setDataLoading(true)
      setDataError('')
      setUser(result.user)
      setActivePage('Overview')
      setPassword('')
    } catch (error) {
      setAuthError(authErrorMessage(error))
    } finally {
      setAuthBusy(false)
    }
  }

  function handleLogout() {
    clearAccessToken()
    setWidgets([])
    setAnalytics(null)
    setSubmissions([])
    setDataLoading(false)
    setDataError('')
    setUser(null)
    setActivePage('Overview')
    setMobileOpen(false)
    setAuthError('')
    setPassword('')
  }

  async function handleCreateWidget(input: CreateWidgetInput): Promise<void> {
    await createWidget(input)
    const result = await getWidgets()
    setWidgets(result.widgets ?? [])
  }

  async function handleUpdateWidget(
    widgetId: string,
    input: CreateWidgetInput,
  ): Promise<void> {
    await updateWidget(widgetId, input)
    const result = await getWidgets()
    setWidgets(result.widgets ?? [])
    const submissionResult = await getSubmissions({ limit: 50, offset: 0 })
    setSubmissions(submissionResult.submissions ?? [])
    setAnalytics(await getAnalytics())
  }

  async function handleDeleteWidget(widgetId: string): Promise<void> {
    await deleteWidget(widgetId)
    const result = await getWidgets()
    setWidgets(result.widgets ?? [])
    const submissionResult = await getSubmissions({ limit: 50, offset: 0 })
    setSubmissions(submissionResult.submissions ?? [])
    setAnalytics(await getAnalytics())
  }

  async function refreshWidgetList(): Promise<void> {
    const result = await getWidgets()
    setWidgets(result.widgets ?? [])
  }

  async function refreshSubmissionList(): Promise<void> {
    const result = await getSubmissions({ limit: 50, offset: 0 })
    setSubmissions(result.submissions ?? [])
  }

  function navigate(page: PageId) {
    setActivePage(page)
    setMobileOpen(false)
  }

  if (authChecking) {
    return (
      <main className="auth-shell">
        <div className="auth-loading">
          <div className="brand-symbol" aria-hidden="true">
            <span /><span /><span />
          </div>
          <strong>Checking your session…</strong>
        </div>
      </main>
    )
  }

  if (!user) {
    return (
      <LoginScreen
        mode={authMode}
        setMode={(mode) => {
          setAuthMode(mode)
          setAuthError('')
        }}
        email={email}
        setEmail={setEmail}
        password={password}
        setPassword={setPassword}
        tenantName={tenantName}
        setTenantName={setTenantName}
        busy={authBusy}
        error={authError}
        onSubmit={() => { void handleAuthSubmit() }}
      />
    )
  }

  return (
    <div className="app-shell">
      <Sidebar
        activePage={activePage}
        onNavigate={navigate}
        mobileOpen={mobileOpen}
        onClose={() => setMobileOpen(false)}
        userEmail={user.email}
        onLogout={handleLogout}
      />

      <div className="app-main">
        <Header
          activePage={activePage}
          onCreateWidget={() => navigate('Widgets')}
          onToggleSidebar={() => setMobileOpen((open) => !open)}
          userEmail={user.email}
          theme={theme}
          onToggleTheme={() => setTheme((current) =>
            current === 'light' ? 'dark' : 'light',
          )}
        />

        <main className="page-content" ref={contentRef}>
          {activePage === 'Overview' ? (
            <Overview
              onNavigate={navigate}
              analytics={analytics}
              widgetCount={widgets.length}
              submissions={submissions}
              loading={dataLoading}
              error={dataError}
            />
          ) : activePage === 'Widgets' ? (
            <WidgetsPage
              widgets={widgets}
              loading={dataLoading}
              error={dataError}
              onRefresh={refreshWidgetList}
              onCreate={handleCreateWidget}
              onUpdate={handleUpdateWidget}
              onDelete={handleDeleteWidget}
            />
          ) : activePage === 'Submissions' ? (
            <SubmissionsPage
              submissions={submissions}
              widgets={widgets}
              loading={dataLoading}
              error={dataError}
              onRefresh={refreshSubmissionList}
            />
          ) : activePage === 'Analytics' ? (
            <AnalyticsPage
              analytics={analytics}
              widgets={widgets}
              loading={dataLoading}
              error={dataError}
              onLoadAnalytics={getAnalytics}
            />
          ) : (
            <PlaceholderPage
              page={activePage}
              onBack={() => navigate('Overview')}
            />
          )}
        </main>
      </div>
    </div>
  )
}

export default App

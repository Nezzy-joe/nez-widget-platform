const API_PREFIX = '/api'
const TOKEN_KEY = 'nez_access_token'

export interface User {
  id: string
  tenant_id: string
  email: string
  created_at: string
  updated_at: string
}

export interface LoginResponse {
  access_token: string
  token_type: string
  user: User
}

export interface WidgetField {
  name: string
  label: string
  field_type: string
  required: boolean
  position: number
}

export interface Widget {
  id: string
  public_id: string
  type: string
  title: string
  description?: string
  button_text: string
  display_options: Record<string, unknown>
  fields: WidgetField[]
  created_at: string
  updated_at: string
}

export interface Submission {
  id: string
  widget_id: string
  data: Record<string, unknown>
  country?: string
  city?: string
  created_at: string
}

export interface SubmissionPage {
  submissions: Submission[]
  pagination: {
    limit: number
    offset: number
    total: number
  }
}

export interface DailySubmissionCount {
  date: string
  count: number
}

export interface SubmissionAnalytics {
  total_submissions: number
  submissions_last_7_days: number
  submissions_last_30_days: number
  daily: DailySubmissionCount[]
}

interface ErrorResponse {
  error?: string
}

export class ApiError extends Error {
  public readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function getAccessToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function clearAccessToken(): void {
  sessionStorage.removeItem(TOKEN_KEY)
}

async function request<T>(
  path: string,
  options: RequestInit = {},
  authenticated = true,
): Promise<T> {
  const headers = new Headers(options.headers)

  if (options.body && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (authenticated) {
    const token = getAccessToken()
    if (token) {
      headers.set('Authorization', `Bearer ${token}`)
    }
  }

  const response = await fetch(`${API_PREFIX}${path}`, {
    ...options,
    headers,
  })

  const payload: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    const error = payload as ErrorResponse | null
    throw new ApiError(
      response.status,
      error?.error || `Request failed with status ${response.status}`,
    )
  }

  return payload as T
}

export async function login(
  email: string,
  password: string,
): Promise<LoginResponse> {
  const result = await request<LoginResponse>(
    '/auth/login',
    {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    },
    false,
  )

  sessionStorage.setItem(TOKEN_KEY, result.access_token)
  return result
}

export async function register(
  tenantName: string,
  email: string,
  password: string,
): Promise<{ user: User }> {
  return request<{ user: User }>(
    '/auth/register',
    {
      method: 'POST',
      body: JSON.stringify({
        tenant_name: tenantName,
        email,
        password,
      }),
    },
    false,
  )
}

export function getMe(): Promise<{ user: User }> {
  return request('/auth/me')
}

export function getWidgets(): Promise<{ widgets: Widget[] }> {
  return request('/widgets')
}

export interface CreateWidgetInput {
  type: string
  title: string
  description: string
  button_text: string
  display_options: Record<string, unknown>
  fields: Array<{
    name: string
    label: string
    field_type: string
    required: boolean
  }>
}

export function createWidget(input: CreateWidgetInput): Promise<unknown> {
  return request('/widgets', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateWidget(
  widgetId: string,
  input: CreateWidgetInput,
): Promise<{ widget: Widget }> {
  return request(`/widgets/${encodeURIComponent(widgetId)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export async function deleteWidget(widgetId: string): Promise<void> {
  await request<unknown>(`/widgets/${encodeURIComponent(widgetId)}`, {
    method: 'DELETE',
  })
}

export function getAnalytics(
  widgetId?: string,
): Promise<SubmissionAnalytics> {
  const query = widgetId
    ? `?widget_id=${encodeURIComponent(widgetId)}`
    : ''

  return request(`/dashboard/analytics${query}`)
}

export function getSubmissions(options: {
  limit?: number
  offset?: number
  widgetId?: string
} = {}): Promise<SubmissionPage> {
  const query = new URLSearchParams()

  if (options.limit !== undefined) {
    query.set('limit', String(options.limit))
  }

  if (options.offset !== undefined) {
    query.set('offset', String(options.offset))
  }

  if (options.widgetId) {
    query.set('widget_id', options.widgetId)
  }

  const suffix = query.size > 0 ? `?${query.toString()}` : ''
  return request(`/dashboard/submissions${suffix}`)
}

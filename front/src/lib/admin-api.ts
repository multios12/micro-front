export type SiteApiTab = {
  tab_label: string
  tab_url: string
}

export type SiteApiResponse = {
  id: number
  site_title: string
  site_subtitle: string
  site_description: string
  tabs: SiteApiTab[]
  foot_information: string
  copyright: string
  updated_at: string
}

export type SiteApiRequest = {
  site_title: string
  site_subtitle: string
  site_description: string
  tabs: SiteApiTab[]
  foot_information: string
  copyright: string
}

export type BlogListApiItem = {
  id: number
  title: string
  summary: string
  category: string
  status: 'public' | 'private'
  published_at: string
  updated_at: string
}

export type BlogListApiResponse = {
  items: BlogListApiItem[]
  page: number
  per_page: number
  total: number
  total_pages: number
}

export type BlogDetailApiResponse = {
  id: number
  title: string
  content: string
  summary: string
  category: string
  status: 'public' | 'private'
  published_at: string
  updated_at: string
}

export type BlogUpsertApiRequest = {
  title: string
  content: string
  category: string
  status: 'public' | 'private'
  published_at: string
}

export type BlogImageApiItem = {
  id: number
  blog_id: number
  url: string
  alt_text: string
  created_at: string
  updated_at: string
}

export type BlogImageApiResponse = {
  items: BlogImageApiItem[]
}

export type PublishTarget = 'all' | 'index' | 'blogs' | 'blog' | 'about'

export class ApiError extends Error {
  status: number
  code?: string
  fields?: Record<string, string>

  constructor(
    message: string,
    options: {
      status: number
      code?: string
      fields?: Record<string, string>
    },
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = options.status
    this.code = options.code
    this.fields = options.fields
  }
}

export function extractValidationFields(error: unknown): Record<string, string> | null {
  if (typeof error !== 'object' || error === null) {
    return null
  }

  const record = error as { fields?: unknown }
  if (typeof record.fields !== 'object' || record.fields === null) {
    return null
  }

  const fields = record.fields as Record<string, string>
  return Object.keys(fields).length > 0 ? fields : null
}

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !(value instanceof FormData) && !(value instanceof URLSearchParams)

async function readErrorResponse(response: Response): Promise<{ code?: string; message: string; fields?: Record<string, string> }> {
  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    const payload = await response.json().catch(() => null)
    if (payload && typeof payload === 'object') {
      const record = payload as Record<string, unknown>
      return {
        code: typeof record.code === 'string' ? record.code : undefined,
        message: typeof record.message === 'string' ? record.message : response.statusText || 'Request failed',
        fields:
          record.fields && typeof record.fields === 'object'
            ? (record.fields as Record<string, string>)
            : undefined,
      }
    }
  }

  const text = await response.text().catch(() => '')
  return {
    message: text || response.statusText || 'Request failed',
  }
}

async function requestJson<T>(path: string, init: Omit<RequestInit, 'body'> & { body?: unknown } = {}): Promise<T> {
  const headers = new Headers(init.headers)
  let body: BodyInit | null | undefined = init.body as BodyInit | null | undefined

  if (isPlainObject(init.body)) {
    if (!headers.has('content-type')) {
      headers.set('content-type', 'application/json')
    }
    body = JSON.stringify(init.body)
  }

  const response = await fetch(path, {
    ...init,
    headers,
    body,
  })

  if (!response.ok) {
    const error = await readErrorResponse(response)
    throw new ApiError(error.message, {
      status: response.status,
      code: error.code,
      fields: error.fields,
    })
  }

  if (response.status === 204) {
    return undefined as T
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    return (await response.json()) as T
  }

  return (await response.text()) as T
}

async function requestFormData<T>(path: string, init: RequestInit & { body: FormData }): Promise<T> {
  const response = await fetch(path, init)

  if (!response.ok) {
    const error = await readErrorResponse(response)
    throw new ApiError(error.message, {
      status: response.status,
      code: error.code,
      fields: error.fields,
    })
  }

  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    return (await response.json()) as T
  }

  return (await response.text()) as T
}

export async function fetchSiteSettings(): Promise<SiteApiResponse> {
  return requestJson<SiteApiResponse>('admin/api/site')
}

export async function updateSiteSettings(request: SiteApiRequest): Promise<SiteApiResponse> {
  return requestJson<SiteApiResponse>('admin/api/site', {
    method: 'PUT',
    body: request,
  })
}

export async function fetchBlogList(params: {
  page?: number
  perPage?: number
  status?: 'public' | 'private' | ''
} = {}): Promise<BlogListApiResponse> {
  const search = new URLSearchParams()
  if (params.page && params.page > 1) {
    search.set('page', String(params.page))
  }
  if (params.perPage && params.perPage !== 20) {
    search.set('per_page', String(params.perPage))
  }
  if (params.status) {
    search.set('status', params.status)
  }
  const query = search.toString()
  return requestJson<BlogListApiResponse>(`admin/api/blogs${query ? `?${query}` : ''}`)
}

export async function fetchBlogDetail(blogId: number): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>(`admin/api/blogs/${blogId}`)
}

export async function updateBlog(blogId: number, request: BlogUpsertApiRequest): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>(`admin/api/blogs/${blogId}`, {
    method: 'PUT',
    body: request,
  })
}

export async function createBlog(request: BlogUpsertApiRequest): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>('admin/api/blogs', {
    method: 'POST',
    body: request,
  })
}

export async function deleteBlog(blogId: number): Promise<{ id: number; result: string }> {
  return requestJson<{ id: number; result: string }>(`admin/api/blogs/${blogId}`, {
    method: 'DELETE',
  })
}

export async function fetchBlogImages(blogId: number): Promise<BlogImageApiResponse> {
  return requestJson<BlogImageApiResponse>(`admin/api/blogs/${blogId}/images`)
}

export async function uploadBlogImage(
  blogId: number,
  file: File,
  altText = '',
): Promise<BlogImageApiItem> {
  const formData = new FormData()
  formData.set('file', file)
  formData.set('alt_text', altText)
  return requestFormData<BlogImageApiItem>(`admin/api/blogs/${blogId}/images`, {
    method: 'POST',
    body: formData,
  })
}

export async function deleteBlogImage(
  blogId: number,
  imageId: number,
): Promise<{ id: number; blog_id: number; result: string }> {
  return requestJson<{ id: number; blog_id: number; result: string }>(
    `admin/api/blogs/${blogId}/images/${imageId}`,
    {
      method: 'DELETE',
    },
  )
}

export async function publish(target: PublishTarget, blogId?: number): Promise<{ result: string }> {
  return requestJson<{ result: string }>('admin/api/publish', {
    method: 'POST',
    body: blogId === undefined ? { target } : { target, blog_id: blogId },
  })
}

export async function createBlogPreview(blogId: number | 'about'): Promise<{ result: string; url: string }> {
  return requestJson<{ result: string; url: string }>(`admin/api/blogs/${blogId}/preview`, {
    method: 'POST',
    body: typeof blogId === 'number' ? { blog_id: blogId } : { blog_id: 0 },
  })
}

export async function createSitePreview(): Promise<{ result: string; url: string }> {
  return requestJson<{ result: string; url: string }>('admin/api/site/preview', {
    method: 'POST',
  })
}

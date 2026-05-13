export type SiteApiTab = {
  tab_label: string
  tab_url: string
}

export type SiteApiResponse = {
  id: number
  site_title: string
  site_subtitle: string
  site_description: string
  site_url: string
  tabs: SiteApiTab[]
  foot_information: string
  copyright: string
  updated_at: string
}

export type SiteApiRequest = {
  site_title: string
  site_subtitle: string
  site_description: string
  site_url: string
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
  title_image_template: string
  published_at: string
  updated_at: string
}

export type BlogUpsertApiRequest = {
  title: string
  content: string
  category: string
  status: 'public' | 'private'
  title_image_template: string
  published_at: string
}

export type TitleImageTemplateApiItem = {
  id: string
  label: string
  description: string
}

export type TitleImageTemplatesApiResponse = {
  items: TitleImageTemplateApiItem[]
}

export type TitleImagePreviewApiResponse = {
  svg: string
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

export type BlogImageViewItem = BlogImageApiItem & {
  displayUrl: string
}

export type BlogImageViewResponse = {
  items: BlogImageViewItem[]
}

export type PublishTarget = 'all' | 'index' | 'blogs' | 'blog' | 'about'

export type PreviewApiResponse = {
  result: string
  url: string
}

export type PreviewViewResponse = PreviewApiResponse & {
  displayUrl: string
}

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

// 管理画面ベースURL
const appBasePath = (() => {
  const base = import.meta.env.BASE_URL ?? '/'
  if (base === './' || base === '/') {
    return './'
  }
  return `/${base.replace(/^\/+|\/+$/g, '')}/`
})()

// 管理画面URL【<appBasePath>/admin】
const adminPath = (path: string) => `${appBasePath}admin/${path.replace(/^\/+/, '')}`
// 管理画面APIURL [<appBasePath/admin/api/*]
export const adminApiPath = (path: string) => adminPath(`api/${path}`)
// 管理画面リソースURL [<appBasePath/*]
export const adminResourcePath = (path: string) => {
  if (/^(?:[a-z][a-z\d+\-.]*:)?\/\//i.test(path) || /^(?:data|blob):/i.test(path)) {
    return path
  }
  if (path.startsWith('/admin/')) {
    return `${appBasePath}${path.replace(/^\/+/, '')}`
  }
  if (path.startsWith('admin/')) {
    return `${appBasePath}${path}`
  }
  return path
}
// 管理画面アップロードパス
export const blogImageUploadPath = (blogId: number) => adminApiPath(`blogs/${blogId}/images`)

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
  return requestJson<SiteApiResponse>(adminApiPath('site'))
}

export async function updateSiteSettings(request: SiteApiRequest): Promise<SiteApiResponse> {
  return requestJson<SiteApiResponse>(adminApiPath('site'), {
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
  return requestJson<BlogListApiResponse>(`${adminApiPath('blogs')}${query ? `?${query}` : ''}`)
}

export async function fetchBlogDetail(blogId: number): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>(adminApiPath(`blogs/${blogId}`))
}

export async function updateBlog(blogId: number, request: BlogUpsertApiRequest): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>(adminApiPath(`blogs/${blogId}`), {
    method: 'PUT',
    body: request,
  })
}

export async function createBlog(request: BlogUpsertApiRequest): Promise<BlogDetailApiResponse> {
  return requestJson<BlogDetailApiResponse>(adminApiPath('blogs'), {
    method: 'POST',
    body: request,
  })
}

export async function fetchTitleImageTemplates(): Promise<TitleImageTemplatesApiResponse> {
  return requestJson<TitleImageTemplatesApiResponse>(adminApiPath('title-image/templates'))
}

export async function createTitleImagePreview(request: {
  title: string
  category?: string
  template: string
}): Promise<TitleImagePreviewApiResponse> {
  return requestJson<TitleImagePreviewApiResponse>(adminApiPath('title-image/preview'), {
    method: 'POST',
    body: request,
  })
}

export async function deleteBlog(blogId: number): Promise<{ id: number; result: string }> {
  return requestJson<{ id: number; result: string }>(adminApiPath(`blogs/${blogId}`), {
    method: 'DELETE',
  })
}

export async function fetchBlogImages(blogId: number): Promise<BlogImageViewResponse> {
  const response = await requestJson<BlogImageApiResponse>(adminApiPath(`blogs/${blogId}/images`))
  return {
    items: response.items.map((item) => ({
      ...item,
      displayUrl: adminResourcePath(item.url),
    })),
  }
}

export async function uploadBlogImage(
  blogId: number,
  file: File,
  altText = '',
): Promise<BlogImageApiItem> {
  const formData = new FormData()
  formData.set('file', file)
  formData.set('alt_text', altText)
  return requestFormData<BlogImageApiItem>(blogImageUploadPath(blogId), {
    method: 'POST',
    body: formData,
  })
}

export async function deleteBlogImage(
  blogId: number,
  imageId: number,
): Promise<{ id: number; blog_id: number; result: string }> {
  return requestJson<{ id: number; blog_id: number; result: string }>(
    adminApiPath(`blogs/${blogId}/images/${imageId}`),
    {
      method: 'DELETE',
    },
  )
}

export async function publish(target: PublishTarget, blogId?: number): Promise<{ result: string }> {
  return requestJson<{ result: string }>(adminApiPath('publish'), {
    method: 'POST',
    body: blogId === undefined ? { target } : { target, blog_id: blogId },
  })
}

export async function createBlogPreview(blogId: number | 'about'): Promise<PreviewViewResponse> {
  const response = await requestJson<PreviewApiResponse>(adminApiPath(`blogs/${blogId}/preview`), {
    method: 'POST',
    body: typeof blogId === 'number' ? { blog_id: blogId } : { blog_id: 0 },
  })
  return {
    ...response,
    displayUrl: adminResourcePath(response.url),
  }
}

export async function createSitePreview(): Promise<PreviewViewResponse> {
  const response = await requestJson<PreviewApiResponse>(adminApiPath('site/preview'), {
    method: 'POST',
  })
  return {
    ...response,
    displayUrl: adminResourcePath(response.url),
  }
}

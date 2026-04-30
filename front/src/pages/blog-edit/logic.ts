export type BlogEditMode = 'blog' | 'about' | 'new'

export type BlogEditViewModel = {
  mode: BlogEditMode
  headerTitle: string
  articleId: string
  cancelHref: string
  saveHref: string
  showPublishButton: boolean
  imageNote: string
  content: string
  deleteMessage: string
}

export const ABOUT_BLOG_ID = 9999999

export const createBlogEditViewModel = (blogId: string): BlogEditViewModel => {
  const mode: BlogEditMode = blogId === 'about' ? 'about' : blogId === 'new' || blogId === '' ? 'new' : 'blog'

  return {
    mode,
    headerTitle: 'BLOG EDIT',
    articleId: mode === 'about' ? '9999999' : mode === 'new' ? '...' : '42',
    cancelHref: mode === 'about' ? '#/dashboard' : '#/blogs',
    saveHref: mode === 'about' ? '#/blog-edit/about' : mode === 'new' ? '#/blog-edit' : `#/blog-edit/${blogId}`,
    showPublishButton: mode === 'blog',
    imageNote: '画像サイズを1920x1080以下に変換して、PNGで表示します。',
    content:
      mode === 'about'
        ? `## about

プロフィールページ用の本文です。

- ブログの説明
- 管理者プロフィール
- リンク一覧

![sample](/admin/images/41/1.png)`
        : mode === 'new'
          ? ''
          : `## 公開サイトの導線を見直す

Latest セクションから記事一覧へ自然に遷移できるように調整しました。

### 確認ポイント

- トップページの見出し導線
- 記事一覧でのカード選択範囲
- カテゴリページとの役割分担

![sample](/admin/images/42/3.png)`,
    deleteMessage:
      mode === 'about'
        ? 'about 記事を削除しますか。削除後は元に戻せません。'
        : mode === 'new'
          ? ''
          : 'この記事を削除しますか。削除後は元に戻せません。',
  }
}

export function resolveBlogEditMode(blogId: string): BlogEditMode {
  if (blogId === 'about') {
    return 'about'
  }
  if (blogId === 'new' || blogId === '') {
    return 'new'
  }
  return 'blog'
}

export function resolveBlogEditTargetId(blogId: string): number {
  if (blogId === 'new' || blogId === '') {
    return Number.NaN
  }
  return blogId === 'about' ? ABOUT_BLOG_ID : Number.parseInt(blogId, 10)
}

export function createBlogEditLabels(blogId: string) {
  const mode = resolveBlogEditMode(blogId)

  return {
    headerTitle: mode === 'about' ? 'ABOUT' : mode === 'new' ? 'NEW BLOG' : 'BLOG EDIT',
    cancelHref: mode === 'about' ? '#/dashboard' : '#/blogs',
    imageNote: '画像サイズを1920x1080以下に変換して、PNGで表示します。',
    deleteMessage:
      mode === 'about'
        ? 'about 記事を削除しますか。削除後は元に戻せません。'
        : mode === 'new'
          ? ''
          : 'この記事を削除しますか。削除後は元に戻せません。',
  }
}

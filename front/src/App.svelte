<script lang="ts">
  import { onMount } from 'svelte'

  import AdminShell from './components/AdminShell.svelte'
  import BlogEditPage from './pages/blog-edit/BlogEditPage.svelte'
  import BlogsPage from './pages/blogs/BlogsPage.svelte'
  import DashboardPage from './pages/dashboard/DashboardPage.svelte'
  import SitePage from './pages/site/SitePage.svelte'
  import { blogCount, refreshBlogCount } from './lib/blog-count'

  type Route = 'dashboard' | 'blogs' | 'site' | 'blog-edit'
  type NavKey = 'dashboard' | 'blogs' | 'site' | 'about'

  let route: Route = 'dashboard'
  let blogId = 'new'
  const parseRoute = (hash: string): { route: Route; blogId?: string } | null => {
    if (hash === '#/about') {
      return { route: 'blog-edit', blogId: 'about' }
    }

    const match = hash.match(/^#\/(dashboard|blogs|site|blog-edit)(?:\/([^/]+))?\/?$/)
    if (!match) {
      return null
    }

    return {
      route: match[1] as Route,
      blogId: match[2],
    }
  }

  const syncRoute = () => {
    const parsed = parseRoute(location.hash)
    if (parsed) {
      route = parsed.route
      blogId = parsed.blogId ?? (parsed.route === 'blog-edit' ? 'new' : '42')
    }
  }

  onMount(() => {
    if (!parseRoute(location.hash)) {
      location.hash = '#/dashboard'
    }

    void refreshBlogCount()

    syncRoute()
    const onHashChange = () => syncRoute()

    addEventListener('hashchange', onHashChange)
    return () => removeEventListener('hashchange', onHashChange)
  })

  $: activeNav = (route === 'blog-edit' && blogId === 'about' ? 'about' : route === 'blog-edit' ? 'blogs' : route) as NavKey
</script>

<svelte:head>
  <meta name="description" content="micro-front の管理画面" />
</svelte:head>

<AdminShell active={activeNav} blogCount={$blogCount}>
  {#if route === 'dashboard'}
    <DashboardPage />
  {:else if route === 'blogs'}
    <BlogsPage />
  {:else if route === 'site'}
    <SitePage />
  {:else if route === 'blog-edit'}
    <BlogEditPage blogId={blogId} />
  {/if}
</AdminShell>

<script lang="ts">
  import { onMount } from 'svelte'

  import AdminHeader from '../../components/AdminHeader.svelte'
  import BlogListTable from '../../components/BlogListTable.svelte'
  import FlashMessage from '../../components/FlashMessage.svelte'
  import Pagination from '../../components/Pagination.svelte'
  import Toast from '../../components/Toast.svelte'
  import { fetchBlogList, fetchSiteSettings, publish } from '../../lib/admin-api'
  import {
    buildDashboardPaginationItems,
    createDashboardData,
    dashboardPageSize,
    dashboardPublishAction,
    dashboardTableActions,
    mapDashboardBlogRows,
    type DashboardBlogRow,
    type DashboardPaginationItem,
    type DashboardSettingRow,
    type DashboardStatCard,
  } from './logic'

  let loading = true
  let error = ''
  let toastOpen = false
  let toastTitle = ''
  let toastMessage = ''
  let toastTone: 'success' | 'warning' | 'error' = 'error'
  let dashboardBlogRows: DashboardBlogRow[] = []
  let dashboardStatCards: DashboardStatCard[] = []
  let dashboardSettings: DashboardSettingRow[] = []
  let dashboardPaginationItems: DashboardPaginationItem[] = []
  let currentPage = 1
  let totalPages = 1
  let allDashboardRows: DashboardBlogRow[] = []
  let publishing = false

  const refreshPage = () => {
    const start = (currentPage - 1) * dashboardPageSize
    dashboardBlogRows = allDashboardRows.slice(start, start + dashboardPageSize)
    dashboardPaginationItems = buildDashboardPaginationItems(currentPage, totalPages, (page) => {
      currentPage = page
      refreshPage()
    })
  }

  const publishAll = async () => {
    publishing = true
    toastOpen = false

    try {
      await publish('all')
      toastTitle = '公開開始'
      toastMessage = '全体の公開用 HTML を再生成しました。'
      toastTone = 'success'
      toastOpen = true
    } catch (err) {
      toastTitle = 'エラー'
      toastMessage = err instanceof Error ? err.message : '公開処理に失敗しました'
      toastTone = 'error'
      toastOpen = true
    } finally {
      publishing = false
    }
  }

  onMount(async () => {
    loading = true
    error = ''
    toastOpen = false

    try {
      const [blogs, site] = await Promise.all([
        fetchBlogList({ perPage: 10000 }),
        fetchSiteSettings(),
      ])

      const dashboardData = createDashboardData({
        blogs: blogs.items,
        site,
      })
      totalPages = dashboardData.totalPages
      dashboardStatCards = dashboardData.statCards
      dashboardSettings = dashboardData.settings
      allDashboardRows = mapDashboardBlogRows(blogs.items)
      dashboardBlogRows = allDashboardRows.slice(0, dashboardPageSize)
      dashboardPaginationItems = buildDashboardPaginationItems(1, totalPages, (page) => {
        currentPage = page
        refreshPage()
      })

      currentPage = 1
    } catch (err) {
      toastTitle = 'エラー'
      toastMessage = err instanceof Error ? err.message : 'ダッシュボードの読み込みに失敗しました'
      toastOpen = true
    } finally {
      loading = false
    }
  })
</script>

<svelte:head>
  <title>Dashboard | micro-front</title>
</svelte:head>

<AdminHeader title="Dashboard">
  <svelte:fragment slot="actions">
    <button
      class={dashboardPublishAction.primary ? 'admin-button admin-button-primary' : 'admin-button'}
      type="button"
      on:click={publishAll}
      disabled={publishing}
    >
      {publishing ? '再生成中...' : dashboardPublishAction.label}
    </button>
  </svelte:fragment>
</AdminHeader>

{#if toastOpen}
  <Toast tone={toastTone} title={toastTitle} message={toastMessage} onClose={() => (toastOpen = false)} />
{/if}

{#if loading}
  <FlashMessage tone="success" title="読み込み中" message="管理API からダッシュボード情報を取得しています。" />
{/if}

<section class="admin-grid-4">
  {#each dashboardStatCards as card}
    <article class="admin-stat">
      <span class="admin-label">{card.label}</span>
      <strong>{card.value}</strong>
      {#if card.note}
        <span class="admin-note">{card.note}</span>
      {/if}
    </article>
  {/each}
</section>

<section class="admin-table-wrap">
  <div class="admin-panel-head">
    <h2>記事一覧</h2>
    <div class="admin-topbar-actions">
      {#each dashboardTableActions as action}
        <a
          class={action.primary ? 'admin-button admin-button-primary' : 'admin-button'}
          href={action.href}
        >
          {action.label}
        </a>
      {/each}
    </div>
  </div>

  <BlogListTable rows={dashboardBlogRows} />

  <div class="mt-4">
    <Pagination items={dashboardPaginationItems} />
  </div>
</section>

<section class="admin-panel">
  <div class="admin-panel-head">
    <h2>設定値一覧</h2>
  </div>

  <table class="admin-table">
    <thead>
      <tr>
        <th>項目</th>
        <th>値</th>
      </tr>
    </thead>
    <tbody>
      {#each dashboardSettings as setting}
        <tr>
          <td>{setting[0]}</td>
          <td>{setting[1]}</td>
        </tr>
      {/each}
    </tbody>
  </table>
</section>

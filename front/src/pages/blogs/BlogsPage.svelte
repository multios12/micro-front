<script lang="ts">
  import { onMount } from 'svelte'

  import AdminHeader from '../../components/AdminHeader.svelte'
  import BlogListTable from '../../components/BlogListTable.svelte'
  import FlashMessage from '../../components/FlashMessage.svelte'
  import Pagination from '../../components/Pagination.svelte'
  import Toast from '../../components/Toast.svelte'
  import { fetchBlogList } from '../../lib/admin-api'
  import {
    blogsHeaderActions,
    blogsStatusOptions,
    buildBlogsCountLabel,
    buildBlogsPaginationItems,
    filterBlogsRows,
    mapBlogsRows,
    type BlogsPaginationItem,
    type BlogsRow,
    type BlogsFilterState,
  } from './logic'

  let loading = true
  let error = ''
  let toastOpen = false
  let toastTitle = ''
  let toastMessage = ''
  let page = 1
  let totalPages = 1
  let totalCount = 0
  let selectedStatus: BlogsFilterState['status'] = 'all'
  let searchValue = ''
  let categoryValue = 'all'
  let rows: BlogsRow[] = []
  let categories: string[] = []
  let paginationItems: BlogsPaginationItem[] = []
  let rawRows: BlogsRow[] = []
  let blogsCountLabel = ''

  async function loadBlogs(nextPage = page) {
    loading = true
    error = ''
    toastOpen = false

    try {
      page = nextPage
      const response = await fetchBlogList({
        page,
        perPage: 20,
        status: selectedStatus === 'all' ? '' : selectedStatus,
      })

      totalPages = Math.max(1, response.total_pages)
      totalCount = response.total
      rawRows = mapBlogsRows(response.items)
      categories = Array.from(new Set(rawRows.map((row) => row.category))).sort((left, right) =>
        left.localeCompare(right, 'ja'),
      )
      if (!categories.includes(categoryValue)) {
        categoryValue = 'all'
      }
      rows = filterBlogsRows(rawRows, {
        status: selectedStatus,
        searchValue,
        categoryValue,
      })
      paginationItems = buildBlogsPaginationItems(page, totalPages, (next) => loadBlogs(next))
    } catch (err) {
      toastTitle = 'エラー'
      toastMessage = err instanceof Error ? err.message : '記事一覧の読み込みに失敗しました'
      toastOpen = true
    } finally {
      loading = false
    }
  }

  const handleStatusChange = (value: BlogsFilterState['status']) => {
    selectedStatus = value
    page = 1
    loadBlogs(1)
  }

  const handleClear = () => {
    selectedStatus = 'all'
    categoryValue = 'all'
    searchValue = ''
    page = 1
    loadBlogs(1)
  }

  $: {
    rawRows
    searchValue
    categoryValue
    rows = filterBlogsRows(rawRows, {
      status: selectedStatus,
      searchValue,
      categoryValue,
    })
  }

  $: blogsCountLabel = buildBlogsCountLabel(rows.length, totalCount)

  onMount(() => {
    void loadBlogs()
  })
</script>

<svelte:head>
  <title>Blogs | micro-front</title>
</svelte:head>

  <AdminHeader title="Blogs">
    <svelte:fragment slot="actions">
    {#each blogsHeaderActions as action}
      <a
        class={action.primary ? 'admin-button admin-button-primary' : 'admin-button'}
        href={action.href}
      >
        {action.label}
      </a>
    {/each}
  </svelte:fragment>
</AdminHeader>

{#if toastOpen}
  <Toast tone="error" title={toastTitle} message={toastMessage} onClose={() => (toastOpen = false)} />
{/if}

{#if loading}
  <FlashMessage tone="success" title="読み込み中" message="管理API から記事一覧を取得しています。" />
{/if}

<section class="admin-panel">
  <div class="admin-filter-panel">
    <div class="admin-filter-group">
      <span class="admin-label">公開状態</span>
      <div class="flex flex-wrap gap-3">
        {#each blogsStatusOptions as option}
          <label class="flex items-center gap-2 text-sm text-slate-200">
            <input
              type="radio"
              name="status"
              checked={selectedStatus === option.value}
              on:change={() => handleStatusChange(option.value)}
            >
            {option.label}
          </label>
        {/each}
      </div>
    </div>

    <div class="admin-filter-group">
      <span class="admin-label">カテゴリ</span>
      <select class="admin-select" bind:value={categoryValue}>
        <option value="all">カテゴリ: すべて</option>
        {#each categories as category}
          <option value={category}>{category}</option>
        {/each}
      </select>
    </div>

    <div class="admin-filter-group">
      <input
        class="admin-input"
        type="text"
        bind:value={searchValue}
        aria-label="検索条件"
        placeholder="タイトル、概要、カテゴリで検索"
      >
      <button class="admin-button w-full justify-center" type="button" on:click={handleClear}>
        条件をクリア
      </button>
    </div>
  </div>
</section>

<section class="admin-table-wrap">
  <div class="admin-panel-head">
    <h2>記事一覧</h2>
    <span class="admin-note">{blogsCountLabel}</span>
  </div>

  <BlogListTable rows={rows} includeUpdatedAt />

  <div class="mt-4">
    <Pagination items={paginationItems} />
  </div>
</section>

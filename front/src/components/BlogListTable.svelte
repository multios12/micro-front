<script lang="ts">
  type BlogRow = {
    id: number
    title: string
    summary: string
    category: string
    status: 'public' | 'private'
    publishedAt: string
    updatedAt?: string
    href: string
  }

  export let rows: readonly BlogRow[] = []
  export let includeUpdatedAt = false
  export let compactSummary = false

  const statusClass = (status: BlogRow['status']) =>
    status === 'public' ? 'admin-chip admin-chip-success' : 'admin-chip'

  const openRow = (href: string) => {
    location.hash = href
  }
</script>

<div class="grid gap-4 lg:hidden">
  {#each rows as row}
    <button
      class="rounded-[18px] border border-white/10 bg-slate-800/90 p-4 shadow-[0_24px_64px_rgba(0,0,0,0.24)]"
      type="button"
      on:click={() => openRow(row.href)}
    >
      <div class="text-base font-semibold text-slate-100">ID: {row.id}</div>
      <div class="mt-3 grid gap-1">
        <div class="admin-label">タイトル</div>
        <div class="text-lg font-semibold text-white">{row.title}</div>
      </div>
      <div class="mt-3 grid gap-1">
        <div class="admin-label">カテゴリ</div>
        <div class="text-sm text-slate-200">{row.category}</div>
      </div>
      <div class="mt-4 flex items-center justify-between gap-3">
        <span class={statusClass(row.status)}>{row.status === 'public' ? '公開' : '非公開'}</span>
        <span class="text-sm text-slate-300">{row.publishedAt}</span>
      </div>
      {#if includeUpdatedAt && row.updatedAt}
        <div class="mt-3 text-right text-xs text-slate-500">{row.updatedAt}</div>
      {/if}
      {#if !compactSummary}
        <p class="mt-3 text-sm leading-6 text-slate-300">{row.summary}</p>
      {/if}
    </button>
  {/each}
</div>

<div class="hidden lg:block">
  <table class="admin-table">
    <thead>
      <tr>
        <th>ID</th>
        <th>タイトル</th>
        <th>カテゴリ</th>
        <th>公開状態</th>
        <th>公開日</th>
        {#if includeUpdatedAt}
          <th>更新日</th>
        {/if}
      </tr>
    </thead>
    <tbody>
      {#each rows as row}
        <!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
        <tr class="admin-table-row-link" tabindex="0" role="link" on:click={() => openRow(row.href)} on:keydown={(event) => (event.key === 'Enter' || event.key === ' ') && openRow(row.href)}>
          <td>{row.id}</td>
          <td>
            <strong class="block text-slate-100">{row.title}</strong>
            <span class="mt-1 block text-sm text-slate-400">{row.summary}</span>
          </td>
          <td>{row.category}</td>
          <td>
            <span class={statusClass(row.status)}>{row.status === 'public' ? '公開' : '非公開'}</span>
          </td>
          <td>{row.publishedAt}</td>
          {#if includeUpdatedAt}
            <td>{row.updatedAt}</td>
          {/if}
        </tr>
      {/each}
    </tbody>
  </table>
</div>

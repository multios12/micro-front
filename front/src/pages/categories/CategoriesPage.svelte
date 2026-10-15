<script lang="ts">
  import { onMount, tick } from 'svelte'
  import AdminHeader from '../../components/AdminHeader.svelte'
  import FlashMessage from '../../components/FlashMessage.svelte'
  import ConfirmDialog from '../../components/ConfirmDialog.svelte'
  import { createCategory, deleteCategory, extractValidationFields, fetchCategoryList, moveCategory, updateCategory, type CategoryApiItem } from '../../lib/admin-api'
  import { buildCategoryRows } from './logic'

  let loading = true
  let error = ''
  let categories: CategoryApiItem[] = []
  let formOpen = false
  let parent: CategoryApiItem | null = null
  let name = ''
  let sortOrder = 0
  let busy = false
  let operationError = ''
  let fields: Record<string, string> = {}
  let deleteTarget: CategoryApiItem | null = null
  let editTarget: CategoryApiItem | null = null
  let creatingWithoutParent = false

  $: rows = buildCategoryRows(categories)

  async function openCreate(nextParent: CategoryApiItem | null, withoutParent = false) {
    editTarget = null
    creatingWithoutParent = withoutParent
    parent = nextParent
    name = ''
    sortOrder = Math.max(0, ...rows.filter((row) => row.category && (row.category.parent_id === (nextParent?.id ?? null))).map((row) => row.category!.sort_order)) + 10
    fields = {}
    operationError = ''
    formOpen = true
    await tick()
    document.getElementById('category-name')?.focus()
  }

  async function openEdit(category: CategoryApiItem) {
    if (busy || loading) return
    editTarget = category
    parent = categories.find((c) => c.id === category.parent_id) ?? null
    name = category.name
    sortOrder = category.sort_order
    fields = {}
    operationError = ''
    formOpen = true
    await tick()
    document.getElementById('category-name')?.focus()
  }

  function handleRowKeydown(event: KeyboardEvent, category: CategoryApiItem) {
    if (event.target !== event.currentTarget || (event.key !== 'Enter' && event.key !== ' ')) return
    event.preventDefault()
    void openEdit(category)
  }

  async function saveCategory() {
    if (busy) return
    busy = true
    operationError = ''
    fields = {}
    try {
      if (editTarget) {
        await updateCategory(editTarget.id, { name, sort_order: sortOrder })
      } else {
        await createCategory({ parent_id: parent?.id ?? null, is_heading: parent === null && !creatingWithoutParent, name, sort_order: sortOrder })
      }
      formOpen = false
      await loadCategories()
    } catch (err) {
      operationError = err instanceof Error ? err.message : 'カテゴリの保存に失敗しました。'
      fields = extractValidationFields(err) ?? {}
    } finally {
      busy = false
    }
  }

  async function confirmDelete() {
    if (busy || !deleteTarget) return
    const target = deleteTarget
    deleteTarget = null
    busy = true
    operationError = ''
    try {
      await deleteCategory(target.id)
      formOpen = false
      await loadCategories()
    } catch (err) {
      operationError = err instanceof Error ? err.message : 'カテゴリの削除に失敗しました。'
    } finally {
      busy = false
    }
  }

  async function move(category: CategoryApiItem, direction: 'up' | 'down') {
    if (busy) return
    busy = true
    operationError = ''
    try {
      await moveCategory(category.id, direction)
      formOpen = false
      await loadCategories()
    } catch (err) {
      operationError = err instanceof Error ? err.message : '表示順の変更に失敗しました。'
    } finally {
      busy = false
    }
  }

  async function loadCategories() {
    loading = true
    error = ''
    try {
      const response = await fetchCategoryList()
      categories = response.items
    } catch (err) {
      error = err instanceof Error ? err.message : 'カテゴリ一覧の取得に失敗しました。'
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void loadCategories()
  })
</script>

<svelte:head>
  <title>Categories | micro-front</title>
</svelte:head>

<AdminHeader title="Categories" />

<section class="admin-table-wrap" aria-labelledby="categories-heading" aria-busy={loading}>
  <div class="admin-panel-head">
    <h2 id="categories-heading">カテゴリ一覧</h2>
    <div class="flex flex-wrap items-center justify-end gap-3">
      {#if !loading && !error}
        <span class="admin-note">{categories.length}件</span>
      {/if}
      <button class="admin-button admin-button-primary" type="button" disabled={loading || busy || !!error} on:click={() => openCreate(null)}>親見出しを追加</button>
      <button class="admin-button" type="button" disabled={loading || busy || !!error} on:click={() => openCreate(null, true)}>通常カテゴリを追加</button>
    </div>
  </div>
  {#if operationError}
    <div class="mb-4" role="alert">
      <FlashMessage tone="error" title="操作に失敗しました" message={operationError} />
    </div>
  {/if}
  {#if formOpen}
    <form class="admin-panel mb-4" on:submit|preventDefault={saveCategory}>
      <h3 class="mb-4 font-semibold">{editTarget ? `「${editTarget.name}」を編集` : parent ? `「${parent.name}」にカテゴリを追加` : creatingWithoutParent ? '通常カテゴリを追加' : '親見出しを追加'}</h3>
      <div class="admin-stack">
        <div class="admin-field">
          <label class="admin-label" for="category-name">カテゴリ名</label>
          <input id="category-name" class="admin-input" class:admin-input-error={!!fields.name} bind:value={name} required disabled={busy} aria-invalid={!!fields.name} aria-describedby={fields.name ? 'category-name-error' : undefined} />
          {#if fields.name}<p id="category-name-error" class="text-sm text-rose-300">{fields.name}</p>{/if}
        </div>
        <div class="flex flex-wrap gap-2">
          <button class="admin-button" type="button" disabled={busy} on:click={() => { formOpen = false; operationError = '' }}>キャンセル</button>
          <button class="admin-button admin-button-primary" type="submit" disabled={busy}>{busy ? '保存中...' : editTarget ? '保存する' : '追加する'}</button>
          {#if editTarget}
            <button class="admin-button admin-button-danger ml-auto" type="button" disabled={busy} on:click={() => { operationError = ''; deleteTarget = editTarget }}>削除</button>
          {/if}
        </div>
      </div>
    </form>
  {/if}
  {#if loading}
    <p class="admin-note" role="status">カテゴリ一覧を読み込んでいます。</p>
  {:else if error}
    <div role="alert">
      <FlashMessage tone="error" title="カテゴリ一覧の取得に失敗しました" message={error} />
    </div>
    <button class="admin-button mt-4" type="button" on:click={loadCategories}>再読み込み</button>
  {:else if categories.length === 0}
    <p class="admin-note">カテゴリはまだ登録されていません。</p>
  {:else}
    <div class="overflow-x-auto">
      <table class="admin-table">
        <thead>
          <tr>
            <th scope="col">カテゴリ</th>
            <th scope="col" class="text-right">操作</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as row (row.key)}
            {@const category = row.category}
            {#if category}
            <!-- svelte-ignore a11y_no_noninteractive_element_to_interactive_role -->
            <tr
              class="admin-table-row-link"
              role="button"
              tabindex={busy ? -1 : 0}
              aria-label={`${category.name}を編集`}
              aria-disabled={busy}
              on:click={(event) => { if (!(event.target instanceof Element && event.target.closest('button'))) void openEdit(category) }}
              on:keydown={(event) => handleRowKeydown(event, category)}
            >
              <td>
                {#if row.isHeading}
                  <strong class="text-slate-100">{category.name}</strong>
                {:else if category.parent_id === null}
                  <span>{category.name}</span>
                {:else}
                  <span class="flex items-center gap-2 pl-4">
                    <span class="text-slate-500" aria-hidden="true">└</span>
                    <span>{category.name}</span>
                  </span>
                {/if}
              </td>
              <td>
                <div class="flex justify-end gap-2 whitespace-nowrap">
                  <button class="admin-button disabled:cursor-not-allowed disabled:opacity-40" type="button" disabled={busy || !row.canMoveUp} aria-label={`${category.name}を上へ移動`} on:click|stopPropagation={() => move(category, 'up')}>上へ</button>
                  <button class="admin-button disabled:cursor-not-allowed disabled:opacity-40" type="button" disabled={busy || !row.canMoveDown} aria-label={`${category.name}を下へ移動`} on:click|stopPropagation={() => move(category, 'down')}>下へ</button>
                  {#if row.isHeading}
                    <button class="admin-button admin-button-secondary" type="button" disabled={busy} aria-label={`${category.name}にカテゴリを追加`} on:click|stopPropagation={() => openCreate(category)}>カテゴリを追加</button>
                  {/if}
                </div>
              </td>
            </tr>
            {/if}
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</section>

<ConfirmDialog
  open={deleteTarget !== null}
  title={rows.some((row) => row.category?.id === deleteTarget?.id && row.isHeading) ? '親見出しの削除' : '小見出しの削除'}
  message={`「${deleteTarget?.name ?? ''}」を削除しますか？`}
  onConfirm={confirmDelete}
  onCancel={() => { deleteTarget = null }}
/>

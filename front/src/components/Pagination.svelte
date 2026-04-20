<script lang="ts">
  export let items: readonly { label: string; href?: string; active?: boolean; onClick?: () => void }[] = []
</script>

<nav class="admin-pagination" aria-label="ページネーション">
  {#each items as item}
    {#if item.onClick}
      <button
        class={`admin-pagination-link ${item.active ? 'admin-pagination-link-active' : ''}`}
        type="button"
        on:click={item.onClick}
      >
        {item.label}
      </button>
    {:else}
      <a
        class={`admin-pagination-link ${item.active ? 'admin-pagination-link-active' : ''}`}
        href={item.href}
      >
        {item.label}
      </a>
    {/if}
  {/each}
</nav>

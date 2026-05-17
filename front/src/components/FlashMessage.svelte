<script lang="ts">
  export let tone: 'success' | 'warning' | 'error' = 'success'
  export let title = ''
  export let message = ''

  const toneClass = {
    success: 'admin-flash-success',
    warning: 'admin-flash-warning',
    error: 'admin-flash-error',
  }[tone]
</script>

<div class={`admin-flash ${toneClass}`}>
  <div class="text-sm font-semibold">{title}</div>
  {#if message}
    <p class="mt-2 text-sm leading-6 opacity-90">{message}</p>
  {/if}
</div>

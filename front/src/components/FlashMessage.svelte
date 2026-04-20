<script lang="ts">
  export let tone: 'success' | 'warning' | 'error' = 'success'
  export let title = ''
  export let message = ''

  const toneClass = {
    success: 'border-emerald-400/20 bg-emerald-400/10 text-emerald-100',
    warning: 'border-amber-400/20 bg-amber-400/10 text-amber-100',
    error: 'border-rose-400/20 bg-rose-400/10 text-rose-100',
  }[tone]
</script>

<div class={`rounded-2xl border p-4 ${toneClass}`}>
  <div class="text-sm font-semibold">{title}</div>
  {#if message}
    <p class="mt-2 text-sm leading-6 opacity-90">{message}</p>
  {/if}
</div>

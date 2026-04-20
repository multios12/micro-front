<script lang="ts">
  import { onMount } from "svelte";

  export let open = true;
  export let tone: 'success' | 'warning' | 'error' = 'error';
  export let title = '';
  export let message = '';
  export let onClose: (() => void) | undefined = undefined;
  export let duration = 10000;

  const toneClass = {
    success: 'border-emerald-400/30 bg-emerald-500/15 text-emerald-50',
    warning: 'border-amber-400/30 bg-amber-500/15 text-amber-50',
    error: 'border-rose-400/30 bg-rose-500/15 text-rose-50',
  }[tone];

  let timer: ReturnType<typeof setTimeout> | null = null;
  let mounted = false;

  const clearTimer = () => {
    if (timer !== null) {
      clearTimeout(timer);
      timer = null;
    }
  };

  const startTimer = () => {
    clearTimer();
    if (!open || duration <= 0) {
      return;
    }
    timer = setTimeout(() => {
      onClose?.();
    }, duration);
  };

  onMount(() => {
    mounted = true;
    startTimer();
    return clearTimer;
  });

  $: if (mounted) {
    if (open) {
      startTimer();
    } else {
      clearTimer();
    }
  }
</script>

{#if open}
  <button
    class={`fixed right-4 top-4 z-50 w-[min(28rem,calc(100vw-1rem))] cursor-pointer rounded-2xl border p-4 text-left shadow-[0_24px_64px_rgba(0,0,0,0.45)] backdrop-blur-md transition hover:scale-[1.01] ${toneClass}`}
    type="button"
    aria-label={title || message || "通知"}
    on:click={() => onClose?.()}
  >
    <div class="flex items-start gap-3">
      <div class="min-w-0 flex-1">
        {#if title}
          <div class="text-sm font-semibold leading-5">{title}</div>
        {/if}
        {#if message}
          <p class="mt-2 text-sm leading-6 opacity-95">{message}</p>
        {/if}
      </div>
    </div>
  </button>
{/if}

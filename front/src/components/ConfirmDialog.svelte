<script lang="ts">
  export let open = false
  export let title = '削除確認'
  export let message = ''
  export let confirmLabel = '削除する'
  export let cancelLabel = 'キャンセル'
  export let onConfirm: (() => void) | undefined
  export let onCancel: (() => void) | undefined
</script>

{#if open}
  <div class="admin-modal">
    <div class="admin-modal-card">
      <div class="admin-panel-head">
        <h2>{title}</h2>
      </div>
      {#if message}
        <p class="admin-note">{message}</p>
      {/if}
      <div class="mt-4 flex flex-wrap gap-2 border-t border-white/10 pt-4">
        <button class="admin-button" type="button" on:click={onCancel}>{cancelLabel}</button>
        <button class="admin-button admin-button-danger" type="button" on:click={onConfirm}>{confirmLabel}</button>
      </div>
    </div>
  </div>
{/if}

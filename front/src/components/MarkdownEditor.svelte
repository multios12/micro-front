<script lang="ts">
  export let id = ''
  export let label = '本文'
  export let value = ''
  export let placeholder = ''
  export let error = ''
</script>

<div class="admin-field">
  <label class="admin-label" for={id}>{label}</label>
  <textarea
    id={id}
    class="admin-textarea"
    class:admin-textarea-error={Boolean(error)}
    bind:value
    placeholder={placeholder}
  ></textarea>
  {#if error}
    <p class="admin-error-message">{error}</p>
  {/if}
</div>

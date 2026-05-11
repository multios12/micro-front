<script lang="ts">
  import type { TitleImageTemplateApiItem } from "../lib/admin-api";

  export let templates: TitleImageTemplateApiItem[] = [];
  export let value = "diary";
  export let error = "";
</script>

<div class="title-image-template-list">
  {#each templates as template}
    <label class="title-image-template-option" class:is-selected={value === template.id}>
      <input type="radio" bind:group={value} value={template.id} />
      <span class="title-image-template-label">{template.label}</span>
      <span class="title-image-template-description">{template.description}</span>
    </label>
  {/each}
</div>
{#if error}
  <p class="admin-error-message">{error}</p>
{/if}

<style>
  .title-image-template-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 8px;
  }

  .title-image-template-option {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 4px 8px;
    align-items: start;
    padding: 10px;
    border: 1px solid var(--admin-border, #d8d8d8);
    border-radius: 6px;
    cursor: pointer;
  }

  .title-image-template-option.is-selected {
    border-color: var(--admin-accent, #2563eb);
    background: rgba(37, 99, 235, 0.06);
  }

  .title-image-template-label {
    font-weight: 700;
    line-height: 1.25;
  }

  .title-image-template-description {
    grid-column: 2;
    color: var(--admin-muted, #666);
    font-size: 0.82rem;
    line-height: 1.45;
  }
</style>

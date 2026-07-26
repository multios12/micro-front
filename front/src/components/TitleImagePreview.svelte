<script lang="ts">
  export let svg = "";
  export let loading = false;

  $: previewUrl = svg
    ? `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
    : "";
</script>

<div class="title-image-preview">
  {#if previewUrl}
    <img src={previewUrl} alt="タイトル画像プレビュー" />
  {:else}
    <div class="title-image-preview-empty">
      {loading ? "生成中..." : "プレビューなし"}
    </div>
  {/if}
</div>

<style>
  .title-image-preview {
    width: 100%;
    aspect-ratio: 16 / 9;
    overflow: hidden;
    border: 1px solid var(--admin-border, #d8d8d8);
    border-radius: 6px;
    background: #f5f5f5;
  }

  .title-image-preview img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .title-image-preview-empty {
    display: grid;
    place-items: center;
    width: 100%;
    height: 100%;
    color: var(--admin-muted, #666);
    font-size: 0.92rem;
  }
</style>

<script lang="ts">
  import { tick } from "svelte";

  type ImageItem = {
    id: number;
    label: string;
    imageUrl: string;
    displayUrl?: string;
    altText?: string;
  };

  export let title = "画像";
  export let note = "";
  export let buttonLabel = "画像を選択";
  export let insertButtonLabel = "画像挿入";
  export let items: readonly ImageItem[] = [];
  export let onSelectFile: (file: File) => void | Promise<void> = () => {};
  export let onCopyItem: (item: ImageItem) => void | Promise<void> = () => {};
  export let onInsertItem: (item: ImageItem) => void | Promise<void> = () => {};
  export let onDeleteItem: (item: ImageItem) => void | Promise<void> = () => {};

  let fileInput: HTMLInputElement | null = null;
  let opening = false;

  const openFilePicker = async () => {
    if (opening) {
      return;
    }
    opening = true;
    fileInput?.click();
    await tick();
    opening = false;
  };

  const handleFileChange = async (event: Event) => {
    const input = event.currentTarget as HTMLInputElement;
    const file = input.files?.[0];
    input.value = "";
    if (!file) {
      return;
    }
    await onSelectFile(file);
  };
</script>

<section class="admin-dropzone">
  <div class="admin-panel-head">
    <h2>{title}</h2>
  </div>

  <div class="flex flex-wrap items-center gap-2">
    <button
      class="admin-button admin-button-secondary"
      type="button"
      on:click={openFilePicker}
    >
      {buttonLabel}
    </button>
    {#if note}
      <span class="admin-note">{note}</span>
    {/if}
  </div>

  <input
    bind:this={fileInput}
    class="hidden"
    type="file"
    accept="image/*"
    on:change={handleFileChange}
  />

  {#if items.length > 0}
    <div class="grid gap-4 md:grid-cols-2">
      {#each items as item}
        <article class="rounded-[18px] border border-white/10 bg-white/5 p-4">
          <div
            class="overflow-hidden rounded-xl border border-white/10 bg-slate-950/30"
          >
            <img
              class="aspect-[16/9] w-full object-cover"
              src={item.displayUrl ?? item.imageUrl}
              alt={item.altText ?? `image-${item.id}`}
              loading="lazy"
            />
          </div>
          <div class="mt-4 flex flex-wrap gap-2">
            <button
              class="admin-button admin-button-secondary"
              type="button"
              on:click={() => onInsertItem(item)}
            >
              {insertButtonLabel}
            </button>
            <button
              class="admin-button"
              type="button"
              on:click={() => onCopyItem(item)}
            >
              {item.label}
            </button>
            <button
              class="admin-button admin-button-danger"
              type="button"
              on:click={() => onDeleteItem(item)}
            >
              削除
            </button>
          </div>
        </article>
      {/each}
    </div>
  {/if}
</section>

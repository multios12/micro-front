<script lang="ts">
  import { faArrowLeft, faImage } from "@fortawesome/free-solid-svg-icons";
  import Icon from "./Icon.svelte";

  interface Props {
    error?: string;
    uploading?: boolean;
    input?: HTMLInputElement | null;
    onClose?: () => void;
    onOpenPicker?: () => void;
    onSelectFile?: (event: Event) => void | Promise<void>;
  }

  let {
    error = "",
    uploading = false,
    input = $bindable<HTMLInputElement | null>(null),
    onClose,
    onOpenPicker,
    onSelectFile,
  }: Props = $props();
</script>

<div
  class="md-image-modal-backdrop"
  role="button"
  tabindex="0"
  onclick={(event) => {
    if (event.target === event.currentTarget) {
      onClose?.();
    }
  }}
  onkeydown={(event) => {
    if (event.target !== event.currentTarget) {
      return;
    }
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onClose?.();
    }
    if (event.key === "Escape") {
      event.preventDefault();
      onClose?.();
    }
  }}
>
  <div class="md-image-modal">
    {#if error !== ""}
      <div class="notification is-danger mb-3">{error}</div>
    {/if}
    <div class="md-image-actions">
      <button
        class="button is-light"
        type="button"
        aria-label="back"
        disabled={uploading}
        onclick={onClose}
      >
        <span class="icon">
          <Icon icon={faArrowLeft} />
        </span>
      </button>
      <button
        class="button is-link md-image-select-button"
        type="button"
        aria-label="add image"
        disabled={uploading}
        onclick={onOpenPicker}
      >
        <span class="icon">
          <Icon icon={faImage} />
        </span>
        <span>画像選択</span>
      </button>
      {#if uploading}
        <button class="button is-light is-loading" aria-label="uploading"
        ></button>
      {/if}
    </div>
    <input
      class="is-hidden"
      type="file"
      accept="image/*"
      bind:this={input}
      onchange={onSelectFile}
    />
  </div>
</div>

<style>
  :root {
    --modal-gackground: rgba(4, 8, 20, 0.72);
    --modal-radius: 0.75rem;
  }
  .md-image-modal-backdrop {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    background: color-mix(in srgb, black 16%, transparent);
    z-index: 1200;
  }

  .md-image-modal {
    width: min(34rem, 100%);
    padding: 1rem;
    border-radius: var(--modal-radius);
    background: var(--modal-gackground);
    box-shadow: 0 1.2rem 3rem color-mix(in srgb, black 18%, transparent);
  }

  .md-image-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .md-image-select-button {
    flex: 1 1 auto;
    justify-content: center;
  }

  @media screen and (max-width: 768px) {
    .md-image-modal {
      padding: 0.85rem;
    }

    .md-image-actions {
      gap: 0.5rem;
    }
  }
</style>

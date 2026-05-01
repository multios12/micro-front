<script lang="ts">
  import { Link2, Trash2 } from "lucide-svelte";
  import Icon from "./Icon.svelte";

  interface Props {
    value?: string;
    input?: HTMLInputElement | null;
    onApply?: () => void;
    onRemove?: () => void;
    onClose?: () => void;
  }

  let {
    value = $bindable(""),
    input = $bindable<HTMLInputElement | null>(null),
    onApply,
    onRemove,
    onClose,
  }: Props = $props();
</script>

<div
  class="md-link-modal-backdrop"
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
  <div class="md-link-modal">
    <div class="field has-addons">
      <div class="control is-expanded">
        <input
          class="input"
          type="text"
          bind:this={input}
          bind:value
          placeholder="https://example.com"
          onkeydown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              onApply?.();
            }
            if (event.key === "Escape") {
              event.preventDefault();
              onClose?.();
            }
          }}
        />
      </div>
      <div class="control">
        <button
          class="button is-link"
          type="button"
          aria-label="apply link"
          onclick={onApply}
        >
          <span class="icon"><Icon icon={Link2} /></span>
        </button>
      </div>
      <div class="control">
        <button
          class="button is-light"
          type="button"
          aria-label="remove link"
          onclick={onRemove}
        >
          <span class="icon"><Icon icon={Trash2} /></span>
        </button>
      </div>
    </div>
  </div>
</div>

<style>
  .md-link-modal-backdrop {
    position: fixed;
    inset: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    background: color-mix(in srgb, black 16%, transparent);
    z-index: 1200;
  }

  .md-link-modal {
    width: min(34rem, 100%);
    padding: 1rem;
    border-radius: var(--modal-radius);
    background: var(--modal-gackground);
    box-shadow: 0 1.2rem 3rem color-mix(in srgb, black 18%, transparent);
  }

  .md-link-modal .field.has-addons {
    display: flex;
    flex-wrap: nowrap;
    align-items: center;
  }

  .md-link-modal .field.has-addons .control.is-expanded {
    flex: 1 1 auto;
    min-width: 0;
  }

  .md-link-modal .field.has-addons .control:not(.is-expanded) {
    flex: 0 0 auto;
  }

  .md-link-modal .field.has-addons .input {
    min-width: 0;
  }

  @media screen and (max-width: 768px) {
    .md-link-modal {
      padding: 0.85rem;
    }
  }
</style>

<script lang="ts">
  import type { IconDefinition } from "@fortawesome/fontawesome-svg-core";
  import {
    faArrowsRotate,
    faBold,
    faChevronDown,
    faCode,
    faGripLines,
    faItalic,
    faLink,
    faListOl,
    faListUl,
    faQuoteLeft,
    faStrikethrough,
  } from "@fortawesome/free-solid-svg-icons";
  import Icon from "./Icon.svelte";
  import type { ParagraphKind } from "./types";

  export const paragraphs = [
    { key: "normal", value: "本文", icon: faGripLines },
    { key: "h1", value: "見出し1", badge: "H1" },
    { key: "h2", value: "見出し2", badge: "H2" },
    { key: "h3", value: "見出し3", badge: "H3" },
    { key: "ol", value: "番号リスト", icon: faListOl },
    { key: "ul", value: "段落リスト", icon: faListUl },
    { key: "code", value: "コード", icon: faCode },
    { key: "quote", value: "引用", icon: faQuoteLeft },
  ] satisfies Array<{
    key: ParagraphKind;
    value: string;
    icon?: IconDefinition;
    badge?: string;
  }>;

  interface Props {
    value?: ParagraphKind;
    bold?: boolean;
    italic?: boolean;
    link?: boolean;
    strike?: boolean;
    onChange?: (value: ParagraphKind) => void;
    onBold?: () => void;
    onCarryOver?: () => void;
    onItalic?: () => void;
    onLink?: () => void;
    onStrike?: () => void;
  }

  let {
    value = $bindable("normal"),
    bold = false,
    italic = false,
    link = false,
    strike = false,
    onChange,
    onBold,
    onCarryOver,
    onItalic,
    onLink,
    onStrike,
  }: Props = $props();
  let isParagraphMenuOpen = $state(false);

  const currentParagraph = $derived(
    paragraphs.find((item) => item.key === value) ?? paragraphs[0],
  );

  const handleChange = (nextValue: ParagraphKind) => {
    value = nextValue;
    isParagraphMenuOpen = false;
    onChange?.(nextValue);
  };

  const closeParagraphMenu = () => {
    isParagraphMenuOpen = false;
  };
</script>

<div id="toolbar" class="md-toolbar">
  <div class="md-paragraph-select">
    {#if isParagraphMenuOpen}
      <button
        class="md-paragraph-backdrop"
        type="button"
        aria-label="close paragraph menu"
        onclick={closeParagraphMenu}
      ></button>
    {/if}

    <button
      class="button md-paragraph-trigger"
      type="button"
      aria-haspopup="menu"
      aria-expanded={isParagraphMenuOpen}
      onclick={() => (isParagraphMenuOpen = !isParagraphMenuOpen)}
    >
      {#key currentParagraph.key}
        {#if currentParagraph.icon}
          <span class="icon md-paragraph-icon-slot">
            <Icon icon={currentParagraph.icon} />
          </span>
        {:else}
          <span class="icon md-paragraph-icon-slot">
            <span class="md-paragraph-badge">{currentParagraph.badge}</span>
          </span>
        {/if}
      {/key}
      <span class="icon is-small">
        <Icon icon={faChevronDown} />
      </span>
    </button>

    {#if isParagraphMenuOpen}
      <div class="md-paragraph-menu" role="menu">
        {#each paragraphs as item}
          <button
            class="button is-ghost md-paragraph-option"
            class:is-active={item.key === value}
            type="button"
            role="menuitemradio"
            aria-checked={item.key === value}
            onclick={() => handleChange(item.key)}
          >
            {#if item.icon}
              <span class="icon md-paragraph-icon-slot">
                <Icon icon={item.icon} />
              </span>
            {:else}
              <span class="icon md-paragraph-icon-slot">
                <span class="md-paragraph-badge">{item.badge}</span>
              </span>
            {/if}
            <span>{item.value}</span>
          </button>
        {/each}
      </div>
    {/if}
  </div>
  <button
    class="button is-ghost md-toolbar-button"
    class:is-active={bold}
    type="button"
    aria-label="bold"
    onclick={onBold}
  >
    <Icon icon={faBold} />
  </button>
  <button
    class="button is-ghost md-toolbar-button"
    class:is-active={italic}
    type="button"
    aria-label="italic"
    onclick={onItalic}
  >
    <Icon icon={faItalic} />
  </button>
  <button
    class="button is-ghost md-toolbar-button"
    class:is-active={strike}
    type="button"
    aria-label="strike"
    onclick={onStrike}
  >
    <Icon icon={faStrikethrough} />
  </button>
  <button
    class="button is-ghost md-toolbar-button"
    type="button"
    aria-label="carry over marker"
    title="次回グループ追加へ引き継ぐ位置を挿入"
    onclick={onCarryOver}
  >
    <Icon icon={faArrowsRotate} />
  </button>
  <button
    class="button is-ghost md-toolbar-button"
    class:is-active={link}
    type="button"
    aria-label="link"
    onclick={onLink}
  >
    <Icon icon={faLink} />
  </button>
</div>

<style>
  .md-toolbar {
    display: flex;
    align-items: center;
    flex-wrap: nowrap;
    gap: 0.5rem;
    overflow: visible;
    padding: 0.55rem 0.65rem;
    border-bottom: 1px solid color-mix(in srgb, #64748b 28%, transparent);
    background: linear-gradient(
        180deg,
        color-mix(in srgb, white 8%, transparent),
        color-mix(in srgb, white 4%, transparent)
      ),
      #111827;
  }

  .md-paragraph-select {
    position: relative;
  }

  .md-paragraph-backdrop {
    position: fixed;
    inset: 0;
    z-index: 9;
    border: none;
    background: transparent;
    padding: 0;
  }

  .md-paragraph-trigger {
    min-width: 3.8rem;
    justify-content: space-between;
    gap: 0.35rem;
    color: #f6f7fb;
    border-color: color-mix(in srgb, white 14%, transparent);
    background: #1b2030;
  }

  .md-paragraph-trigger:hover,
  .md-paragraph-trigger:focus-visible {
    color: #ffffff;
    border-color: color-mix(in srgb, #93c5fd 38%, transparent);
    background: #22314a;
  }

  .md-paragraph-menu {
    position: absolute;
    top: calc(100% + 0.35rem);
    left: 0;
    z-index: 10;
    display: flex;
    flex-direction: column;
    min-width: 13rem;
    padding: 0.35rem;
    border: 1px solid color-mix(in srgb, white 12%, transparent);
    border-radius: var(--modal-radius);
    background: #161b28;
    box-shadow: 0 0.85rem 2rem color-mix(in srgb, black 28%, transparent);
  }

  .md-paragraph-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 1.25rem;
    height: 1.25rem;
    color: #f6f7fb;
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.04em;
    line-height: 1;
  }

  .md-paragraph-icon-slot {
    flex: 0 0 1.5rem;
    width: 1.5rem;
    justify-content: center;
    margin-right: 0.1rem;
  }

  .md-paragraph-option {
    justify-content: flex-start;
    gap: 0.45rem;
    color: #eef2ff;
  }

  .md-paragraph-option.is-active {
    color: #8ec5ff;
    background: color-mix(in srgb, #2f7df4 18%, transparent);
  }

  .md-toolbar-button {
    flex: 0 0 auto;
    min-width: 2.25rem;
    min-height: 2.2rem;
    padding: 0;
    border-radius: 0.65rem;
  }

  .md-toolbar-button:hover,
  .md-toolbar-button:focus-visible {
    color: #ffffff;
    background: #22314a;
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, #93c5fd 20%, transparent),
      0 0.2rem 0.6rem color-mix(in srgb, #0f172a 24%, transparent);
  }

  .md-toolbar-button.is-active {
    color: #e0f2fe;
    background: #274765;
    box-shadow:
      inset 0 0 0 1px color-mix(in srgb, #93c5fd 24%, transparent),
      0 0.25rem 0.65rem color-mix(in srgb, #0f172a 30%, transparent);
  }

  @media screen and (max-width: 768px) {
    .md-toolbar {
      gap: 0.35rem;
    }

    .md-paragraph-trigger {
      min-width: 3.5rem;
    }
  }
</style>

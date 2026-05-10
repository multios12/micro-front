<script lang="ts">
  import ImageModal from "./ImageModal.svelte";
  import LinkModal from "./LinkModal.svelte";
  import Toolbar from "./Toolbar.svelte";
  import {
    applyCodeBlock,
    applyLineFormat,
    getNextCursorOffset,
    inlineMarkers,
    toImageAltText,
  } from "./markdown";
  import {
    detectParagraph,
    getInlineRange,
    getLineBounds,
    getLinkRange,
  } from "./selection";
  import type { InlineKind, ParagraphKind } from "./types";

  const carryOverMarker = "----ここまで前回内容で置換";

  interface Props {
    id?: string;
    value?: string;
    error?: string;
    imageUploadPath?: string;
    onTextChange?: (value: string) => void;
    onImageUploaded?: (imageUrl: string) => void | Promise<void>;
  }

  let {
    id = "",
    value = $bindable(""),
    error = "",
    imageUploadPath = "",
    onTextChange,
    onImageUploaded,
  }: Props = $props();
  let textarea = $state<HTMLTextAreaElement | null>(null);
  let paragraph = $state<ParagraphKind>("normal");
  let isBold = $state(false);
  let isItalic = $state(false);
  let isLink = $state(false);
  let isStrike = $state(false);
  let isLinkModalOpen = $state(false);
  let linkValue = $state("");
  let linkInput = $state<HTMLInputElement | null>(null);
  let pendingLinkSelection = $state({ start: 0, end: 0 });
  let isImageModalOpen = $state(false);
  let imageInput = $state<HTMLInputElement | null>(null);
  let imageError = $state("");
  let isImageUploading = $state(false);
  let pendingImageCursor = $state(0);

  const handleInput = (event: Event) => {
    const target = event.currentTarget as HTMLTextAreaElement;
    onTextChange?.(target.value);
    syncToolbarState(target);
  };

  const handleKeydown = (event: KeyboardEvent) => {
    if (textarea === null) {
      return;
    }
    if (!event.metaKey && !event.ctrlKey) {
      return;
    }

    const key = event.key.toLowerCase();
    switch (key) {
      case "b":
        event.preventDefault();
        applyInline("bold");
        return;
      case "i":
        event.preventDefault();
        applyInline("italic");
        return;
      case "k":
        event.preventDefault();
        openLinkModal();
        return;
      case "x":
        if (event.shiftKey) {
          event.preventDefault();
          applyInline("strike");
        }
        return;
      case "1":
        if (event.altKey) {
          event.preventDefault();
          applyParagraph("h1");
        }
        return;
      case "2":
        if (event.altKey) {
          event.preventDefault();
          applyParagraph("h2");
        }
        return;
      case "3":
        if (event.altKey) {
          event.preventDefault();
          applyParagraph("h3");
        }
        return;
      case "7":
        if (event.shiftKey) {
          event.preventDefault();
          applyParagraph("ol");
        }
        return;
      case "8":
        if (event.shiftKey) {
          event.preventDefault();
          applyParagraph("ul");
        }
        return;
      case "9":
        if (event.shiftKey) {
          event.preventDefault();
          applyParagraph("quote");
        }
        return;
      case "\\":
        if (event.shiftKey) {
          event.preventDefault();
          applyParagraph("code");
        }
        return;
      case "0":
        if (event.altKey) {
          event.preventDefault();
          applyParagraph("normal");
        }
        return;
      default:
        return;
    }
  };

  const updateValue = (nextValue: string) => {
    value = nextValue;
    onTextChange?.(nextValue);
  };

  const applyParagraph = (kind: ParagraphKind) => {
    if (textarea === null) {
      return;
    }

    const rawValue = textarea.value;
    const selectionStart = textarea.selectionStart;
    const { lineStart, lineEnd } = getLineBounds(rawValue, selectionStart);
    const currentLine = rawValue.slice(lineStart, lineEnd);
    const nextBlock =
      kind === "code"
        ? applyCodeBlock(rawValue, currentLine, lineStart, lineEnd)
        : applyLineFormat(currentLine, kind);
    const nextValue =
      rawValue.slice(0, lineStart) + nextBlock + rawValue.slice(lineEnd);
    const cursorOffset = getNextCursorOffset(
      currentLine,
      nextBlock,
      selectionStart - lineStart,
    );

    updateValue(nextValue);
    paragraph = kind;

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      const nextCursor = lineStart + cursorOffset;
      textarea.setSelectionRange(nextCursor, nextCursor);
      syncToolbarState(textarea);
    });
  };

  const applyInline = (kind: InlineKind) => {
    if (textarea === null) {
      return;
    }

    const marker = inlineMarkers[kind];
    const rawValue = textarea.value;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const range = getInlineRange(rawValue, start, end, marker);
    let nextValue = rawValue;
    let nextStart = start;
    let nextEnd = end;

    if (range !== null) {
      nextValue =
        rawValue.slice(0, range.openStart) +
        rawValue.slice(range.contentStart, range.contentEnd) +
        rawValue.slice(range.closeEnd);
      nextStart = range.contentStart - marker.length;
      nextEnd = range.contentEnd - marker.length;
    } else if (start !== end) {
      nextValue =
        rawValue.slice(0, start) +
        marker +
        rawValue.slice(start, end) +
        marker +
        rawValue.slice(end);
      nextStart = start + marker.length;
      nextEnd = end + marker.length;
    } else {
      nextValue =
        rawValue.slice(0, start) + marker + marker + rawValue.slice(end);
      nextStart = start + marker.length;
      nextEnd = nextStart;
    }

    updateValue(nextValue);

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(nextStart, nextEnd);
      syncToolbarState(textarea);
    });
  };

  const openLinkModal = () => {
    if (textarea === null) {
      return;
    }

    pendingLinkSelection = {
      start: textarea.selectionStart,
      end: textarea.selectionEnd,
    };

    const currentLink = getLinkRange(
      textarea.value,
      textarea.selectionStart,
      textarea.selectionEnd,
    );
    linkValue =
      currentLink === null
        ? ""
        : textarea.value.slice(currentLink.urlStart, currentLink.urlEnd);
    isLinkModalOpen = true;

    requestAnimationFrame(() => {
      linkInput?.focus();
      linkInput?.select();
    });
  };

  const openImageModal = () => {
    if (textarea === null) {
      return;
    }

    pendingImageCursor = textarea.selectionStart;
    imageError = "";
    isImageModalOpen = true;
  };

  const closeImageModal = () => {
    if (isImageUploading) {
      return;
    }

    isImageModalOpen = false;
    imageError = "";
    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(pendingImageCursor, pendingImageCursor);
      syncToolbarState(textarea);
    });
  };

  const openImagePicker = () => {
    imageInput?.click();
  };

  const onImageSelected = async (event: Event) => {
    const target = event.currentTarget as HTMLInputElement;
    const file = target.files?.item(0);
    target.value = "";

    if (file == null) {
      return;
    }
    if (imageUploadPath.trim() === "") {
      imageError = "画像アップロード先が設定されていません";
      return;
    }

    isImageUploading = true;
    imageError = "";

    try {
      const data = new FormData();
      data.append("file", file);
      const response = await fetch(imageUploadPath, {
        method: "post",
        body: data,
      });
      if (!response.ok) {
        imageError = await response.text();
        return;
      }

      const imageUrl = await readImageUploadUrl(response);
      insertImageMarkdown(imageUrl, file.name);
      await onImageUploaded?.(imageUrl);
      isImageModalOpen = false;
    } catch {
      imageError = "画像をアップロードできませんでした";
    } finally {
      isImageUploading = false;
    }
  };

  const readImageUploadUrl = async (response: Response) => {
    const contentType = response.headers.get("content-type") ?? "";
    if (contentType.includes("application/json")) {
      const payload = await response.json().catch(() => null);
      if (payload && typeof payload === "object") {
        const url = (payload as { url?: unknown }).url;
        if (typeof url === "string" && url.trim() !== "") {
          return url;
        }
      }

      throw new Error("Image upload response does not include url");
    }

    return response.text();
  };

  const closeLinkModal = () => {
    isLinkModalOpen = false;
    linkValue = "";
    requestAnimationFrame(() => {
      textarea?.focus();
      if (textarea !== null) {
        textarea.setSelectionRange(
          pendingLinkSelection.start,
          pendingLinkSelection.end,
        );
        syncToolbarState(textarea);
      }
    });
  };

  const applyLink = () => {
    if (textarea === null) {
      return;
    }

    const url = linkValue.trim();
    if (url === "") {
      closeLinkModal();
      return;
    }

    const rawValue = textarea.value;
    const start = pendingLinkSelection.start;
    const end = pendingLinkSelection.end;
    const currentLink = getLinkRange(rawValue, start, end);
    let nextValue = rawValue;
    let nextStart = start;
    let nextEnd = end;

    if (currentLink !== null) {
      const text = rawValue.slice(
        currentLink.contentStart,
        currentLink.contentEnd,
      );
      const replacement = `[${text}](${url})`;
      nextValue =
        rawValue.slice(0, currentLink.openStart) +
        replacement +
        rawValue.slice(currentLink.closeEnd);
      nextStart = currentLink.openStart;
      nextEnd = currentLink.openStart + replacement.length;
    } else {
      const text = rawValue.slice(start, end).trim() || url;
      const replacement = `[${text}](${url})`;
      nextValue = rawValue.slice(0, start) + replacement + rawValue.slice(end);
      nextStart = start;
      nextEnd = start + replacement.length;
    }

    updateValue(nextValue);
    isLinkModalOpen = false;
    linkValue = "";

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(nextStart, nextEnd);
      syncToolbarState(textarea);
    });
  };

  const insertImageMarkdown = (imageUrl: string, fileName: string) => {
    if (textarea === null) {
      return;
    }

    const rawValue = textarea.value;
    const cursor = pendingImageCursor;
    const { lineStart, lineEnd } = getLineBounds(rawValue, cursor);
    const currentLine = rawValue.slice(lineStart, lineEnd);
    const altText = toImageAltText(fileName);
    const imageMarkdown = `![${altText}](${imageUrl})`;

    let nextValue = rawValue;
    let nextCursor = cursor;
    if (currentLine.trim() === "") {
      nextValue =
        rawValue.slice(0, lineStart) + imageMarkdown + rawValue.slice(lineEnd);
      nextCursor = lineStart + imageMarkdown.length;
    } else {
      const insertion = `\n${imageMarkdown}`;
      nextValue =
        rawValue.slice(0, lineEnd) + insertion + rawValue.slice(lineEnd);
      nextCursor = lineEnd + insertion.length;
    }

    updateValue(nextValue);
    imageError = "";

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(nextCursor, nextCursor);
      syncToolbarState(textarea);
    });
  };

  const insertCarryOverMarker = () => {
    if (textarea === null) {
      return;
    }

    const rawValue = textarea.value;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const before = rawValue.slice(0, start);
    const after = rawValue.slice(end);

    const needsLeadingNewline = before !== "" && !before.endsWith("\n");
    const needsTrailingNewline = after !== "" && !after.startsWith("\n");
    const insertion = `${needsLeadingNewline ? "\n" : ""}${carryOverMarker}${needsTrailingNewline ? "\n" : ""}`;
    const nextValue = `${before}${insertion}${after}`;
    const nextCursor = before.length + insertion.length;

    updateValue(nextValue);

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(nextCursor, nextCursor);
      syncToolbarState(textarea);
    });
  };

  const removeLink = () => {
    if (textarea === null) {
      return;
    }

    const rawValue = textarea.value;
    const start = pendingLinkSelection.start;
    const end = pendingLinkSelection.end;
    const currentLink = getLinkRange(rawValue, start, end);
    if (currentLink === null) {
      closeLinkModal();
      return;
    }

    const text = rawValue.slice(
      currentLink.contentStart,
      currentLink.contentEnd,
    );
    const nextValue =
      rawValue.slice(0, currentLink.openStart) +
      text +
      rawValue.slice(currentLink.closeEnd);
    const nextStart = currentLink.openStart;
    const nextEnd = currentLink.openStart + text.length;

    updateValue(nextValue);
    isLinkModalOpen = false;
    linkValue = "";

    requestAnimationFrame(() => {
      if (textarea === null) {
        return;
      }

      textarea.focus();
      textarea.setSelectionRange(nextStart, nextEnd);
      syncToolbarState(textarea);
    });
  };

  const syncParagraphWithCursor = (target: HTMLTextAreaElement) => {
    paragraph = detectParagraph(target.value, target.selectionStart);
  };

  const syncInlineState = (target: HTMLTextAreaElement) => {
    const { value: currentValue, selectionStart, selectionEnd } = target;
    isBold =
      getInlineRange(
        currentValue,
        selectionStart,
        selectionEnd,
        inlineMarkers.bold,
      ) !== null;
    isItalic =
      getInlineRange(
        currentValue,
        selectionStart,
        selectionEnd,
        inlineMarkers.italic,
      ) !== null;
    isLink = getLinkRange(currentValue, selectionStart, selectionEnd) !== null;
    isStrike =
      getInlineRange(
        currentValue,
        selectionStart,
        selectionEnd,
        inlineMarkers.strike,
      ) !== null;
  };

  const syncToolbarState = (target: HTMLTextAreaElement) => {
    syncParagraphWithCursor(target);
    syncInlineState(target);
  };

</script>

<div class="md-input" class:md-input-error={Boolean(error)}>
  <Toolbar
    bind:value={paragraph}
    bold={isBold}
    italic={isItalic}
    link={isLink}
    strike={isStrike}
    onChange={applyParagraph}
    onBold={() => applyInline("bold")}
    onCarryOver={insertCarryOverMarker}
    onItalic={() => applyInline("italic")}
    onLink={openLinkModal}
    onStrike={() => applyInline("strike")}
  />

  <textarea
    id={id || undefined}
    class="textarea md-input-area"
    class:md-input-area-error={Boolean(error)}
    bind:this={textarea}
    bind:value
    aria-invalid={Boolean(error)}
    aria-describedby={error && id ? `${id}-error` : undefined}
    oninput={handleInput}
    onkeydown={handleKeydown}
    onclick={(event) => syncToolbarState(event.currentTarget as HTMLTextAreaElement)}
    onkeyup={(event) => syncToolbarState(event.currentTarget as HTMLTextAreaElement)}
    onselect={(event) => syncToolbarState(event.currentTarget as HTMLTextAreaElement)}
    placeholder={`# 見出し1

本文

- リスト
- リスト

> 引用

\`\`\`
コード
\`\`\``}
    spellcheck="false"
  ></textarea>

  {#if error}
    <p id={id ? `${id}-error` : undefined} class="admin-error-message">
      {error}
    </p>
  {/if}

  {#if isLinkModalOpen}
    <LinkModal
      bind:value={linkValue}
      bind:input={linkInput}
      onApply={applyLink}
      onRemove={removeLink}
      onClose={closeLinkModal}
    />
  {/if}

  {#if isImageModalOpen}
    <ImageModal
      error={imageError}
      uploading={isImageUploading}
      bind:input={imageInput}
      onClose={closeImageModal}
      onOpenPicker={openImagePicker}
      onSelectFile={onImageSelected}
    />
  {/if}
</div>

<style>
  .md-input {
    display: flex;
    flex-direction: column;
    min-height: 100%;
    overflow: hidden;
    border: 1px solid color-mix(in srgb, #64748b 40%, transparent);
    border-radius: 0.75rem;
    background: color-mix(in srgb, white 5%, transparent);
    transition:
      border-color 160ms ease,
      background 160ms ease,
      box-shadow 160ms ease;
  }

  .md-input:focus-within {
    border-color: color-mix(in srgb, #fbbf24 50%, transparent);
    background: color-mix(in srgb, white 7%, transparent);
  }

  .md-input.md-input-error {
    border-color: rgba(251, 113, 133, 0.9);
    background: #34172f;
    box-shadow:
      inset 0 0 0 1px rgba(244, 63, 94, 0.12),
      0 0 0 1px rgba(244, 63, 94, 0.08);
  }

  .md-input-area {
    flex: 1 1 auto;
    width: 100%;
    min-height: 22rem;
    resize: vertical;
    padding: 0.9rem 1rem;
    border: 0;
    border-radius: 0;
    background: color-mix(in srgb, white 5%, transparent);
    color: var(--color-slate-100, #f1f5f9);
    caret-color: currentColor;
    outline: none;
    font-family: "Iosevka Custom", "SFMono-Regular", "Consolas",
      "Liberation Mono", monospace;
    font-size: 0.96rem;
    line-height: 1.65;
    tab-size: 2;
    white-space: pre-wrap;
    overflow: auto;
  }

  .md-input-area.md-input-area-error {
    background: #34172f;
    color: #f1f5f9;
    caret-color: #f1f5f9;
  }

  .md-input-area::selection {
    background: rgba(148, 163, 184, 0.28);
  }

  .md-input-area::placeholder {
    color: rgba(226, 232, 240, 0.45);
  }

  @media screen and (max-width: 768px) {
    .md-input-area {
      min-height: 18rem;
    }

    .md-input-area {
      padding: 0.8rem 0.9rem;
      font-size: 0.92rem;
    }
  }
</style>

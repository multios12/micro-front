import type { InlineRange, LineBounds, LinkRange, ParagraphKind } from "./types";

const headingPattern = /^\s{0,3}(#{1,3})\s+/;
const orderedListPattern = /^\s*\d+\.\s+/;
const unorderedListPattern = /^\s*-\s+/;
const quotePattern = /^\s*>\s+/;
const codeFencePattern = /^\s*```/;

/**
 * Returns the start and end indexes of the line containing the given cursor.
 */
export const getLineBounds = (
  rawValue: string,
  cursor: number,
): LineBounds => {
  const lineStart = rawValue.lastIndexOf("\n", Math.max(0, cursor - 1)) + 1;
  const lineEndIndex = rawValue.indexOf("\n", cursor);
  const lineEnd = lineEndIndex === -1 ? rawValue.length : lineEndIndex;
  return { lineStart, lineEnd };
};

/**
 * Returns the full text of the line containing the given cursor.
 */
export const getLineAtCursor = (rawValue: string, cursor: number) => {
  const { lineStart, lineEnd } = getLineBounds(rawValue, cursor);
  return rawValue.slice(lineStart, lineEnd);
};

/**
 * Detects whether the cursor is currently inside a fenced code block.
 *
 * This counts code fences before the cursor. An odd number means the cursor is
 * between an opening and closing fence.
 */
export const isCursorInsideCodeBlock = (rawValue: string, cursor: number) => {
  const lines = rawValue.slice(0, cursor).split("\n");
  let fenceCount = 0;
  for (const line of lines) {
    if (codeFencePattern.test(line)) {
      fenceCount += 1;
    }
  }
  return fenceCount % 2 === 1;
};

/**
 * Detects the paragraph type at the current cursor position.
 */
export const detectParagraph = (
  rawValue: string,
  cursor: number,
): ParagraphKind => {
  const line = getLineAtCursor(rawValue, cursor);
  if (isCursorInsideCodeBlock(rawValue, cursor)) {
    return "code";
  }
  if (headingPattern.test(line)) {
    const level = line.match(headingPattern)?.[1].length ?? 0;
    return level === 1 ? "h1" : level === 2 ? "h2" : "h3";
  }
  if (orderedListPattern.test(line)) {
    return "ol";
  }
  if (unorderedListPattern.test(line)) {
    return "ul";
  }
  if (quotePattern.test(line)) {
    return "quote";
  }
  return "normal";
};

/**
 * Finds the inline Markdown range related to the current cursor or selection.
 *
 * A range is returned when the cursor is inside decorated text, or when the
 * selection is inside, wraps, or overlaps decorated text. This keeps toolbar
 * active states responsive for natural partial selections.
 */
export const getInlineRange = (
  rawValue: string,
  start: number,
  end: number,
  marker: string,
): InlineRange | null => {
  const { lineStart, lineEnd } = getLineBounds(rawValue, start);
  const lineEndIndex = rawValue.indexOf("\n", end);
  const expandedLineEnd = lineEndIndex === -1 ? rawValue.length : lineEndIndex;
  const lineText = rawValue.slice(lineStart, Math.max(lineEnd, expandedLineEnd));
  const selectionStart = start - lineStart;
  const selectionEnd = end - lineStart;

  let searchIndex = 0;
  while (searchIndex <= lineText.length) {
    const openIndex = lineText.indexOf(marker, searchIndex);
    if (openIndex === -1) {
      return null;
    }
    const contentStart = openIndex + marker.length;
    const closeIndex = lineText.indexOf(marker, contentStart);
    if (closeIndex === -1) {
      return null;
    }

    const isCursorInsideRange =
      start === end &&
      selectionStart >= contentStart &&
      selectionStart <= closeIndex;
    const isSelectionInsideContent =
      start !== end &&
      selectionStart >= contentStart &&
      selectionEnd <= closeIndex;
    const isSelectionWrappingRange =
      start !== end &&
      selectionStart <= openIndex &&
      selectionEnd >= closeIndex + marker.length;
    const isSelectionOverlappingContent =
      start !== end &&
      selectionStart < closeIndex &&
      selectionEnd > contentStart;

    if (
      isCursorInsideRange ||
      isSelectionInsideContent ||
      isSelectionWrappingRange ||
      isSelectionOverlappingContent
    ) {
      return {
        openStart: lineStart + openIndex,
        contentStart: lineStart + contentStart,
        contentEnd: lineStart + closeIndex,
        closeEnd: lineStart + closeIndex + marker.length,
      };
    }
    searchIndex = closeIndex + marker.length;
  }

  return null;
};

/**
 * Finds the Markdown link range related to the current cursor or selection.
 */
export const getLinkRange = (
  rawValue: string,
  start: number,
  end: number,
): LinkRange | null => {
  const checkStart = start === end ? start : start + 1;
  const lineStart = rawValue.lastIndexOf("\n", Math.max(0, start - 1)) + 1;
  const lineEndIndex = rawValue.indexOf("\n", end);
  const lineEnd = lineEndIndex === -1 ? rawValue.length : lineEndIndex;
  const lineText = rawValue.slice(lineStart, lineEnd);
  const linkPattern = /\[([^\]]+)\]\(([^)\s]+)\)/g;

  for (const match of lineText.matchAll(linkPattern)) {
    const matched = match[0];
    const text = match[1];
    const url = match[2];
    const localStart = match.index ?? 0;
    const openStart = lineStart + localStart;
    const contentStart = openStart + 1;
    const contentEnd = contentStart + text.length;
    const urlStart = contentEnd + 2;
    const urlEnd = urlStart + url.length;
    const closeEnd = openStart + matched.length;

    if (
      (checkStart >= contentStart && checkStart <= contentEnd) ||
      (start === end && start >= openStart && start <= closeEnd) ||
      (start !== end && start >= contentStart && end <= contentEnd)
    ) {
      return {
        openStart,
        contentStart,
        contentEnd,
        urlStart,
        urlEnd,
        closeEnd,
      };
    }
  }

  return null;
};

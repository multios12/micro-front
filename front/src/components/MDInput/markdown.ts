import type { InlineKind, ParagraphKind } from "./types";

/**
 * Inline decoration markers used by the editor toolbar.
 */
export const inlineMarkers = {
  bold: "**",
  italic: "*",
  strike: "~~",
} as const satisfies Record<InlineKind, string>;

const listMarkerPattern = /^(\s*)(?:\d+\.\s+|-\s+|>\s+|#{1,3}\s+)/;
const codeFencePattern = /^\s*```/;

/**
 * Removes the paragraph marker at the start of a single Markdown line.
 */
export const normalizeLine = (line: string) =>
  line.replace(listMarkerPattern, "");

/**
 * Wraps the current line in a fenced code block.
 *
 * If the line is already surrounded by code fences, the original line is
 * returned so repeated toolbar actions do not duplicate fences.
 */
export const applyCodeBlock = (
  rawValue: string,
  line: string,
  lineStart: number,
  lineEnd: number,
) => {
  const before = rawValue.slice(0, lineStart);
  const after = rawValue.slice(lineEnd);
  const previousLineStart = before.lastIndexOf("\n", Math.max(0, before.length - 2)) + 1;
  const previousLine = before.slice(previousLineStart).replace(/\n$/, "");
  const nextLineEnd = after.indexOf("\n");
  const nextLine =
    nextLineEnd === -1 ? after : after.slice(0, nextLineEnd);

  const hasWrappedFences =
    codeFencePattern.test(previousLine) && codeFencePattern.test(nextLine);

  if (hasWrappedFences) {
    return line;
  }

  return `\`\`\`\n${line}\n\`\`\``;
};

/**
 * Applies a paragraph-level Markdown format to one line.
 *
 * Existing heading/list/quote markers are removed first so changing from one
 * paragraph type to another does not stack multiple markers.
 */
export const applyLineFormat = (line: string, kind: ParagraphKind) => {
  if (line.trim() === "") {
    return line;
  }

  const baseLine = normalizeLine(line);
  switch (kind) {
    case "h1":
      return `# ${baseLine}`;
    case "h2":
      return `## ${baseLine}`;
    case "h3":
      return `### ${baseLine}`;
    case "ol":
      return `1. ${baseLine}`;
    case "ul":
      return `- ${baseLine}`;
    case "quote":
      return `> ${baseLine}`;
    case "code":
    case "normal":
    default:
      return baseLine;
  }
};

/**
 * Calculates where the caret should land after replacing a line.
 *
 * The editor keeps the caret near the same content position even when the
 * replacement adds or removes Markdown prefixes. Code blocks are multi-line,
 * so their cursor offset is handled separately.
 */
export const getNextCursorOffset = (
  previousLine: string,
  nextBlock: string,
  previousOffset: number,
) => {
  if (nextBlock.includes("\n")) {
    const lines = nextBlock.split("\n");
    const firstContentLine = lines[1] ?? "";
    return Math.min(nextBlock.length, 4 + Math.min(previousOffset, firstContentLine.length));
  }

  const previousIndentStripped = normalizeLine(previousLine);
  const removedPrefix = previousLine.length - previousIndentStripped.length;
  const nextIndentStripped = normalizeLine(nextBlock);
  const addedPrefix = nextBlock.length - nextIndentStripped.length;
  const baseOffset = Math.max(0, previousOffset - removedPrefix);
  return Math.min(nextBlock.length, addedPrefix + baseOffset);
};

/**
 * Creates the default image alt text from an uploaded file name.
 */
export const toImageAltText = (fileName: string) => {
  const baseName = fileName.replace(/\.[^.]+$/, "").trim();
  return baseName === "" ? "image" : baseName;
};

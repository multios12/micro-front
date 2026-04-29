export type ParagraphKind =
  | "normal"
  | "h1"
  | "h2"
  | "h3"
  | "ol"
  | "ul"
  | "code"
  | "quote";

export type InlineKind = "bold" | "italic" | "strike";

export type InlineRange = {
  openStart: number;
  contentStart: number;
  contentEnd: number;
  closeEnd: number;
};

export type LinkRange = InlineRange & {
  urlStart: number;
  urlEnd: number;
};

export type LineBounds = {
  lineStart: number;
  lineEnd: number;
};

/**
 * Turns a partial JSON token stream into chat-friendly copy for the UI.
 * The backend/agent still emits JSON; this is presentation-only.
 */

const SECTION_LABELS: Record<string, string> = {
  architecture: "architecture",
  execution_plan: "execution plan",
  risks: "risks",
  assumptions: "assumptions",
  open_questions: "open questions",
  metrics: "confidence metrics",
  idea: "idea refinements",
};

const STRING_FIELD_RE =
  /"(?:title|text|name|overview|description|responsibility)":\s*"((?:[^"\\]|\\.){3,120})/g;

export interface StreamPresentation {
  /** Primary line shown like ChatGPT “thinking” / status text. */
  headline: string;
  /** Short bullets derived from partial JSON — feels conversational. */
  bullets: string[];
  /** Whether the model has started emitting structured content. */
  hasContent: boolean;
}

/** Idle / waiting copy before the first token arrives. */
export function waitingPresentation(
  statusLine: string,
  role: string,
): StreamPresentation {
  const roleVerb: Record<string, string> = {
    build: "Building the technical foundation",
    review: "Reviewing the design",
    refine: "Refining the proposal",
    devils_advocate: "Stress-testing assumptions",
  };
  return {
    headline: statusLine || roleVerb[role] || "Working on your brainstorm",
    bullets: [],
    hasContent: false,
  };
}

/** Derive chat copy from an in-progress JSON buffer. */
export function presentStreamBuffer(
  buffer: string,
  statusLine: string,
  role: string,
): StreamPresentation {
  const trimmed = buffer.trim();
  if (!trimmed) {
    return waitingPresentation(statusLine, role);
  }

  const sections = detectSections(trimmed);
  const snippets = extractStringSnippets(trimmed);

  let headline = statusLine;
  if (!headline) {
    if (sections.length > 0) {
      headline = `Updating ${formatList(sections)}…`;
    } else {
      headline = "Composing structured output…";
    }
  }

  const bullets: string[] = [];
  for (const snippet of snippets.slice(-4)) {
    const line = clip(snippet.replace(/\\n/g, " ").replace(/\\"/g, '"'), 100);
    if (line && !bullets.includes(line)) {
      bullets.push(line);
    }
  }

  if (bullets.length === 0 && sections.length > 0) {
    bullets.push(`Working on ${sections[sections.length - 1]}…`);
  }

  return {
    headline,
    bullets,
    hasContent: true,
  };
}

function detectSections(buf: string): string[] {
  const found: string[] = [];
  for (const [key, label] of Object.entries(SECTION_LABELS)) {
    if (buf.includes(`"${key}"`)) {
      found.push(label);
    }
  }
  return found;
}

function extractStringSnippets(buf: string): string[] {
  const out: string[] = [];
  for (const match of buf.matchAll(STRING_FIELD_RE)) {
    const text = match[1];
    if (text) out.push(text);
  }
  return out;
}

function formatList(items: string[]): string {
  if (items.length === 1) return items[0];
  if (items.length === 2) return `${items[0]} and ${items[1]}`;
  return `${items.slice(0, -1).join(", ")}, and ${items[items.length - 1]}`;
}

function clip(s: string, max: number): string {
  const t = s.trim();
  if (t.length <= max) return t;
  return `${t.slice(0, max - 1)}…`;
}

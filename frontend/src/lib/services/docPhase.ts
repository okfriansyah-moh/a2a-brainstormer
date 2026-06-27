/** SSE doc.phase payload from finalize document generation. */
export type DocPhasePayload = {
  doc_key?: string;
  step?: string;
  detail?: string;
  section?: string;
  findings_count?: number;
};

const STEP_VERBS: Record<string, string> = {
  section_enhance: "enhancing",
  section_repair: "repairing",
  coherence_audit: "checking consistency",
  coherence_fix: "fixing consistency",
  enricher: "preparing",
  draft: "generating draft",
  repair: "repairing document",
  complete: "complete",
};

/** Format a human-readable running line for the finalize progress panel. */
export function formatDocPhaseRunningLine(
  payload: DocPhasePayload,
  docLabel: string,
): string | null {
  if (!payload.doc_key) {
    return null;
  }
  const step = payload.step ?? "";
  const verb = STEP_VERBS[step];

  if (step === "coherence_audit") {
    if (payload.findings_count && payload.findings_count > 0) {
      return `${docLabel} — ${payload.findings_count} consistency issue(s) found`;
    }
    return `${docLabel} — checking consistency…`;
  }

  if (payload.section && verb) {
    return `${docLabel} · §${payload.section} — ${verb}…`;
  }

  if (payload.detail) {
    if (verb) {
      return `${docLabel}: ${payload.detail}`;
    }
    return `${docLabel}: ${payload.detail}`;
  }

  if (verb) {
    return `${docLabel} — ${verb}…`;
  }

  return null;
}

/**
 * When a section changes, return a log line for the completed prior section.
 * Returns null when no transition should be logged.
 */
export function sectionTransitionLogLine(
  prev: { docKey: string; section: string } | null,
  next: DocPhasePayload,
  docLabel: string,
): string | null {
  if (!prev || !next.section || !next.doc_key) {
    return null;
  }
  if (prev.docKey !== next.doc_key || prev.section === next.section) {
    return null;
  }
  return `${docLabel} · §${prev.section} — done`;
}

/** Whether this phase event should append a permanent log line (not just runningLine). */
export function shouldAppendPhaseLog(payload: DocPhasePayload): boolean {
  if (payload.step === "complete") {
    return true;
  }
  if (
    payload.step === "coherence_audit" &&
    payload.findings_count &&
    payload.findings_count > 0
  ) {
    return true;
  }
  return false;
}

/** Log line for a completed doc.phase complete step. */
export function formatDocPhaseLogLine(
  payload: DocPhasePayload,
  docLabel: string,
): string | null {
  if (payload.step === "complete" && payload.detail) {
    return `✦ ${docLabel} — ${payload.detail}`;
  }
  if (
    payload.step === "coherence_audit" &&
    payload.findings_count &&
    payload.findings_count > 0
  ) {
    return `${docLabel} — ${payload.findings_count} consistency issue(s) found`;
  }
  return null;
}

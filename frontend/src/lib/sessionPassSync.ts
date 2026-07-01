import type { CanonicalState, Session } from "$lib/types";

/** True when the server finished the expected pipeline pass (status left "running"). */
export function isServerPassComplete(
  session: Session,
  expectedPass: number,
): boolean {
  if (session.status === "running") return false;
  const completed = session.current_state?.meta?.iteration ?? 0;
  if (expectedPass > 0 && completed < expectedPass) return false;
  return (
    session.status === "active" ||
    session.status === "converged" ||
    session.status === "approved"
  );
}

export function passCountsFromState(state: CanonicalState | null | undefined): {
  execution_plan_steps: number;
  risks_count: number;
  open_questions_count: number;
} {
  return {
    execution_plan_steps: state?.execution_plan?.length ?? 0,
    risks_count: state?.risks?.length ?? 0,
    open_questions_count: state?.open_questions?.length ?? 0,
  };
}

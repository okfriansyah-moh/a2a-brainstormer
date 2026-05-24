/**
 * Session store — holds all state for the active brainstorm session.
 * Consumed by the session workspace route and all workspace components.
 *
 * Shape (see docs/PLAN.md §8.9):
 *   session_id     — UUID of the active session, null before session creation
 *   idea           — the original idea text entered by the user
 *   state          — current CanonicalState (null before first iteration)
 *   iteration      — current iteration number (0 = not yet iterated)
 *   agents         — ordered list of session agents (includes role + skills[])
 *   loading        — true while an API call is in flight
 *   agentStatuses  — per-agent live status driven by SSE events
 */
import { writable } from "svelte/store";
import type { CanonicalState, SessionAgent } from "$lib/types";
import type { SSEEvent } from "$lib/services/sse";

export type AgentStatus = "waiting" | "running" | "done" | "error";

export interface SessionStoreState {
  session_id: string | null;
  idea: string;
  state: CanonicalState | null;
  iteration: number;
  agents: SessionAgent[];
  loading: boolean;
  /** Live per-agent status map, keyed by agent_id. Updated via SSE events. */
  agentStatuses: Record<string, AgentStatus>;
}

const initialState: SessionStoreState = {
  session_id: null,
  idea: "",
  state: null,
  iteration: 0,
  agents: [],
  loading: false,
  agentStatuses: {},
};

function createSessionStore() {
  const { subscribe, set, update } = writable<SessionStoreState>(initialState);

  return {
    subscribe,

    /** Replace the full session state (called after createSession). */
    setSession(sessionId: string, idea: string) {
      update((s) => ({
        ...s,
        session_id: sessionId,
        idea,
        state: null,
        iteration: 0,
      }));
    },

    /** Set the ordered list of session agents. */
    setAgents(agents: SessionAgent[]) {
      update((s) => ({ ...s, agents }));
    },

    /** Replace the canonical state and advance the iteration counter. */
    updateState(newState: CanonicalState) {
      update((s) => ({
        ...s,
        state: newState,
        iteration: newState.meta?.iteration ?? s.iteration,
      }));
    },

    /** Toggle the loading flag. */
    setLoading(loading: boolean) {
      update((s) => ({ ...s, loading }));
    },

    /** Reset to initial state (e.g. when navigating away from a session). */
    reset() {
      set(initialState);
    },

    /**
     * applyEvent updates agentStatuses based on incoming SSE lifecycle events.
     * Called by the session workspace page for every event from the SSE stream.
     *
     * Events handled:
     *   agent.started  → set agent status to 'running'
     *   agent.complete → set agent status to 'done'
     *   agent.error    → set agent status to 'error'
     *   iteration.start → reset all agent statuses to 'waiting'
     */
    applyEvent(evt: SSEEvent) {
      const payload = evt.data as Record<string, unknown> | null;
      const agentID =
        payload && typeof payload === "object"
          ? (payload["agent_id"] as string | undefined)
          : undefined;

      update((s) => {
        switch (evt.type) {
          case "iteration.start": {
            // Reset all agent statuses to waiting at the start of a new iteration.
            const reset: Record<string, AgentStatus> = {};
            for (const agent of s.agents) {
              reset[agent.id] = 'waiting';
            }
            return { ...s, agentStatuses: reset };
          }
          case "agent.started": {
            if (!agentID) return s;
            return {
              ...s,
              agentStatuses: { ...s.agentStatuses, [agentID]: "running" },
            };
          }
          case "agent.complete": {
            if (!agentID) return s;
            return {
              ...s,
              agentStatuses: { ...s.agentStatuses, [agentID]: "done" },
            };
          }
          case "agent.error": {
            if (!agentID) return s;
            return {
              ...s,
              agentStatuses: { ...s.agentStatuses, [agentID]: "error" },
            };
          }
          default:
            return s;
        }
      });
    },
  };
}

export const sessionStore = createSessionStore();

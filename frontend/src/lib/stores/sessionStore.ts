/**
 * Session store — holds all state for the active brainstorm session.
 * Consumed by the session workspace route and all workspace components.
 *
 * Shape (see docs/PLAN.md §8.9):
 *   session_id  — UUID of the active session, null before session creation
 *   idea        — the original idea text entered by the user
 *   state       — current CanonicalState (null before first iteration)
 *   iteration   — current iteration number (0 = not yet iterated)
 *   agents      — ordered list of session agents (includes role + skills[])
 *   loading     — true while an API call is in flight
 */
import { writable } from "svelte/store";
import type { CanonicalState, SessionAgent } from "$lib/types";

export interface SessionStoreState {
  session_id: string | null;
  idea: string;
  state: CanonicalState | null;
  iteration: number;
  agents: SessionAgent[];
  loading: boolean;
}

const initialState: SessionStoreState = {
  session_id: null,
  idea: "",
  state: null,
  iteration: 0,
  agents: [],
  loading: false,
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
  };
}

export const sessionStore = createSessionStore();

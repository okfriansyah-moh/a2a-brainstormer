/**
 * Agent registry store — holds the full agent catalogue and skill library.
 * Consumed by agent management pages, the session creation form, and
 * any component that needs to list or select agents and skills.
 *
 * Shape (see docs/PLAN.md §8.9):
 *   agents   — full list of registered agents, each including their skills[]
 *   skills   — full skill library
 *   loading  — true while an API call is in flight
 */
import { writable } from "svelte/store";
import type { Agent, Skill } from "$lib/types";

export interface AgentRegistryStoreState {
  agents: Agent[];
  skills: Skill[];
  loading: boolean;
}

const initialState: AgentRegistryStoreState = {
  agents: [],
  skills: [],
  loading: false,
};

function createAgentRegistryStore() {
  const { subscribe, set, update } =
    writable<AgentRegistryStoreState>(initialState);

  return {
    subscribe,

    /** Replace the full agent list (called after getAgents). */
    setAgents(agents: Agent[]) {
      update((s) => ({ ...s, agents }));
    },

    /** Replace the full skill list (called after getSkills). */
    setSkills(skills: Skill[]) {
      update((s) => ({ ...s, skills }));
    },

    /** Append a newly created agent to the list. */
    addAgent(agent: Agent) {
      update((s) => ({ ...s, agents: [...s.agents, agent] }));
    },

    /** Remove an agent from the list by ID. */
    removeAgent(agentId: string) {
      update((s) => ({
        ...s,
        agents: s.agents.filter((a) => a.id !== agentId),
      }));
    },

    /** Update an existing agent in the list by ID. */
    updateAgent(updated: Agent) {
      update((s) => ({
        ...s,
        agents: s.agents.map((a) => (a.id === updated.id ? updated : a)),
      }));
    },

    /** Append a newly created skill to the library. */
    addSkill(skill: Skill) {
      update((s) => ({ ...s, skills: [...s.skills, skill] }));
    },

    /** Remove a skill from the library by ID. */
    removeSkill(skillId: string) {
      update((s) => ({
        ...s,
        skills: s.skills.filter((sk) => sk.id !== skillId),
      }));
    },

    /** Update an existing skill in the library by ID. */
    updateSkill(updated: Skill) {
      update((s) => ({
        ...s,
        skills: s.skills.map((sk) => (sk.id === updated.id ? updated : sk)),
      }));
    },

    /** Toggle the loading flag. */
    setLoading(loading: boolean) {
      update((s) => ({ ...s, loading }));
    },

    /** Reset to initial state. */
    reset() {
      set(initialState);
    },
  };
}

export const agentRegistryStore = createAgentRegistryStore();

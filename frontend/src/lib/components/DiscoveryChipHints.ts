/** Static discovery chip defaults (v2 mockup) + async merge helper. */
export interface DiscoveryHints {
  q2: string[];
  q3: string[];
  q4: string[];
}

export const STATIC_DISCOVERY_HINTS: DiscoveryHints = {
  q2: [
    "Core data model",
    "API contracts",
    "Authentication / auth",
    "UI prototype",
    "Integration with existing systems",
    "Performance baseline",
    "Security review",
    "Documentation",
  ],
  q3: [
    "Zero data loss",
    "Sub-100ms latency",
    "Horizontal scalability",
    "Full audit trail",
    "Multi-tenant isolation",
    "Offline support",
    "GDPR / compliance",
    "Self-hostable",
  ],
  q4: [
    "Saves hours per week",
    "Less operational overhead",
    "Cheaper to run",
    "More reliable / fewer incidents",
    "Better developer experience",
    "Enables workflows not possible before",
  ],
};

/** Merge async LLM hints over static defaults (never replaces non-empty tiers). */
export function mergeHints(dynamic: Partial<DiscoveryHints>): DiscoveryHints {
  return {
    q2: dynamic.q2?.length ? dynamic.q2 : [...STATIC_DISCOVERY_HINTS.q2],
    q3: dynamic.q3?.length ? dynamic.q3 : [...STATIC_DISCOVERY_HINTS.q3],
    q4: dynamic.q4?.length ? dynamic.q4 : [...STATIC_DISCOVERY_HINTS.q4],
  };
}

/** Rotating status lines during finalize document generation (presentation only). */
export const DOC_LOADING_PHRASES: Record<string, string[]> = {
  architecture: [
    "Reading canonical architecture state…",
    "Mapping layers and module boundaries…",
    "Drafting component responsibilities…",
    "Describing data flows between modules…",
    "Adding deployment and observability notes…",
    "Checking consistency across sections…",
    "Polishing technical depth per component…",
    "Aligning diagrams with execution plan…",
    "Stress-testing failure modes…",
    "Composing architecture markdown…",
  ],
  plan: [
    "Reading execution plan from canonical state…",
    "Structuring implementation tasks…",
    "Writing goals and file ownership…",
    "Adding validation commands per task…",
    "Linking tasks to architecture modules…",
    "Filling invariant checks and dependencies…",
    "Expanding thin task stubs…",
    "Checking task format against spec…",
    "Repairing rubric findings…",
    "Composing PLAN markdown…",
  ],
  readme: [
    "Summarising the product idea…",
    "Drafting quick-start commands…",
    "Describing repo layout…",
    "Adding setup prerequisites…",
    "Linking architecture and roadmap…",
    "Writing contributor guidance…",
    "Polishing onboarding copy…",
    "Checking README completeness…",
    "Composing README markdown…",
  ],
};

export const DEFAULT_DOC_LOADING_PHRASES: string[] = [
  "Loading canonical session state…",
  "Preparing document scaffold…",
  "Running AI enhancement pass…",
  "Enhancing section by section…",
  "Checking document quality…",
  "Applying consistency fixes…",
  "Streaming model output…",
  "Almost done with this document…",
];

export function docLoadingPhrases(docKey: string): string[] {
  return DOC_LOADING_PHRASES[docKey] ?? DEFAULT_DOC_LOADING_PHRASES;
}

/** True when the line is the generic placeholder set before SSE arrives. */
export function isGenericDocRunningLine(line: string | null): boolean {
  if (!line) return true;
  return /^Generating .+…$/.test(line);
}

# PRD and RFC Confluence Publishing Design

Date: 2026-06-02
Status: Proposed

## Objective

Extend the existing finalize workflow so the same finalize output page can generate two new document types, `prd` and `rfc`, in addition to the current `architecture`, `roadmap`, `plan`, and `readme` outputs.

`prd` and `rfc` are not local-only artifacts. They must be published to Confluence as part of the same generation workflow.

Required sequencing:

- If only `rfc` is selected, generate RFC directly from the converged final session state and publish it to Confluence.
- If `prd` and `rfc` are both selected, generate PRD first, publish PRD, then generate RFC using the converged final session state plus the generated PRD content, then publish RFC.
- Confluence publishing always creates new pages. Existing pages are never updated in this workflow.
- Confluence destination is chosen on the finalize page at generation time.

## Current State

The current implementation already supports selectable output docs on the finalize page and backend generation via the session and markdown modules.

Relevant extension points:

- `frontend/src/routes/session/[id]/finalize/+page.svelte`
- `frontend/src/lib/services/api.ts`
- `frontend/src/lib/types.ts`
- `backend/internal/modules/session/model.go`
- `backend/internal/modules/session/handler.go`
- `backend/internal/modules/session/service.go`
- `backend/internal/modules/markdown/generator.go`
- `backend/internal/modules/markdown/orchestrator.go`

Today the backend treats document generation as a flat list of independent keys. That works for `architecture`, `roadmap`, `plan`, and `readme`, but it does not model the dependency rule that RFC may optionally depend on freshly generated PRD content.

## Goals

- Add `prd` and `rfc` as first-class output document types.
- Keep the existing finalize page as the single user-facing entry point for generation.
- Publish PRD and RFC to Confluence in the same workflow that generates them.
- Guarantee server-side generation ordering for `prd -> rfc` when both are selected.
- Return generated content and Confluence publish metadata to the frontend.
- Preserve current behavior for `architecture`, `roadmap`, `plan`, and `readme`.

## Non-Goals

- Replacing the current finalize flow with a separate publishing wizard.
- Updating existing Confluence pages in place.
- Introducing Bitbucket as the canonical publication target for PRD or RFC.
- Requiring PRD as a prerequisite for generating RFC when RFC is selected alone.

## Alternatives Considered

### Option A: Extend existing finalize generation with ordered orchestration

This keeps the current finalize page and endpoint model, but introduces a backend orchestration layer that can order generation, pass upstream generated content into downstream generators, and publish selected docs to Confluence.

Pros:

- Matches the existing UX.
- Smallest conceptual change for users.
- Places sequencing and publishing rules in the backend where they are enforceable.
- Preserves current per-document generator model.

Cons:

- Requires expanding the generation contract beyond independent flat keys.

### Option B: Add a second publish endpoint after finalize

This would split finalization from PRD/RFC publishing.

Pros:

- Cleaner technical separation between finalize and publish.

Cons:

- Violates the requirement that the same finalize output page handles PRD and RFC generation.
- Adds user-visible workflow complexity.

### Option C: Hide PRD dependency inside the RFC generator only

This would keep orchestration minimal by letting the RFC generator internally derive or synthesize any PRD context it needs.

Pros:

- Smallest initial change to the orchestration surface.

Cons:

- Hides business workflow in a single generator.
- Makes PRD-first sequencing hard to reason about and hard to test.
- Makes Confluence publication state harder to report accurately.

## Recommended Approach

Adopt Option A.

Keep the existing finalize page and backend endpoints, but add an ordered document-generation and Confluence-publishing workflow behind them.

Key rule:

- The frontend selects output docs and Confluence destination.
- The backend computes the ordered generation plan.
- The backend generates and publishes documents sequentially.
- The response returns generated documents plus publication metadata.

## User Flow

1. User runs ideation until the session reaches a converged, finalizable state.
2. User opens the existing finalize page.
3. The page shows selectable outputs:
   - `architecture`
   - `roadmap`
   - `plan`
   - `readme`
   - `prd`
   - `rfc`
4. The page also asks the user for Confluence destination information required for publication.
5. User clicks generate.
6. Backend finalizes the session if needed.
7. Backend computes an ordered generation plan.
8. Backend generates each selected document sequentially.
9. For `prd` and `rfc`, backend creates new Confluence pages immediately after generation.
10. Frontend displays generated content and publication results.

## Generation Rules

### Base ordering

Current document ordering remains deterministic. The UI and backend share a canonical order.

Suggested order:

- `architecture`
- `roadmap`
- `plan`
- `readme`
- `prd`
- `rfc`

### Sequencing constraints

- If `rfc` is selected and `prd` is not selected, generate RFC from the session's converged final canonical state.
- If both `prd` and `rfc` are selected, the backend must reorder or validate the request so PRD executes before RFC regardless of frontend checkbox order.
- RFC generation may consume the generated PRD content only when PRD is part of the same run.

### Determinism

Business sequencing must remain deterministic:

- Same session state + same selected docs + same Confluence inputs should produce the same ordered generation plan.
- Generation ordering must not depend on UI click timing.
- The publish target must come from explicit request input, not hidden ambient state.

## Frontend Design

### Finalize page

Extend `frontend/src/routes/session/[id]/finalize/+page.svelte` to:

- Add `prd` and `rfc` to `ALL_DOCS`.
- Keep the current sequential progress presentation.
- Add Confluence destination inputs on the same page.
- Include publish results for PRD and RFC in the result cards.

Required behavior:

- If both `prd` and `rfc` are selected, display them in the canonical generation order.
- If generation succeeds, show page title and URL for created Confluence pages.
- If Confluence creation fails for PRD or RFC, show a per-document failure state.

### Frontend request model

Extend finalize and single-document generation requests so the page can provide publish destination data.

Suggested request additions:

- Confluence cloud/site identifier
- Confluence space identifier
- Optional parent page identifier
- Optional title prefix or placement metadata if needed by the business rule

The exact field names should be backend-owned and mirrored in `frontend/src/lib/types.ts`.

## Backend Design

### Session module

Extend `backend/internal/modules/session/model.go`:

- Add `prd` and `rfc` to `AllowedOutputDocs`.
- Decide whether `DefaultOutputDocs` stays unchanged or expands. The recommendation is to leave defaults unchanged and make PRD/RFC opt-in on the finalize page.
- Extend finalize and single-document request/response types with Confluence publish input and publish result data.

`FinalizeResponse` and `GenerateDocumentResponse` should return:

- generated document content
- per-document publish metadata when the key is `prd` or `rfc`

Suggested publish result shape:

- `target`: `confluence`
- `status`: `created` or `failed`
- `page_id`
- `page_url`
- `title`
- `error_message` when publication fails

### Handler layer

Extend `backend/internal/modules/session/handler.go` so:

- `POST /sessions/{id}/finalize` accepts Confluence destination input in the request body.
- `POST /sessions/{id}/generate-document` accepts the same publish input when generating `prd` or `rfc`.
- Validation rejects PRD/RFC publication requests when required Confluence fields are missing.

The finalize handler should continue to finalize the session first, then delegate document generation and publication to a backend orchestrator.

### Service layer

`backend/internal/modules/session/service.go` should continue to own session state transitions only.

Do not move Confluence publishing into the session service. Session service should remain responsible for:

- finalizable-state validation
- output docs validation
- status transition to approved

Generation and publishing orchestration should remain outside the session service to preserve module boundaries.

## Markdown Module Design

### New generators

Add two new generators in `backend/internal/modules/markdown/`:

- PRD generator
- RFC generator

The content should follow the example patterns from the provided PRD and tech doc references, but the implementation must derive content from the converged session state and any explicitly provided upstream generated content.

### Generator contracts

The current `Generators map[string]func(state.CanonicalState) (string, error)` is too narrow for RFC depending on generated PRD content.

Introduce an internal orchestration contract that can provide generation context, for example:

- canonical state
- selected document key
- previously generated documents in the same run
- publish destination inputs if needed by title-building logic

The public handler-facing generation surface should still return a map of generated documents, but internally the markdown module needs a richer execution context.

### Ordered orchestration

Add orchestration logic that:

- resolves the canonical ordered key list
- generates each selected document sequentially
- stores generated content in an in-memory per-request map
- passes generated PRD content into RFC generation when available

This orchestration belongs in the markdown package or a closely related backend orchestration unit, not in the frontend.

## Confluence Publishing Design

### Required behavior

PRD and RFC generation must publish to Confluence in the same request flow.

Rules:

- Always create new pages.
- Never update an existing page.
- Destination is supplied from the finalize page.
- Publishing failure is visible in the response.

### Publisher boundary

Introduce a Confluence-specific publishing component under platform infrastructure, for example under `backend/internal/platform/`.

Responsibilities:

- create Confluence page
- return page id, title, and URL
- normalize publication errors

The markdown module should not perform raw Confluence API calls directly. It should generate content. A dedicated Confluence publisher should own remote publication.

### Title strategy

Titles should be deterministic and session-derived.

Suggested pattern:

- `PRD - <short session title> - <date or session id suffix>`
- `RFC - <short session title> - <date or session id suffix>`

Because pages are always newly created, the title strategy should avoid accidental collisions while remaining readable.

## Persistence Design

Because each PRD/RFC publish creates a new page, publication history must be stored append-only.

Add a new persistence record for generated publication history, for example a new table that stores:

- generated document run id
- session id
- doc key
- confluence page id
- confluence page url
- page title
- publish status
- created at

This history supports:

- auditability
- re-runs without overwriting old pages
- future UI listing of previously published artifacts

This table should be additive through a new migration file. Existing migrations must not be modified.

## API Contract Changes

### Finalize request

Extend finalize input with optional publication config that becomes required when selected docs contain `prd` or `rfc`.

Suggested semantics:

- no PRD/RFC selected: publication fields may be absent
- PRD or RFC selected: publication fields required

### Finalize response

Extend generated document payload to include optional publish result.

### Single-document generation

`POST /sessions/{id}/generate-document` remains supported.

Rules:

- `key = prd`: generate PRD and publish PRD
- `key = rfc`: generate RFC from converged session state and publish RFC
- for non-published docs, existing behavior remains unchanged

## Error Handling

### Validation errors

Return HTTP 400 when:

- output doc key is invalid
- `prd` or `rfc` is selected without required Confluence destination fields
- session id is invalid

Return HTTP 422 when:

- session has no final canonical state available for generation

### Generation failures

Return HTTP 500 when:

- document generation fails before any response can be assembled

### Publication failures

Preferred behavior:

- document generation succeeds
- Confluence page creation fails
- response still returns generated document content with publish result marked failed

This preserves user access to the generated PRD or RFC body even when publication fails.

If the current finalize endpoint cannot safely represent partial success, use a response shape that reports generated docs plus per-doc publish failures instead of failing the whole request.

## Testability

### Backend tests

Add tests covering:

- allowed doc validation for `prd` and `rfc`
- finalize request validation when Confluence fields are missing
- ordered generation plan for `prd + rfc`
- RFC-only generation path
- PRD then RFC generation using generated PRD as input
- Confluence publish success result mapping
- Confluence publish failure result mapping
- append-only persistence of publication history

### Frontend tests

Add tests covering:

- finalize page displays `prd` and `rfc`
- finalize page sends publication inputs with generation request
- sequential progress UI for PRD then RFC
- publish result rendering for created Confluence pages
- publish failure rendering without losing generated content visibility

## Security and Operational Constraints

- Confluence credentials must not be accepted from end users in the request body.
- Any secrets required for Confluence integration must be resolved server-side from approved configuration.
- Session validation rules remain unchanged: only approved or converged-ready sessions can generate output docs.
- Publication target inputs from the finalize page must be validated and bounded.

## Implementation Notes

- Keep existing document generation behavior unchanged for current document keys.
- Add PRD and RFC as opt-in output docs rather than changing defaults.
- Put Confluence integration behind a dedicated platform component, not inside session service and not inside raw generators.
- Use a new migration for publication history if auditability is required in the first implementation slice.

## Open Decisions Resolved

- Finalize page is the user entry point for PRD and RFC generation.
- RFC-only generation is allowed and uses the converged final session state.
- PRD plus RFC generation must run PRD first, then RFC.
- PRD and RFC publishing to Confluence is mandatory.
- Publishing always creates new Confluence pages.
- Confluence destination is selected on the finalize page.

## Spec Self-Review

- No placeholders or TBD markers remain.
- PRD/RFC sequencing is explicit and consistent.
- Module responsibilities stay aligned with current boundaries: session for lifecycle, markdown for generation, platform for Confluence publishing.
- Existing output docs are preserved and not behaviorally changed.

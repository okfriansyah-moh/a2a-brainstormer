Below is a **refined, execution-grade blueprint** aligned with:

- Go **1.26**
- A2A via a2a-go
- LLM abstraction (**Copilot-first**, Claude-ready)
- **Role-fluid agents (A ↔ B switching)**
- **Modular monolith \+ vertical slice architecture**
- ## Frontend as a **structured multi-agent workspace (not chat)**

  # **1\. System Objective**

Idea → controlled multi-agent iteration → convergence → **architecture.md \+ roadmap.md**

Non-goals:

- not a chatbot
- ## not an autonomous swarm

  # **2\. High-Level Architecture**

```
frontend/ (Next.js)
       ↓
backend/ (Go 1.26 modular monolith)
       ↓
┌─────────────── A2A ───────────────┐
↓                                   ↓
Agent A (Go)                     Agent B (Go)
(a2a-go server)                 (a2a-go server)
       ↓                                ↓
       LLM Provider (Copilot / Claude)

Postgres ← canonical state
Markdown generator → .md files
```

---

# **3\. Backend Architecture (Modular Monolith \+ Vertical Slice)**

## **Root Structure**

```
backend/
 cmd/
   server/
 internal/
   platform/
   shared/
 modules/
   session/
   iteration/
   agent/
   state/
   convergence/
   markdown/
```

---

# **4\. Architectural Philosophy (Important)**

### **Modular Monolith**

- single deployable unit
- clear module boundaries
- ## no distributed complexity

  ### **Vertical Slice**

Each module owns:

- handler
- service
- repository
- domain model

NO horizontal layering like:

- controllers/
- services/
- repositories/

Instead:

feature-first

---

# **5\. Internal Platform Layer**

```
internal/platform/
 http/
 a2a/
 llm/
 db/
 config/
 logger/
```

---

## **5.1 A2A Client Wrapper**

Wrap a2a-go

Responsibilities:

- send task
- handle response
- ## retry \+ validation

  ## **5.2 LLM Abstraction**

```
type LLMProvider interface {
   Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
```

Implementations:

- CopilotProvider (default)
- ## ClaudeProvider (future)

  ## **5.3 DB Layer**

- PostgreSQL
- ## minimal ORM (sqlc / pgx preferred)

  # **6\. Modules (Vertical Slices)**

  ***

  ## **6.1 session/**

Handles lifecycle.

```
modules/session/
 handler.go
 service.go
 repository.go
 model.go
```

### **Responsibilities**

- create session
- store initial idea
- ## manage status

  ## **6.2 iteration/**

Core loop entry.

```
modules/iteration/
 handler.go
 service.go
 engine.go
```

### **Responsibilities**

- run iteration
- call agents
- ## update state

  ## **6.3 agent/**

A2A interaction layer.

```
modules/agent/
 client.go
 role.go
```

### **Responsibilities**

- call Agent A / B
- assign role (build/review)
- ## enforce schema

  ## **6.4 state/**

Canonical state management.

```
modules/state/
 model.go
 merge.go
 validator.go
```

### **Responsibilities**

- state schema
- merge logic
- ## validation

  ## **6.5 convergence/**

```
modules/convergence/
 engine.go
```

### **Responsibilities**

- detect convergence
- ## stop condition

  ## **6.6 markdown/**

```
modules/markdown/
 generator.go
```

### **Responsibilities**

- render:
  - architecture.md
  - roadmap.md

  ***

  # **7\. Agents (Separate Services)**

Both identical:

```
agent/
 cmd/server/
 internal/
   handler/
   llm/
```

---

## **Agent Responsibilities**

- accept A2A `/task`
- execute role:
  - build
  - review
- ## return structured JSON

  ## **Role is dynamic**

Request includes:

```
{
 "role": "build" | "review",
 "state": {}
}
```

---

# **8\. Canonical State Model**

```
{
 "idea": {},
 "architecture": {},
 "execution_plan": [],
 "risks": [],
 "assumptions": [],
 "open_questions": [],
 "metrics": {
   "confidence": 0.0
 },
 "meta": {
   "iteration": 0,
   "roles": {
     "agentA": "",
     "agentB": ""
   }
 }
}
```

---

# **9\. Iteration Engine (Core Logic)**

```
for i := 1; i <= maxIter; i++ {

   if i%2 == 1 {
       roleA = "build"
       roleB = "review"
   } else {
       roleA = "review"
       roleB = "build"
   }

   outA := agent.Call(A, roleA, state)
   outB := agent.Call(B, roleB, outA)

   newState := state.Merge(outA, outB)

   if convergence.Check(state, newState) {
       break
   }

   state = newState
}
```

---

# **10\. Merge Strategy**

Rules:

- union risks (deduplicate)
- remove resolved items
- collapse duplicate steps
- ## reject vague outputs

  ## **Stability Rule**

- if both agents agree → lock
- ## if conflict persists → user decision

  # **11\. Convergence Engine**

Stop when:

- no new critical risks
- execution plan complete
- Δconfidence \< threshold
- OR user approves
- ## OR max iterations reached

  # **12\. API Design**

```
POST   /sessions
POST   /sessions/{id}/iterate
GET    /sessions/{id}
POST   /sessions/{id}/finalize
```

---

# **13\. Frontend Architecture**

---

## **Stack**

- Next.js
- Tailwind
- ## React Query

  ## **Layout**

```
-------------------------------------------------
| Agent A Panel       | Agent B Panel           |
| Role: Build/Review  | Role: Build/Review      |
-------------------------------------------------
| Control Panel (User Input + Actions)          |
-------------------------------------------------
| State View (Architecture / Plan / Risks)      |
-------------------------------------------------
| Iteration Timeline                           |
-------------------------------------------------
```

---

## **Key UX Principles**

- NOT chat-based
- show:
  - role per iteration
  - diff changes
  - canonical state

  ***

  # **14\. Frontend Modules**

```
frontend/
 app/
 components/
   agent-panel/
   control-panel/
   state-view/
   timeline/
 hooks/
 services/api/
```

---

# **15\. Output Artifacts**

Generated:

- architecture.md
- ## roadmap.md

  # **16\. Deployment**

  ## **MVP**

- docker-compose
- services:
  - backend
  - agent A
  - agent B
  - postgres

  ***

  # **17\. Failure Modes**

  ***

  ## **1\. Oscillation**

Fix:

- stability bias
- ## user override

  ## **2\. Weak critique**

Fix:

- ## enforce reviewer outputs issues

  ## **3\. Schema drift**

Fix:

- ## strict validation

  ## **4\. Copilot inconsistency**

Fix:

- low temperature
- ## structured prompts

  # **18\. What You Are Building**

**Deterministic multi-agent design system (IDE-like)**

Not:

- AI chat
- ## swarm agents

  # **19\. Recommended Next Step**

Critical path:

### **1\. Prompt contracts (Builder / Reviewer / Refiner)**

→ determines output quality

### **2\. Go skeleton (modules \+ wiring)**

### **3\. UI wireframe (to prevent chat-like UX)**

---

If you skip prompts:

system works technically but produces low-quality designs.

regenerate the details blueprint and architecture and refine frontend techstack to use svelte instead, svelte latest stable version

Below is the **final refined blueprint** with:

- Go **1.26**
- A2A via a2a-go
- LLM abstraction (**Copilot-first**, Claude-ready, **per-agent provider + model + credentials**)
- **Dynamic N-agent registry** (session-selectable, minimum 2 agents per session)
- **Role catalog** (build / review / refine / devils_advocate)
- **Tiered LLM config** (global default → agent-level → session override; credentials via env var refs only)
- **Per-agent skill system** (user-defined skills attached to agents, passed at runtime, injected into LLM context)
- **Modular monolith \+ vertical slice (backend)**
- **Frontend using latest stable Svelte (SvelteKit)**
- ## deterministic convergence system (not chat)

  # **1\. System Objective**

Transform an idea → iterative multi-agent validation → **converged execution blueprint**

Outputs:

- `architecture.md`
- ## `roadmap.md`

  # **2\. High-Level Architecture**

```
frontend/ (SvelteKit)
       ↓
backend/ (Go 1.26 modular monolith)
       ↓
┌──────────────────────── A2A ────────────────────────────┐
↓               ↓               ↓               ↓
Agent 1 (Go)  Agent 2 (Go)  Agent 3 (Go)   Agent N (Go)
(a2a-go/v2)   (a2a-go/v2)   (a2a-go/v2)    (a2a-go/v2)
LLM: Copilot  LLM: Claude   LLM: Claude    LLM: any
Model: X      Model: Y      Model: Z

PostgreSQL ← canonical state + agent registry
LLM Config Resolver → per-agent credentials (env vars only)
Markdown generator → .md outputs
```

Min 2 agents per session. Agents selected at session creation from the registry.

---

# **3\. Backend Architecture**

## **Root Structure**

```
backend/
 cmd/
   server/
 internal/
   platform/
   shared/
 modules/
   session/
   iteration/
   agent/
   state/
   convergence/
   markdown/
```

---

# **4\. Backend Design Approach**

## **Modular Monolith**

- single deployable
- clear module boundaries
- ## avoids distributed complexity

  ## **Vertical Slice**

Each module owns:

- handler
- service
- repository
- ## domain logic

  # **5\. Internal Platform Layer**

```
internal/platform/
 http/
 a2a/
 llm/
 db/
 config/
 logger/
```

---

## **A2A Layer**

SDK: `github.com/a2aproject/a2a-go/v2` (requires Go ≥ 1.24.4)

Install:

```
go get github.com/a2aproject/a2a-go/v2
```

Key packages used by this project:

| Package     | Role in this project                                                     |
| ----------- | ------------------------------------------------------------------------ |
| `a2a`       | Core types: `Message`, `Task`, `Part`, `AgentCard`, `AgentSkill`, events |
| `a2asrv`    | Agent-side: `AgentExecutor` interface, HTTP/gRPC/JSON-RPC handler setup  |
| `a2aclient` | Backend-side: connect to an agent via `AgentCard`, send messages         |
| `a2agrpc`   | gRPC transport handler (optional — REST/JSON-RPC also supported)         |

**Backend → Agent call flow (via `a2aclient`):**

1. Resolve agent's `AgentCard` from `{agent.endpoint}/.well-known/agent.json`
2. Create transport-agnostic client: `a2aclient.NewFromCard(ctx, card, opts...)`
3. Assemble domain context (role, state, skills, llm_config) as a `DataPart`
4. Send: `client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})`
5. Receive `SendMessageResult` → extract updated state from response `Artifact.Parts`

**Agent-side server setup (`a2asrv`):**

```go
executor := &BrainstormExecutor{} // implements a2asrv.AgentExecutor
handler  := a2asrv.NewHandler(executor, opts...)
http.Handle("/", a2asrv.NewRESTHandler(handler))
http.ListenAndServe(":8080", nil)
```

`internal/platform/a2a/` wraps both the client factory and server setup.
Responsibilities: AgentCard resolution, client lifecycle, retry on transient errors.

## **LLM Abstraction**

```
type LLMProvider interface {
   Generate(ctx context.Context, req LLMRequest) (LLMResponse, error)
}
```

Providers:

- GitHub Copilot (primary)
- Claude (ready by interface)

### **LLM Config (Tiered Resolver)**

Per-agent LLM config resolves in this priority order:

```
session override → agent-level config → global default
```

Config shape:

```go
type LLMConfig struct {
    Provider      string // "copilot" | "claude"
    Model         string // e.g. "claude-opus-4", "gpt-4o"
    CredentialRef string // name of env var — never the key itself
}
```

Resolved by `internal/platform/llm/resolver.go` at call time.

### **Credential Security Rules**

- API keys are **never stored in the DB or config files**
- `CredentialRef` holds only the **env var name** (e.g. `CLAUDE_API_KEY`)
- Platform resolves the actual key at runtime via `os.Getenv(credentialRef)`
- If the env var is absent at startup → agent is marked **unavailable**; no silent fallback
- ## `llm_config` JSONB column stores only `{provider, model, credential_ref}` — never the key value

  ## **DB Layer**

- PostgreSQL
- ## use pgx / sqlc (no heavy ORM)

  # **6\. Backend Modules**

  ***

  ## **session/**

```
modules/session/
 handler.go
 service.go
 repository.go
 model.go
```

Handles:

- create session with selected agent list
- store idea
- session lifecycle

### **Session-Agent Binding**

When a session is created, the user selects agents from the registry.
Each binding is stored in a `session_agents` join table:

```sql
CREATE TABLE session_agents (
    session_id   UUID NOT NULL REFERENCES sessions(id),
    agent_id     UUID NOT NULL REFERENCES agents(id),
    position     INT  NOT NULL,  -- pipeline execution order (0, 1, 2, …)
    role         TEXT NOT NULL,  -- assigned role from catalog
    llm_override JSONB,          -- optional per-session LLM config override
    PRIMARY KEY (session_id, agent_id)
);
```

- `position` determines pipeline order
- `role` assigned at session creation from the role catalog
- `llm_override` merges with agent-level config (session layer wins)
- ## Minimum 2 agents enforced at session start

  ## **iteration/**

```
modules/iteration/
 handler.go
 service.go
 engine.go
```

Handles:

- iteration trigger
- ## loop execution

  ## **agent/**

Full vertical slice — owns the agent registry and A2A dispatch.

```
modules/agent/
 handler.go     ← CRUD API for agent definitions + skills
 service.go     ← register, list, validate agents; attach/detach skills; dispatch
 repository.go  ← sqlc DB queries
 model.go       ← Agent, Role, LLMConfig, Skill domain types
 client.go      ← A2A dispatch: resolves AgentCard, builds a2aclient, sends message
 role.go        ← Role catalog (typed constants)
```

### **Role Catalog**

```go
type Role string

const (
    RoleBuilder        Role = "build"
    RoleReviewer       Role = "review"
    RoleRefiner        Role = "refine"
    RoleDevilsAdvocate Role = "devils_advocate"
)
```

### **Skill Model**

A skill is a named capability that shapes how an agent approaches its role.
Skills are injected into the agent's system prompt at call time — they are **prompt-level** behaviors, not external tool calls.

```go
type Skill struct {
    ID          uuid.UUID
    Name        string    // e.g. "Security Review", "Cost Optimization"
    Description string    // plain-language description injected into prompt
    Prompt      string    // additional system prompt fragment appended when skill is active
    CreatedAt   time.Time
}
```

### **DB Tables**

```sql
CREATE TABLE agents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL UNIQUE,
    description    TEXT,
    default_role   TEXT NOT NULL,
    system_prompt  TEXT,
    llm_config     JSONB,  -- {provider, model, credential_ref} only
    endpoint       TEXT NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE skills (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    prompt      TEXT NOT NULL,  -- injected into agent system prompt when active
    created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE agent_skills (
    agent_id   UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    skill_id   UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (agent_id, skill_id)
);
```

### **Skill Injection at Runtime**

Before dispatching to an agent, `client.go` assembles the effective system prompt:

```
[agent.system_prompt]
+ [skill_1.prompt]
+ [skill_2.prompt]
+ ...
```

This assembled prompt, together with the resolved `LLMConfig`, role, and current state, is packed into the A2A `Message.Parts` as a `DataPart` and sent via `a2aclient`. The agent binary receives the assembled context through the SDK's `ExecutorContext` — it does not resolve skills or LLM config itself.

**Dispatch pseudocode (`client.go`):**

```go
// 1. Resolve tiered LLM config
llmCfg := resolver.Resolve(globalCfg, agentCfg, sessionOverride)

// 2. Assemble skill prompt fragments
systemPrompt := buildSystemPrompt(agent.SystemPrompt, activeSkills)

// 3. Build A2A message with domain context as DataPart
payload := BrainstormPayload{
    Role:         agent.Role,
    LLMConfig:    llmCfg,
    SystemPrompt: systemPrompt,
    State:        currentState,
}
msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(payload))

// 4. Resolve AgentCard and dispatch
card, _ := agentcard.DefaultResolver.Resolve(ctx, agent.Endpoint)
client, _ := a2aclient.NewFromCard(ctx, card)
result, _ := client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})

// 5. Extract updated state from response artifact
updatedState := extractState(result)
```

Handles:

- agent registry CRUD
- skill registry CRUD
- agent ↔ skill attachment / detachment
- A2A call dispatch (ordered pipeline)
- per-agent LLM config resolution via tiered resolver
- role assignment from catalog
- skill prompt assembly before LLM call

  ## **state/**

```
modules/state/
 model.go
 merge.go
 validator.go
```

Handles:

- canonical state
- merge logic
- ## validation

  ## **convergence/**

```
modules/convergence/
 engine.go
```

Handles:

- ## convergence detection

  ## **markdown/**

```
modules/markdown/
 generator.go
```

Handles:

- architecture.md
- ## roadmap.md

  # **7\. Agents (Separate Services)**

All agent instances share one identical codebase. Config and role are injected at runtime per A2A request.

Structure:

```
agent/
 cmd/server/
 internal/
   executor/      ← implements a2asrv.AgentExecutor
   llm/           ← LLMProvider implementation (Copilot, Claude)
   config/        ← reads env vars; resolves LLMConfig at startup
 agentcard.go    ← declares a2a.AgentCard (name, description, capabilities)
```

---

## **Agent Behavior**

- identical implementation — any instance can take any role
- role and LLM config injected per A2A request
- ## minimum 2 agents required per session; no upper limit

## **Role Catalog**

| Role              | Behavior                                           |
| ----------------- | -------------------------------------------------- |
| `build`           | Proposes / expands architecture and execution plan |
| `review`          | Critiques output, identifies risks and gaps        |
| `refine`          | Synthesizes prior outputs, removes contradictions  |
| `devils_advocate` | Challenges assumptions, surfaces edge cases        |

Roles are assigned at session creation. Distribution by agent count:

| Agents selected | Role assignment                        |
| --------------- | -------------------------------------- |
| 2               | build, review                          |
| 3               | build, review, refine                  |
| 4               | build, review, refine, devils_advocate |
| 5+              | cycles catalog; extras get review      |

User can override any agent’s role at session creation.

---

## **A2A Interaction Model**

The SDK does **not** use a custom JSON task schema. Communication is message-based via `a2a.SendMessageRequest` → `a2a.Message` → `a2a.Part`.

**Backend sends** (via `a2aclient`):

```go
// BrainstormPayload is our domain-specific DataPart content.
type BrainstormPayload struct {
    Role         string    `json:"role"`           // "build" | "review" | ...
    SystemPrompt string    `json:"system_prompt"`  // assembled: agent prompt + skill fragments
    LLMConfig    LLMConfig `json:"llm_config"`     // resolved tiered config
    State        any       `json:"state"`          // current canonical state
}

msg := a2a.NewMessage(a2a.MessageRoleUser, a2a.NewDataPart(BrainstormPayload{
    Role:         "build",
    SystemPrompt: assembledPrompt, // skills already merged in by backend
    LLMConfig:    resolvedCfg,
    State:        currentState,
}))
client.SendMessage(ctx, &a2a.SendMessageRequest{Message: msg})
```

**Agent receives** (via `a2asrv.AgentExecutor`):

```go
func (e *BrainstormExecutor) Execute(
    ctx context.Context,
    execCtx *a2asrv.ExecutorContext,
) iter.Seq2[a2a.Event, error] {
    return func(yield func(a2a.Event, error) bool) {
        // 1. Extract domain payload from DataPart
        var payload BrainstormPayload
        for _, part := range execCtx.Message.Parts {
            if d := part.Data(); d != nil {
                // unmarshal d into payload
            }
        }

        // 2. Call LLM with assembled system prompt
        resp, err := e.llm.Generate(ctx, LLMRequest{
            SystemPrompt: payload.SystemPrompt, // skills already in prompt
            UserMessage:  marshalState(payload.State),
        })

        // 3. Emit task status + artifact with updated state
        yield(a2a.NewSubmittedTask(execCtx, nil), nil)
        yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateWorking, nil), nil)
        artifact := a2a.NewArtifactEvent(execCtx, a2a.NewDataPart(updatedState))
        yield(artifact, nil)
        yield(a2a.NewStatusUpdateEvent(execCtx, a2a.TaskStateCompleted, nil), nil)
    }
}
```

The agent binary **does not** know about skill names, credential refs, or DB records. It receives the fully assembled `SystemPrompt` and operates on the provided `State`.

---

# **8\. Canonical State Model**

```json
{
  "idea": {},
  "architecture": {},
  "execution_plan": [],
  "risks": [],
  "assumptions": [],
  "open_questions": [],
  "metrics": {
    "confidence": 0.0
  },
  "meta": {
    "iteration": 0,
    "agents": [
      {
        "agent_id": "uuid",
        "name": "Agent Alpha",
        "role": "build",
        "provider": "claude",
        "model": "claude-opus-4",
        "skills": ["Security Review", "Cost Optimization"]
      }
    ]
  }
}
```

`meta.agents` is a dynamic list populated from `session_agents` at session creation.
Length ≥ 2 is enforced at session start. Fixed keys `agentA`/`agentB` are removed.
`skills` lists the names of active skills for observability — prompts are not stored in state.

---

# **9\. Iteration Engine (Deterministic)**

Agents run as an ordered pipeline. Each agent receives the cumulative output of the previous.
Roles are fixed per session (assigned at creation — no runtime alternation).

```go
agents := session.GetOrderedAgents() // min 2, ordered by session_agents.position

for i := 1; i <= maxIter; i++ {
    current := state

    for _, agent := range agents {
        // each agent builds on the previous agent's output
        out := agent.Call(agent, agent.Role, current)
        current = out
    }

    newState := Merge(state, current)

    if convergence.Check(state, newState) {
        break
    }

    state = newState
}
```

---

# **10\. Merge Strategy**

- union risks
- deduplicate
- remove resolved
- ## reject vague outputs

  ## **Stability Rule**

- agreement → lock
- ## persistent conflict → user resolves

  # **11\. Convergence**

Stop when:

- no new critical risks
- execution plan complete
- confidence stabilizes
- OR user approves
- ## OR max iterations

  # **12\. Frontend Architecture (SvelteKit)**

  ***

  ## **Stack**

- **SvelteKit (latest stable)**
- TypeScript
- TailwindCSS
- TanStack Query (Svelte Query)
- ## Zustand alternative → Svelte stores

  ## **Folder Structure**

```
frontend/
 src/
   routes/
     +page.svelte
     session/[id]/+page.svelte
     agents/+page.svelte          ← agent registry management
     skills/+page.svelte          ← skill library management
   lib/
     components/
       AgentPanel.svelte
       AgentSelector.svelte
       SkillManager.svelte        ← create/edit skills + assign to agents
       ControlPanel.svelte
       StateView.svelte
       Timeline.svelte
     stores/
       sessionStore.ts
       agentRegistryStore.ts
     services/
       api.ts
```

---

# **13\. UI Layout (Structured Workspace)**

Agent panels render dynamically based on the number of agents selected for the session (min 2).

```
-----------------------------------------------------------------
| Agent 1 Panel   | Agent 2 Panel   | Agent N Panel             |
| Role: build     | Role: review    | Role: refine              |
| LLM: Copilot/X  | LLM: Claude/Y   | LLM: Claude/Z             |
-----------------------------------------------------------------
| Control Panel (Idea input / Start / Next Iteration / Approve) |
-----------------------------------------------------------------
| State View (Architecture / Execution Plan / Risks)            |
-----------------------------------------------------------------
| Iteration Timeline                                            |
-----------------------------------------------------------------
```

- Panels scroll horizontally when N ≥ 4
- ## Each panel shows: agent name, role badge, LLM provider + model, active skills, current output, diff vs previous iteration

  # **14\. Frontend Components**

  ***

  ## **AgentPanel.svelte**

Renders one panel per active session agent (dynamic, not hardcoded A/B).

Props: `agent: SessionAgent` (id, name, role, provider, model, skills[], output)

Displays:

- agent name + role badge
- LLM provider + model label
- skill tags (active skills for this agent)
- structured output (current iteration)
- ## diff highlight vs previous iteration

  ## **AgentSelector.svelte**

Used at session creation. Displays the agent registry, allows user to:

- pick which agents participate
- assign or override roles
- optionally override LLM model per agent for the session
- ## view and deselect per-agent skills for this session (skills default to agent definition; can be toggled off per session)

  ## **SkillManager.svelte**

Standalone page component (route: `/skills`). Allows users to:

- create new skills (name, description, prompt fragment)
- edit / delete existing skills
- attach or detach skills from agents in the agent registry

Displays:

- skill library list
- per-skill: name, description, assigned agents
- ## per-agent: list of currently assigned skills with remove toggle

  ## **ControlPanel.svelte**

Controls:

- Start (with agent selection + idea input)
- Next Iteration
- Approve
- ## Inject feedback

  ## **StateView.svelte**

Shows:

- architecture
- execution plan
- ## risks

  ## **Timeline.svelte**

Shows:

- iteration history
- ## per-agent role per iteration

  # **15\. Frontend State Strategy**

Use Svelte stores:

```ts
sessionStore = {
  session_id,
  idea,
  state,
  iteration,
  agents, // SessionAgent[] — ordered list for active session (includes skills[])
  loading,
};

agentRegistryStore = {
  agents, // Agent[] — full registry (each agent includes assigned skills[])
  skills, // Skill[] — full skill library
  loading,
};
```

---

# **16\. API Integration**

**Skill Registry:**

```
POST   /skills                          ← create skill
GET    /skills                          ← list all skills
GET    /skills/{id}
PUT    /skills/{id}
DELETE /skills/{id}
POST   /agents/{id}/skills/{skill_id}   ← attach skill to agent
DELETE /agents/{id}/skills/{skill_id}   ← detach skill from agent
GET    /agents/{id}/skills              ← list skills for an agent
```

**Agent Registry:**

```
POST   /agents           ← register agent
GET    /agents           ← list all agents (includes skills[] per agent)
GET    /agents/{id}
PUT    /agents/{id}
DELETE /agents/{id}
```

**Session (updated):**

```
POST   /sessions               ← body includes agent_ids[] + idea + optional skill overrides
POST   /sessions/{id}/iterate
GET    /sessions/{id}
POST   /sessions/{id}/finalize
```

`POST /sessions` request body:

```json
{
  "idea": "...",
  "agent_ids": ["uuid-1", "uuid-2"],
  "role_overrides": {
    "uuid-1": "build",
    "uuid-2": "review"
  },
  "llm_overrides": {
    "uuid-1": { "model": "claude-opus-4", "credential_ref": "CLAUDE_API_KEY" }
  },
  "skill_overrides": {
    "uuid-1": ["skill-uuid-a", "skill-uuid-b"],
    "uuid-2": []
  }
}
```

`skill_overrides` is optional. When omitted, the agent's default attached skills are used. An empty array `[]` explicitly disables all skills for that agent in this session.

```

---

# **17\. Output Artifacts**

Generated:

- architecture.md
- ## roadmap.md

  # **18\. Deployment**

  ## **MVP**

- docker-compose
- services:
  - backend
  - agent (shared image — scale with `--scale agent=N` or run named instances)
  - postgres

  ***

  # **19\. Failure Modes**

  ***

  ## **Oscillation**

- ## fix: stability bias \+ user intervention

  ## **Weak outputs**

- ## fix: strong prompt contracts

  ## **Schema drift**

- ## fix: strict validation

  ## **LLM inconsistency (Copilot)**

- fix:
  - low temperature
  - strict JSON schema

  ***

  # **20\. What You Are Building**

**Multi-agent system design IDE (deterministic, not conversational)**
```

---

# **21. Frontend Design System Specification (v1.1)**

> Added in PLAN.md v1.1. Source of truth for the polished UI implemented in Tasks 16–25.
> Reference mockup: `frontend/mockups/future-polished-mockup.html`

## **Visual Language**

The UI uses a warm-to-cool glassmorphism aesthetic — cream-toned backgrounds that shift to blue-grey, frosted-glass panels, and a consistent cyan/blue accent palette. All design decisions derive from a single set of CSS custom properties defined in `frontend/src/app.css`.

**Color tokens:**

| Token        | Value     | Use                                        |
| ------------ | --------- | ------------------------------------------ |
| `--bg-0`     | `#f5efe4` | Warm cream — page background base          |
| `--bg-1`     | `#e8ecf7` | Cool blue-grey — page background accent    |
| `--ink-900`  | `#151b2f` | Near-black — primary text                  |
| `--ink-700`  | `#2d3655` | Dark — secondary headings                  |
| `--ink-500`  | `#5a6282` | Mid — secondary text, labels               |
| `--ink-300`  | `#a8aec7` | Light — placeholders, dividers             |
| `--accent`   | `#0bb6d9` | Cyan — primary interactive, gradient start |
| `--accent-2` | `#1f7ae0` | Blue — gradient end, links                 |
| `--ok`       | `#1b9f66` | Green — success, done state, live chip     |
| `--warn`     | `#d48806` | Amber — warning, review role badge         |
| `--danger`   | `#ce3158` | Red — error, delete actions                |

**Typography stack:**

- Body text: **IBM Plex Sans** (300, 400, 500 weights)
- Code / monospace blocks: **IBM Plex Mono** (400)
- Headings / UI labels: **Space Grotesk** (500, 700)

**Background:**

```css
background:
  radial-gradient(1200px 600px at 10% 10%, #fff8ec, transparent),
  radial-gradient(900px 500px at 90% 10%, #e8f7ff, transparent),
  linear-gradient(135deg, #f5efe4, #e8ecf7);
```

**Glassmorphism panels and cards:**

- `.panel`: `background: rgba(255,255,255,0.72)`, `backdrop-filter: blur(8px)`, `border-radius: 18px`, `box-shadow: 0 10px 30px rgba(35,46,82,0.1)`, 1px white border
- `.card`: same fill + blur, `border-radius: 14px`, lighter shadow, `padding: 20px`

---

## **Views and Route Map**

| Route                    | Purpose                                                                        |
| ------------------------ | ------------------------------------------------------------------------------ |
| `/`                      | Session creation — idea input, agent pool selector, max iterations             |
| `/session/[id]`          | Session workspace — sequential pipeline, pass summary bar, state + risk panels |
| `/session/[id]/finalize` | Export view — generation log, architecture.md + roadmap.md download            |
| `/settings`              | Unified agents / skills / roles management (3 tabs)                            |
| `/settings/agent/new`    | Create agent form                                                              |
| `/settings/agent/[id]`   | Edit agent form                                                                |
| `/settings/skill/new`    | Create skill form                                                              |
| `/settings/skill/[id]`   | Edit skill form                                                                |
| `/history`               | Session history — stat cards + searchable table                                |
| `/agents`                | Redirects → `/settings?tab=agents`                                             |
| `/skills`                | Redirects → `/settings?tab=skills`                                             |

---

## **Home View (/) — Session Creation**

- Sticky glassmorphism topbar: logo "A2A Brainstorm" + nav ("Session History" → `/history`, "⚙ Settings" → `/settings`) + animated Live chip (green pulsing dot)
- Hero `.panel` (max-width 920px, centered in `.artboard`): idea textarea + char count
- 2-column grid: left = Max Iterations input (number, 1–20, default 5); right = Agent Pool checkbox list — one row per registered agent, showing name + role badge + provider/model label
- "Start Session" gradient button (disabled + spinner while loading; disabled if < 2 agents checked)
- "Estimated runtime: ~N min" hint below button (computed as `checkedAgents × iterations × 0.5`)
- Inline validation: soft red border on agent pool if < 2 selected

---

## **Session Workspace (/session/[id]) — Sequential Pipeline View**

- Sticky pass summary bar: "Pipeline Pass N / M" label + agent count chip + `ConfidenceBar` component + shimmer while loading
- Vertical sequential pipeline panel: one `PipelineStage` per agent, separated by connector lines (solid between done stages, dashed before waiting stages)
- Per stage states:
  - **done** (`stage-done`): green check icon, role badge, mono log block (dark bg `#1a1d2e`, IBM Plex Mono), green summary block
  - **running** (`stage-running`): animated blinking dots, mono log block with cursor
  - **waiting** (`stage-waiting`): dimmed 50% opacity, dashed border
- Bottom split (2/3 + 1/3): `CanonicalStatePanel` (idea, architecture, plan accordion, assumptions, open questions) + `RiskBoard` (risks with severity chips)
- Sticky control bar at bottom: "Run Next Iteration" (disabled while loading/converged) + "Inject Feedback" (inline textarea toggle) + "Finalize Session" → `/session/[id]/finalize`

---

## **Export View (/session/[id]/finalize)**

- "Generate Documents" button triggers `POST /sessions/{id}/finalize`
- Animated streaming log panel (dark `#1a1d2e` background, monospace, lines append at 400ms intervals): "Analyzing canonical state...", "Extracting architecture decisions...", "Generating architecture.md...", "Generating roadmap.md...", "Done ✓"
- Two output cards appear after generation: `architecture.md` + `roadmap.md` — each with preview textarea (`readonly`), Copy (clipboard API) + Download (`Blob` → `<a download>`) buttons
- "Download All" + "New Session" buttons in done bar

---

## **Settings View (/settings) — 3-Tab Panel**

- Tab bar: Agents | Skills | Roles
- **Agents tab**: table — Name, Default Role (badge), Provider/Model, Skills count chip, Status (`.chip-ok` / `.chip-warn`), Edit + Delete actions; "Add Agent" button → `/settings/agent/new`
- **Skills tab**: table — Name, Description (truncated), Used By (N agents chip), Edit + Delete; "Add Skill" → `/settings/skill/new`
- **Roles tab**: 4 read-only role cards (BUILD, REVIEW, REFINE, DEVILS ADVOCATE) — each with behavior description and "System Role" chip; "Custom roles coming soon" callout

---

## **Session History View (/history)**

- 4 stat cards: Sessions Completed, Avg Confidence, Docs Generated, Avg Iterations
- Live search input (client-side filter by idea text)
- Sessions table: Title (truncated idea), Date, Iterations, Confidence pill (green ≥ 0.8, amber ≥ 0.5, red < 0.5), Status chip, "View →" action

---

## **Shared Components (v1.1 additions)**

| Component             | File                                        | Props                                                               | Purpose                                                                 |
| --------------------- | ------------------------------------------- | ------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `PipelineStage`       | `lib/components/PipelineStage.svelte`       | `agent`, `status`, `output`, `summary`                              | Replaces `AgentPanel` — vertical stage with done/running/waiting states |
| `ConfidenceBar`       | `lib/components/ConfidenceBar.svelte`       | `value`, `animating`                                                | Segmented progress bar showing confidence %                             |
| `CanonicalStatePanel` | `lib/components/CanonicalStatePanel.svelte` | `state`                                                             | Replaces `StateView` — card-based canonical state display               |
| `RiskBoard`           | `lib/components/RiskBoard.svelte`           | `risks`                                                             | Risk list with severity chips                                           |
| `WarningModal`        | `lib/components/WarningModal.svelte`        | `open`, `title`, `body`, `confirmLabel`, `confirmDanger`, callbacks | Global guarded-action modal                                             |

**Deprecated components (kept for build compatibility):**

| Old Component         | Replacement                      |
| --------------------- | -------------------------------- |
| `AgentPanel.svelte`   | `PipelineStage.svelte`           |
| `ControlPanel.svelte` | Inline in session page           |
| `StateView.svelte`    | `CanonicalStatePanel.svelte`     |
| `Timeline.svelte`     | Pass summary bar in session page |

---

## **Backend Additions for UI (v1.1)**

| Change                       | Endpoint / File                         | Purpose                                                                                   |
| ---------------------------- | --------------------------------------- | ----------------------------------------------------------------------------------------- |
| Add `GET /sessions`          | `session/handler.go` + `repository.go`  | Session list for history view                                                             |
| Return content in finalize   | `POST /sessions/{id}/finalize` response | `architecture_markdown` + `roadmap_markdown` strings for download                         |
| `GenerateContent()` function | `markdown/generator.go`                 | Returns markdown strings instead of writing files; `WriteArtifacts` calls this internally |

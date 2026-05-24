// Package markdown — README.md document generator.
// GenerateReadme follows the §8.20 section skeleton and wraps the body in
// enforceMinLines so the output is always ≥ 1000 lines.
package markdown

import (
	"fmt"
	"strings"

	"a2a-brainstorm/backend/internal/modules/state"
)

// GenerateReadme renders a README.md-style document from s.
// It follows the §8.20 section skeleton and enforces a minimum of 1000 lines
// via the padReadme padder.
func GenerateReadme(s state.CanonicalState) (string, error) {
	var b strings.Builder

	// ── Title + One-line Description ─────────────────────────────────────────
	title := "Project"
	description := "A brainstorm project."
	if text, ok := s.Idea["text"]; ok {
		title = fmt.Sprintf("%v", text)
		description = fmt.Sprintf("%v", text)
	}
	b.WriteString(fmt.Sprintf("# %s\n\n", title))
	b.WriteString(fmt.Sprintf("> %s\n\n", description))

	// Badges row (placeholder — no live URLs, all static).
	b.WriteString("![License: MIT](https://img.shields.io/badge/license-MIT-blue)\n")
	b.WriteString("![Build](https://img.shields.io/badge/build-passing-brightgreen)\n")
	b.WriteString("![Coverage](https://img.shields.io/badge/coverage-80%25-green)\n")
	b.WriteString("![Go Version](https://img.shields.io/badge/go-1.26-blue)\n\n")

	// ── Table of Contents ────────────────────────────────────────────────────
	b.WriteString("## Table of Contents\n\n")
	b.WriteString("- [Overview](#overview)\n")
	b.WriteString("- [System Architecture](#system-architecture)\n")
	b.WriteString("- [Repository Structure](#repository-structure)\n")
	b.WriteString("- [Prerequisites](#prerequisites)\n")
	b.WriteString("- [Quick Start](#quick-start)\n")
	b.WriteString("- [Configuration](#configuration)\n")
	b.WriteString("- [Testing](#testing)\n")
	b.WriteString("- [Risk & Assumptions](#risk--assumptions)\n")
	b.WriteString("- [Roadmap](#roadmap)\n")
	b.WriteString("- [Documentation](#documentation)\n")
	b.WriteString("- [Contributing](#contributing)\n")
	b.WriteString("- [License](#license)\n\n")

	// ── Overview ─────────────────────────────────────────────────────────────
	b.WriteString("## Overview\n\n")
	if len(s.Idea) > 0 {
		if text, ok := s.Idea["text"]; ok {
			b.WriteString(fmt.Sprintf("%v\n\n", text))
		} else {
			writeMap(&b, s.Idea)
		}
	} else {
		b.WriteString("_Project overview not yet defined._\n\n")
	}
	if len(s.Architecture) > 0 {
		b.WriteString("### Architecture Summary\n\n")
		writeMap(&b, s.Architecture)
	}
	b.WriteString(fmt.Sprintf("> Generated at iteration **%d** with confidence **%.4f**.\n\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	// ── System Architecture ──────────────────────────────────────────────────
	b.WriteString("## System Architecture\n\n")
	b.WriteString(renderASCIIComponents(s))
	if len(s.Architecture) > 0 {
		b.WriteString("### Component Descriptions\n\n")
		writeMap(&b, s.Architecture)
	}

	// ── Repository Structure ─────────────────────────────────────────────────
	b.WriteString("## Repository Structure\n\n")
	b.WriteString(renderDirectoryTree(s))

	// ── Prerequisites ────────────────────────────────────────────────────────
	b.WriteString("## Prerequisites\n\n")
	b.WriteString(renderTechStack(s))
	b.WriteString("Install the required tools:\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Go — https://go.dev/dl/\n")
	b.WriteString("go version  # must be ≥ 1.26\n\n")
	b.WriteString("# Node.js — https://nodejs.org/\n")
	b.WriteString("node --version  # must be ≥ 20\n")
	b.WriteString("pnpm --version  # must be ≥ 9\n\n")
	b.WriteString("# Docker — https://www.docker.com/\n")
	b.WriteString("docker --version\n")
	b.WriteString("docker compose version\n")
	b.WriteString("```\n\n")

	// ── Quick Start ──────────────────────────────────────────────────────────
	b.WriteString("## Quick Start\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# 1. Clone the repository\n")
	b.WriteString("git clone <repository-url> && cd <project>\n\n")
	b.WriteString("# 2. Start infrastructure\n")
	b.WriteString("docker compose up -d\n\n")
	b.WriteString("# 3. Apply database migrations\n")
	b.WriteString("cd backend && go run ./cmd/migrate up && cd ..\n\n")
	b.WriteString("# 4. Configure environment\n")
	b.WriteString("cp .env.example .env\n")
	b.WriteString("# Edit .env with your API keys\n\n")
	b.WriteString("# 5. Run all services\n")
	b.WriteString("make dev\n")
	b.WriteString("# Or individually:\n")
	b.WriteString("#   cd backend && go run ./cmd/server    (port 8080)\n")
	b.WriteString("#   cd agent   && go run ./cmd/server    (port 9000)\n")
	b.WriteString("#   cd frontend && pnpm dev              (port 5173)\n")
	b.WriteString("```\n\n")
	b.WriteString("Open [http://localhost:5173](http://localhost:5173) to access the UI.\n\n")

	// ── Configuration ────────────────────────────────────────────────────────
	b.WriteString("## Configuration\n\n")
	b.WriteString("All configuration is driven by environment variables. Copy `.env.example` to `.env`:\n\n")
	b.WriteString(renderEnvVarList(s))
	b.WriteString("### Key Variables\n\n")
	b.WriteString("| Variable | Required | Default | Description |\n")
	b.WriteString("|----------|----------|---------|-------------|\n")
	b.WriteString("| `DATABASE_URL` | Yes | — | PostgreSQL connection string |\n")
	b.WriteString("| `COPILOT_API_KEY` | Conditional | — | GitHub Copilot API key |\n")
	b.WriteString("| `CLAUDE_API_KEY` | Conditional | — | Anthropic Claude API key |\n")
	b.WriteString("| `GLOBAL_LLM_PROVIDER` | No | `copilot` | Default LLM provider |\n")
	b.WriteString("| `GLOBAL_LLM_MODEL` | No | `gpt-4o` | Default model identifier |\n")
	b.WriteString("| `AGENT_ENDPOINT_0` | Yes | — | URL of the first agent binary |\n")
	b.WriteString("| `MAX_ITERATIONS` | No | `10` | Maximum iteration cycles |\n")
	b.WriteString("| `CONVERGENCE_THRESHOLD` | No | `0.02` | Min confidence delta to stop |\n\n")

	// ── Testing ──────────────────────────────────────────────────────────────
	b.WriteString("## Testing\n\n")
	b.WriteString("```bash\n")
	b.WriteString("# Run all backend tests\n")
	b.WriteString("cd backend && go test ./...\n\n")
	b.WriteString("# Run agent tests\n")
	b.WriteString("cd agent && go test ./...\n\n")
	b.WriteString("# Run frontend tests\n")
	b.WriteString("cd frontend && pnpm test\n\n")
	b.WriteString("# Run all tests (convenience target)\n")
	b.WriteString("make test\n")
	b.WriteString("```\n\n")
	if len(s.ExecutionPlan) > 0 {
		b.WriteString("### Per-Step Validation\n\n")
		for i, step := range s.ExecutionPlan {
			b.WriteString(fmt.Sprintf("**Step %d — %s:**\n\n", i+1, step.Title))
			b.WriteString("- Unit tests covering new functions.\n")
			b.WriteString("- `go vet ./...` and `go build ./...` pass.\n\n")
		}
	}

	// ── Risk & Assumptions ───────────────────────────────────────────────────
	b.WriteString("## Risk & Assumptions\n\n")
	b.WriteString("### Risks\n\n")
	b.WriteString(renderRisksTable(s))
	b.WriteString("### Assumptions\n\n")
	if len(s.Assumptions) > 0 {
		for _, a := range s.Assumptions {
			b.WriteString(fmt.Sprintf("- %s\n", a))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("_No assumptions recorded._\n\n")
	}

	// ── Roadmap ──────────────────────────────────────────────────────────────
	b.WriteString("## Roadmap\n\n")
	b.WriteString(renderExecutionPlanList(s))

	// ── Documentation ────────────────────────────────────────────────────────
	b.WriteString("## Documentation\n\n")
	b.WriteString("| Document | Path | Description |\n")
	b.WriteString("|----------|------|-------------|\n")
	b.WriteString("| Architecture | `architecture.md` | Full system design |\n")
	b.WriteString("| Roadmap | `roadmap.md` | Implementation timeline |\n")
	b.WriteString("| Plan | `PLAN.md` | Task-by-task implementation plan |\n")
	b.WriteString("| Startup Guide | `docs/STARTUP_GUIDE.md` | Local development setup |\n\n")

	// ── Contributing ─────────────────────────────────────────────────────────
	b.WriteString("## Contributing\n\n")
	b.WriteString("1. Fork the repository and create a feature branch.\n")
	b.WriteString("2. Write failing tests first (RED → GREEN → REFACTOR).\n")
	b.WriteString("3. Run `make test lint build` and ensure 0 failures.\n")
	b.WriteString("4. Open a pull request with a clear description.\n")
	b.WriteString("5. Ensure the PR description references the relevant task from `PLAN.md`.\n\n")
	b.WriteString("### Code Style\n\n")
	b.WriteString("- Go: follow `gofmt` + `go vet`; use `log/slog` for logging.\n")
	b.WriteString("- TypeScript/Svelte: follow project ESLint config; no inline `fetch()`.\n")
	b.WriteString("- SQL: parameterised queries only; append-only migrations.\n\n")

	// ── License ──────────────────────────────────────────────────────────────
	b.WriteString("## License\n\n")
	b.WriteString("MIT License. See [LICENSE](LICENSE) for details.\n\n")
	b.WriteString("```\n")
	b.WriteString("MIT License\n\n")
	b.WriteString(fmt.Sprintf("Copyright (c) %d — AI-Generated Project\n\n", 2024+(s.Meta.Iteration/10)))
	b.WriteString("Permission is hereby granted, free of charge, to any person obtaining a copy\n")
	b.WriteString("of this software and associated documentation files (the \"Software\"), to deal\n")
	b.WriteString("in the Software without restriction, including without limitation the rights\n")
	b.WriteString("to use, copy, modify, merge, publish, distribute, sublicense, and/or sell\n")
	b.WriteString("copies of the Software, and to permit persons to whom the Software is\n")
	b.WriteString("furnished to do so, subject to the following conditions:\n\n")
	b.WriteString("The above copyright notice and this permission notice shall be included in all\n")
	b.WriteString("copies or substantial portions of the Software.\n\n")
	b.WriteString("THE SOFTWARE IS PROVIDED \"AS IS\", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR\n")
	b.WriteString("IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,\n")
	b.WriteString("FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE\n")
	b.WriteString("AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER\n")
	b.WriteString("LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,\n")
	b.WriteString("OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE\n")
	b.WriteString("SOFTWARE.\n")
	b.WriteString("```\n\n")

	b.WriteString(fmt.Sprintf("---\n_Generated at iteration %d. Confidence: %.4f._\n",
		s.Meta.Iteration, s.Metrics.Confidence))

	body := b.String()
	return enforceMinLines(body, s, padReadme), nil
}

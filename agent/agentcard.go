// Package agent declares the A2A AgentCard for the brainstorm agent binary.
// The card is served at /.well-known/agent-card.json and used by the backend
// to create a2aclient instances via a2aclient.NewFromCard.
package agent

import (
	"github.com/a2aproject/a2a-go/v2/a2a"

	"a2a-brainstorm/agent/internal/config"
)

// NewAgentCard returns the public A2A AgentCard for this binary.
// The base URL is read from the AGENT_PUBLIC_URL env var (via config.GetPublicURL),
// falling back to http://localhost:{AGENT_PORT}. In Docker Compose, set
// AGENT_PUBLIC_URL=http://agent:{AGENT_PORT} so the backend can reach the agent
// via the Docker service name instead of localhost.
func NewAgentCard() *a2a.AgentCard {
	baseURL := config.GetPublicURL()
	return &a2a.AgentCard{
		Name: "brainstorm-agent",
		Description: "A deterministic brainstorm pipeline agent that processes CanonicalState " +
			"through a role-based LLM pass and returns an updated state as a DataPart artifact.",
		Version: "1.0.0",
		SupportedInterfaces: []*a2a.AgentInterface{
			a2a.NewAgentInterface(baseURL, a2a.TransportProtocolHTTPJSON),
		},
		DefaultInputModes:  []string{"application/json"},
		DefaultOutputModes: []string{"application/json"},
		Skills: []a2a.AgentSkill{
			{
				ID:          "build",
				Name:        "Build",
				Description: "Proposes and expands architecture and execution plan.",
				Tags:        []string{"build", "architecture", "design"},
			},
			{
				ID:          "review",
				Name:        "Review",
				Description: "Critiques output, identifies risks and gaps.",
				Tags:        []string{"review", "critique", "quality"},
			},
			{
				ID:          "refine",
				Name:        "Refine",
				Description: "Synthesizes prior outputs, removes contradictions.",
				Tags:        []string{"refine", "synthesis", "consolidation"},
			},
			{
				ID:          "devils_advocate",
				Name:        "Devil's Advocate",
				Description: "Challenges assumptions, surfaces edge cases.",
				Tags:        []string{"devils_advocate", "critique", "challenge"},
			},
		},
	}
}

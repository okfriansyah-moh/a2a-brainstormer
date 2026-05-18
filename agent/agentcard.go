// Package agent declares the A2A AgentCard for the brainstorm agent binary.
// The card is served at /.well-known/agent-card.json and used by the backend
// to create a2aclient instances via a2aclient.NewFromCard.
package agent

import (
	"fmt"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

// NewAgentCard returns the public A2A AgentCard for this binary.
// port is the TCP port the agent is listening on; it is embedded in the
// SupportedInterfaces URL so clients can resolve the correct endpoint.
func NewAgentCard(port int) *a2a.AgentCard {
	baseURL := fmt.Sprintf("http://localhost:%d", port)
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

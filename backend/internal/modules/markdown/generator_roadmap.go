// Package markdown — roadmap generator deprecated in v1.8 (§8.29.5).
package markdown

import (
	"errors"

	"a2a-brainstorm/backend/internal/modules/state"
)

// ErrOutputKeyDeprecated is returned when GenerateRoadmap is invoked directly.
var ErrOutputKeyDeprecated = errors.New("roadmap output key deprecated: use plan")

// GenerateRoadmap is deprecated. Roadmap content is consolidated into plan.md.
func GenerateRoadmap(_ state.CanonicalState) (string, error) {
	return "", ErrOutputKeyDeprecated
}

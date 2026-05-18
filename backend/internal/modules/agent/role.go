// Package agent provides role distribution and validation helpers.
//
// DefaultRoles implements the assignment table from §8.13 of docs/PLAN.md:
//   - 2 agents: build, review
//   - 3 agents: build, review, refine
//   - 4 agents: build, review, refine, devils_advocate
//   - 5+ agents: cycles roleCatalog from position 0; extras assigned "review"
package agent

// roleCatalog is the fixed rotation order used by DefaultRoles.
var roleCatalog = []Role{
	RoleBuilder,
	RoleReviewer,
	RoleRefiner,
	RoleDevilsAdvocate,
}

// DefaultRoles distributes roles across agentCount agents following the table
// in §8.13.  Returns nil when agentCount < 2 (invalid — enforced by session layer).
func DefaultRoles(agentCount int) []Role {
	if agentCount < 2 {
		return nil
	}
	roles := make([]Role, agentCount)
	for i := range roles {
		if i < len(roleCatalog) {
			roles[i] = roleCatalog[i]
		} else {
			// Agents beyond the catalogue are assigned "review".
			roles[i] = RoleReviewer
		}
	}
	return roles
}

// ValidRole returns true if r is one of the known role constants.
// The handler layer uses this to reject unknown role values in HTTP requests.
func ValidRole(r Role) bool {
	switch r {
	case RoleBuilder, RoleReviewer, RoleRefiner, RoleDevilsAdvocate:
		return true
	default:
		return false
	}
}

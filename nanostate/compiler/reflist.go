package nanocms_compiler

import (
	"strings"
)

type RefList struct {
	included        map[string]bool
	referenced_jobs map[string]bool
	required_jobs   map[string]bool // Their content
}

func NewRefList() *RefList {
	return new(RefList)
}

// Get all mentioned references
func (rl *RefList) FindRefs(state *OTree) {
	rl.included = make(map[string]bool)        // State IDs
	rl.referenced_jobs = make(map[string]bool) // the entire blocks
	rl.required_jobs = make(map[string]bool)   // their content
	rl.findRefs(state)
}

func (rl *RefList) GetRequiredJobs() []string {
	refs := make([]string, 0)
	for k := range rl.required_jobs {
		refs = append(refs, k)
	}
	return refs
}

func (rl *RefList) GetReferencedJobs() []string {
	refs := make([]string, 0)
	for k := range rl.referenced_jobs {
		refs = append(refs, k)
	}
	return refs
}

func (rl *RefList) GetIncluded() []string {
	refs := make([]string, 0)
	for k := range rl.included {
		refs = append(refs, k)
	}
	return refs
}

func (rl *RefList) findRefs(state *OTree) {
	for _, blockExpr := range state.GetBranch("state").Keys() {
		expr := blockExpr.(string)
		if strings.Contains(expr, "~") || strings.Contains(expr, "&") {
			for _, expr_t := range strings.Split(expr, " ") {
				if strings.HasPrefix(expr_t, "~") {
					rl.included[strings.Split(expr_t, "/")[0][1:]] = true
					rl.required_jobs[strings.Split(expr_t, "/")[1][1:]] = true
				} else if strings.HasPrefix(expr_t, "&") {
					rl.included[strings.Split(expr_t, "/")[0][1:]] = true
					rl.referenced_jobs[strings.Split(expr_t, "/")[1][1:]] = true
				}
			}
		}
	}
}

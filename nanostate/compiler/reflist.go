package nanocms_compiler

import (
	"fmt"
	"strings"
)

type RefList struct {
	included        map[string]bool
	referenced_jobs map[string]bool
	required_jobs   map[string]bool // Their content
	visited         []string
}

func NewRefList() *RefList {
	rl := new(RefList)
	rl.Flush()
	return rl
}

// Get all mentioned references
func (rl *RefList) FindRefs(state *OTree) {
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

// MarkVisited marks a reference as "seen" and "requested".
// If it gets marked again, it means the request wasn't completed,
// so we hit a infinite cycle, which needs to be broken out.
func (rl *RefList) MarkStateRequested(id string) string {
	for _, mark := range rl.visited {
		if mark == id {
			panic(fmt.Errorf("State with ID '%s' still wasn't resolved", id))
		}
	}
	rl.visited = append(rl.visited, id)
	return id
}

// MarkResolved marks a reference as "resolved" and removes from the stack
func (rl *RefList) MarkStateResolved(id string) *RefList {
	for idx, mark := range rl.visited {
		if mark == id {
			rl.visited = append(rl.visited[:idx], rl.visited[idx+1:]...)
			break
		}
	}
	delete(rl.included, id)

	return rl
}

// Flush and forget everything
func (rl *RefList) Flush() *RefList {
	rl.included = make(map[string]bool)        // State IDs
	rl.referenced_jobs = make(map[string]bool) // the entire blocks
	rl.required_jobs = make(map[string]bool)   // their content
	rl.visited = make([]string, 0)

	return rl
}

func (rl *RefList) findRefs(state *OTree) {
	for _, blockExpr := range state.GetBranch("state").Keys() {
		expr := blockExpr.(string)
		if strings.Contains(expr, "~") || strings.Contains(expr, "&") {
			for _, expr_t := range strings.Split(expr, " ") {
				if strings.HasPrefix(expr_t, "~") {
					rl.included[strings.Split(expr_t, "/")[0][1:]] = true
					rl.required_jobs[strings.Split(expr_t, "/")[1]] = true
				} else if strings.HasPrefix(expr_t, "&") {
					rl.included[strings.Split(expr_t, "/")[0][1:]] = true
					rl.referenced_jobs[strings.Split(expr_t, "/")[1]] = true
				}
			}
		}
	}
}

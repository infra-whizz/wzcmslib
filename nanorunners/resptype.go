package nanocms_runners

import (
	"encoding/json"
)

type RunnerStdResult struct {
	Stdout  string
	Stderr  string
	Errmsg  string
	Errcode int
	Json    map[string]interface{}
}

type RunnerHostResult struct {
	Host     string
	Response map[string]RunnerStdResult
}

type RunnerResponseModule struct {
	Module   string
	Errcode  int
	Errmsg   string
	Response []RunnerHostResult
}
type RunnerResponseGroup struct {
	Errcode  int
	Errmsg   string
	Response []RunnerResponseModule
}

type RunnerResponse struct {
	Id          string
	Description string
	Groups      map[string]RunnerResponseGroup
}

// JSON output of the response structure
func (rr *RunnerResponse) JSON() string {
	j, err := json.Marshal(rr)
	if err != nil {
		panic(err)
	}
	return string(j)
}

// JSON output of the response structure
func (rr *RunnerResponse) PrettyJSON() string {
	j, err := json.MarshalIndent(rr, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(j)
}

// Serialise to a map/interface object
func (rr *RunnerResponse) Serialise() map[string]interface{} {
	data := make(map[string]interface{})
	j, err := json.Marshal(rr)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(j, &data); err != nil {
		panic(err)
	}
	return data
}

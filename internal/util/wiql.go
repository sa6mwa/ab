package util

import (
	"encoding/json"
	"fmt"
)

// ParseWIQLIDs extracts work item IDs from az boards query output.
// It supports multiple shapes observed from az:
// 1) {"workItems":[{"id":123}, ...]}
// 2) [{"id":123}, ...]
// 3) {"value":[{"id":123}, ...]}
func ParseWIQLIDs(raw []byte) ([]int, error) {
	// shape 1
	var s1 struct {
		WorkItems []struct {
			ID int `json:"id"`
		} `json:"workItems"`
	}
	if err := json.Unmarshal(raw, &s1); err == nil && len(s1.WorkItems) > 0 {
		out := make([]int, 0, len(s1.WorkItems))
		for _, w := range s1.WorkItems {
			out = append(out, w.ID)
		}
		return out, nil
	}
	// shape 2
	var s2 []struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(raw, &s2); err == nil && len(s2) > 0 {
		out := make([]int, 0, len(s2))
		for _, w := range s2 {
			out = append(out, w.ID)
		}
		return out, nil
	}
	// shape 3
	var s3 struct {
		Value []struct {
			ID int `json:"id"`
		} `json:"value"`
	}
	if err := json.Unmarshal(raw, &s3); err == nil && len(s3.Value) > 0 {
		out := make([]int, 0, len(s3.Value))
		for _, w := range s3.Value {
			out = append(out, w.ID)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unrecognized WIQL JSON shape; cannot find IDs")
}

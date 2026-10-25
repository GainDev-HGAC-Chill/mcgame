// Open-sourced by BaoLT

// Requirement ID helpers keep state-reader ownership declarations explicit and deterministic.
package achievement

func requirementIDRange(first, last int64) []int64 {
	if last < first {
		return nil
	}
	ids := make([]int64, 0, last-first+1)
	for requirementID := first; requirementID <= last; requirementID++ {
		ids = append(ids, requirementID)
	}
	return ids
}

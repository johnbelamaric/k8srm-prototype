package main

// This prototype demonstrates allocating capacity from nodes,
// adhering to the claim constraints and requests.
// Currently, allocations are for a pod, and on a single node. However,
// the general framework should be extensible across multi-pod workloads and
// multi-node capacity.

type NodeCapacityAllocation struct {
	NodeName       string                   `json:"nodeName"`
	Allocations    []PoolCapacityAllocation `json:"allocations,omitempty"`
	FailureSummary string                   `json:"failureSummary,omitempty"`
	FailureDetails []string                 `json:"failureDetails,omitempty"`
}

type PoolCapacityAllocation struct {
	Driver         string               `json:"driver"`
	PoolName       string               `json:"poolName"`
	ResourceName   string               `json:"resourceName"`
	Allocations    []CapacityAllocation `json:"allocations"`
	Score          int                  `json:"score"`
	FailureSummary string               `json:"failureSummary,omitempty"`
	FailureDetails []string             `json:"failureDetails,omitempty"`
}

type CapacityAllocation struct {
	CapacityRequest `json:",inline"`

	// Topologies contains the topology assignments of the request allocation. Note
	// that exactly one of each topology type from the original Capacity must be in
	// this list. It is possible for the same requested capacity type, we split the
	// request across multiple topologies. This is the case, for example, if a
	// single memory request cannot be satisfied by a single NUMA node.
	Topologies []TopologyAssignment `json:"topologies,omitempty"`
}

type TopologyAssignment struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

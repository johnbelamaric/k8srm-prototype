package main

// This prototype demonstrates allocating capacity from nodes,
// adhering to the claim constraints and requests.
// Currently, allocations are for a pod, and on a single node. However,
// the general framework should be extensible across multi-pod workloads and
// multi-node capacity.

// NodeCapacityAllocation contains the results of an attempt to satisfy a
// set of CapacityClaims (e.g., for a pod) against a node.
type NodeCapacityAllocation struct {
	NodeName                 string                    `json:"nodeName"`
	CapacityClaimAllocations []CapacityClaimAllocation `json:"capacityClaimAllocations"`
}

// CapacityClaimAlloction contains the results of an attempt to satisfy a
// CapacityClaim against a collection of pools (typically a node)
type CapacityClaimAllocation struct {
	ClaimName                string                    `json:"claimName"`
	ResourceClaimAllocations []ResourceClaimAllocation `json:"resourceClaimAllocations,omitempty"`
}

// ResourceClaimAllocation contains the results of an attempt to sastify a
// ResourceClaim against a collection of pools (typically a node)
type ResourceClaimAllocation struct {
	ClaimName       string                   `json:"claimName"`
	PoolAllocations []PoolCapacityAllocation `json:"poolAllocations"`
	FailureSummary  string                   `json:"failureSummary,omitempty"`
	FailureDetails  []string                 `json:"failureDetails,omitempty"`
}

// PoolCapacityAllocation contains the results of an attempt to sastisfy a
// resource claim against a specific resource pool.
type PoolCapacityAllocation struct {
	PoolName            string               `json:"poolName"`
	ResourceName        string               `json:"resourceName"`
	CapacityAllocations []CapacityAllocation `json:"capacityAllocations"`
	Score               int                  `json:"score"`
	FailureSummary      string               `json:"failureSummary,omitempty"`
	FailureDetails      []string             `json:"failureDetails,omitempty"`
}

// CapacityAllocation is a specific set of capacity allocations and their
// topology assigments.
type CapacityAllocation struct {
	CapacityRequest `json:",inline"`

	// Topologies contains the topology assignments of the request allocation. Note
	// that exactly one of each topology type from the original Capacity must be in
	// this list. It is possible for the same requested capacity type, we split the
	// request across multiple topologies. This is the case, for example, if a
	// single memory request cannot be satisfied by a single NUMA node.
	Topologies []TopologyAssignment `json:"topologies,omitempty"`
}

// TopologyAssignment contains the specific topology from which a capacity is drawn.
type TopologyAssignment struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

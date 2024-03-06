package main

type NodeCapacityAllocation struct {
	NodeName    string               `json:"nodeName"`
	Allocations []CapacityAllocation `json:"allocations"`
}

type CapacityAllocation struct {
	Driver     string               `json:"driver"`
	Capacities []CapacityRequest    `json:"capacities"`
	Topologies []TopologyAssignment `json:"topologies,omitempty"`
}

// if there is no topology constraint in the request, and
// the topology is aggregatable, we do not need to assign
// a specific topology
// But if we don't, then we can't fulfill requests with topology constraints
// at least until the actuation engine that *does* do topology assignments
// kicks in.
type TopologyAssignment struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func Schedule(nrs []NodeResources, cc *CapacityClaim) *NodeCapacityAllocation {
	return nil
}

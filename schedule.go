package main

type NodeCapacityAllocation struct {
	NodeName    string               `json:"nodeName"`
	Allocations []CapacityAllocation `json:"allocations"`
}

type CapacityAllocation struct {
	Driver     string               `json:"driver"`
	Capacity   CapacityRequest      `json:"capacity"`
	Topologies []TopologyAssignment `json:"topologies,omitempty"`
}

type TopologyAssignment struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func Schedule(nrs []NodeResources, cc *CapacityClaim) *NodeCapacityAllocation {
	return nil
}

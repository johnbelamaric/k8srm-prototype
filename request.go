package main

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

type CapacityClaim struct {
	Core       ResourceClaim        `json:"core"`
	Extended   []ResourceClaim      `json:"extended, omitempty"`
	Topologies []TopologyConstraint `json:"topologies,omitempty"`
}

type ResourceClaim struct {
	Driver      string               `json:"driver"`
	Constraints string               `json:"constraints,omitempty"`
	Topologies  []TopologyConstraint `json:"topologies,omitempty"`
	Capacities  []CapacityRequest    `json:"capacities,omitempty"`
}

type TopologyConstraint struct {
	Type string `json:"type"`
}

type CapacityRequest struct {
	Capacity string `json:"capacity"`

	// one of
	Counter  *ResourceCounterRequest  `json:"counter,omitempty"`
	Quantity *ResourceQuantityRequest `json:"quantity,omitempty"`
}

type ResourceCounterRequest struct {
	Request int64 `json:"request"`
}

type ResourceQuantityRequest struct {
	Request resource.Quantity `json:"request"`
}

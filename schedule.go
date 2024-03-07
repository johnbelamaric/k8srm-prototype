package main

import (
	"fmt"
	"strings"
)

type NodeCapacityAllocation struct {
	NodeName      string               `json:"nodeName"`
	Allocations   []CapacityAllocation `json:"allocations"`
	FailureReason string               `json:"failureReason"`
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

// Available calculates the available capacity after applying an allocation
func Available(capacity []NodeResources, allocation *NodeCapacityAllocation) []NodeResources {
	panic("not implemented")
	return capacity
}

// Allocate evaluates if a node can fit a claim, and if so, returns
// the allocation (including topology assignments). If not, returns
// the first reason why

func Allocate(nr NodeResources, cc *CapacityClaim) NodeCapacityAllocation {
	result := NodeCapacityAllocation{NodeName: nr.Name}

	// don't really treat core differently
	claims := []ResourceClaim{cc.Core}
	claims = append(claims, cc.Extended...)

	// index our pools
	pools := make(map[string]ResourcePool)
	pools[nr.Core.Driver] = nr.Core
	for _, p := range nr.Extended {
		pools[p.Driver] = p
	}

	// check if each claim can be satisfied
	for _, c := range claims {
		ca := CapacityAllocation{Driver: c.Driver}

		// find the pool corresponding to this claim
		pool, ok := pools[c.Driver]
		if !ok || len(pool.Resources) == 0 {
			// this node cannot satisfy this claim, as
			// it has no resources for this driver
			result.FailureReason = fmt.Sprintf("no resources for driver %q", c.Driver)
			return result
		}

		// filter out resources that do not meet the constraints
		var resources []Resource
		for _, r := range pool.Resources {
			pass, err := r.MeetsConstraints(c.Constraints, pool.Attributes)
			if err != nil {
				result.FailureReason = fmt.Sprintf("error evaluating driver %q resource %q against constraints %v: %s",
					c.Driver, r.Name, c.Constraints, err)
				return result
			}
			if pass {
				resources = append(resources, r)
			}
		}

		if len(resources) == 0 {
			result.FailureReason = fmt.Sprintf("no driver %q resources meet the constraints %v", c.Driver, c.Constraints)
			return result
		}

		// find the first resource that can satisfy the claim
		var failures []string
		for _, r := range resources {
			capacities, reason := r.AllocateCapacity(c)
			if len(capacities) == 0 && reason == "" {
				reason = "unknown"
			}

			if reason != "" {
				failures = append(failures, fmt.Sprintf("  - %s: %s", r.Name, reason))
				continue
			}

			ca.Capacities = capacities
			break
		}
		if len(ca.Capacities) == 0 {
			result.FailureReason = fmt.Sprintf("no resource with sufficient capacity for driver %q:\n%s", c.Driver, strings.Join(failures, "\n"))
			return result
		}
		result.Allocations = append(result.Allocations, ca)
	}
	return result
}

// Schedule finds the first available node that can accomodate the claim
func Schedule(available []NodeResources, cc *CapacityClaim) *NodeCapacityAllocation {
	var failures []string
	for _, nr := range available {
		allocation := Allocate(nr, cc)
		if allocation.FailureReason == "" && len(allocation.Allocations) > 0 {
			return &allocation
		}
		failures = append(failures, allocation.NodeName+": "+allocation.FailureReason)
	}
	fmt.Printf("Could not schedule:\n")
	for _, f := range failures {
		fmt.Printf("- %s\n", f)
	}
	return nil
}

func (r *Resource) AllocateCapacity(rc ResourceClaim) ([]CapacityRequest, string) {
	/* Not ready to consider topology yet
	*
	// see what topology constraints we need to consider
	// here, we combine the topology constraints from the capacity claim (which
	// apply to all resources), as well as the constraint for this particular claim
	topoConstraints := make(map[string]bool)
	for _, t := range cc.Topologies {
		topoConstraints[t.Type] = true
	}
	for _, t := range c.Topologies {
		topoConstraints[t.Type] = true
	}

	// flatten capacities when they are aggregatable across
	// topologies
	var flat []Capacity
	for _, r := range pool.Resources {
		for _, capacity := range r.Capacities {
			flat = append(flat, capacity)
		}
	}
	*/

	var result []CapacityRequest
	// index the capacities in the resource
	capacityMap := make(map[string]Capacity)
	for _, c := range r.Capacities {
		capacityMap[c.Name] = c
	}

	// evaluate each claim capacity and see if we can satisfy it
	for _, cr := range rc.Capacities {
		availCap, ok := capacityMap[cr.Capacity]
		if !ok {
			return nil, fmt.Sprintf("no capacity %q present in resource %q", cr.Capacity, r.Name)
		}
		allocReq, err := availCap.AllocateRequest(cr)
		if err != nil {
			return nil, fmt.Sprintf("error evaluating capacity %q in resource %q: %s", cr.Capacity, r.Name, err)
		}
		if allocReq == nil {
			return nil, fmt.Sprintf("insufficient capacity %q present in resource %q", cr.Capacity, r.Name)
		}
		result = append(result, *allocReq)
	}

	return result, ""
}

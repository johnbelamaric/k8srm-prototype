package main

import (
	"fmt"
	"strings"
)

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

func (nca *NodeCapacityAllocation) Success() bool {
	return nca.FailureSummary == "" && len(nca.FailureDetails) == 0
}

func (nca *NodeCapacityAllocation) FailureReason() string {
	if nca.Success() {
		return ""
	}

	if nca.FailureSummary != "" {
		return nca.FailureSummary
	}

	return fmt.Sprintf("could not allocate capacity from any of %d pools", len(nca.FailureDetails))
}

func (nca *NodeCapacityAllocation) Score() int {
	if !nca.Success() {
		return 0
	}

	score := 0
	for _, pca := range nca.Allocations {
		score += pca.Score
	}

	return score
}

type PoolCapacityAllocation struct {
	Driver         string               `json:"driver"`
	ResourceName   string               `json:"resourceName"`
	Capacities     []CapacityRequest    `json:"capacities"`
	Topologies     []TopologyAssignment `json:"topologies,omitempty"`
	Score          int                  `json:"score"`
	FailureSummary string               `json:"failureSummary,omitempty"`
	FailureDetails []string             `json:"failureDetails,omitempty"`
}

func (pca *PoolCapacityAllocation) Success() bool {
	return pca.FailureSummary == "" && len(pca.FailureDetails) == 0 && len(pca.Capacities) > 0
}

func (pca *PoolCapacityAllocation) FailureReason() string {
	if pca.Success() {
		return ""
	}

	if pca.FailureSummary != "" {
		return pca.FailureSummary
	}

	return fmt.Sprintf("could not allocate capacity from any of %d resources", len(pca.FailureDetails))

}

// TODO:(johnbelamaric) open question
// if there is no topology constraint in the request, and
// the topology is aggregatable, we do not need to assign
// a specific topology
// But if we don't, then we can't fulfill other requests with topology constraints
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

// AllocateForPod evaluates if a node can fit a pod claim, and if so, returns
// the allocation (including topology assignments) and a score.
// If not, returns the reason why the allocation is impossible.

func (nr *NodeResources) AllocateForPod(cc *PodCapacityClaim) NodeCapacityAllocation {
	result := NodeCapacityAllocation{NodeName: nr.Name}

	// for now, don't really treat core differently
	// but we will have to when we incorporate topology
	var claims []ResourceClaim
	claims = append(claims, cc.PodClaim.Claims...)
	for _, contClaim := range cc.ContainerClaims {
		claims = append(claims, contClaim.Claims...)
	}

	// find the best pool to satisfy each claim
	// TODO(johnbelamaric): fix this so the as each claim is sastisfied,
	// we reduce the pool capacity. Right now if there are multiple claims
	// for the same pool, we could double-allocate
	for _, c := range claims {
		var poolResults []*PoolCapacityAllocation
		var best *PoolCapacityAllocation

		// find the best pool that can satisfy the claim
		for _, pool := range nr.Pools {
			poolResult := pool.AllocateCapacity(c)
			poolResults = append(poolResults, &poolResult)
			if !poolResult.Success() {
				continue
			}
			if best == nil || best.Score < poolResult.Score {
				best = &poolResult
			}
		}
		if best == nil {
			result.FailureSummary = fmt.Sprintf("claim driver %q: no resource with sufficient capacity in any pool", c.Driver)
			for _, pca := range poolResults {
				result.FailureDetails = append(result.FailureDetails, pca.FailureReason())
			}
			// TODO(johnbelamaric): restructure to try every claim even if one fails
			return result
		}
		result.Allocations = append(result.Allocations, *best)
	}
	return result
}

// AllocateCapacity will evaluate a resource claim against the pool, and
// return the options for making those allocations against the pools resources.
func (pool *ResourcePool) AllocateCapacity(rc ResourceClaim) PoolCapacityAllocation {
	result := PoolCapacityAllocation{Driver: pool.Driver}

	if rc.Driver != "" && rc.Driver != pool.Driver {
		result.FailureSummary = fmt.Sprintf("pool %q: driver mismatch", pool.Name)
		return result
	}

	var failures []string

	// filter out resources that do not meet the constraints
	var resources []Resource
	for _, r := range pool.Resources {
		pass, err := r.MeetsConstraints(rc.Constraints, pool.Attributes)
		if err != nil {
			result.FailureSummary = fmt.Sprintf("pool %q: error evaluating resource %q against constraints: %s",
				pool.Name, r.Name, err)
			return result
		}
		if !pass {
			failures = append(failures, fmt.Sprintf("%s: does not meet constraints", r.Name))
			continue
		}

		resources = append(resources, r)
	}

	if len(resources) == 0 {
		result.FailureSummary = fmt.Sprintf("pool %q: no resources meet the constraints %v", pool.Name, rc.Constraints)
		return result
	}

	// find the first resource that can satisfy the claim
	for _, r := range resources {
		capacities, reason := r.AllocateCapacity(rc)
		if len(capacities) == 0 && reason == "" {
			reason = "unknown"
		}

		if reason != "" {
			failures = append(failures, fmt.Sprintf("%s: %s", r.Name, reason))
			continue
		}

		//TODO(johnbelamaric): loop through all instead of using first, add scoring
		result.Score = 1
		result.Capacities = capacities
		break
	}

	if len(result.Capacities) == 0 {
		result.FailureSummary = fmt.Sprintf("pool %q: no resources with sufficient capacity", pool.Name)
		result.FailureDetails = failures
		return result
	}

	return result
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

// SchedulePod finds the best available node that can accomodate the pod claim
func SchedulePod(available []NodeResources, cc *PodCapacityClaim) *NodeCapacityAllocation {
	var results []*NodeCapacityAllocation
	var best *NodeCapacityAllocation
	for _, nr := range available {
		nca := nr.AllocateForPod(cc)
		results = append(results, &nca)
		if !nca.Success() {
			continue
		}
		if best == nil || best.Score() < nca.Score() {
			best = &nca
		}
	}

	if best != nil {
		return best
	}

	fmt.Printf("Could not schedule:\n")
	for _, nca := range results {
		fmt.Printf("%s: %s\n", nca.NodeName, nca.FailureReason())
		if len(nca.FailureDetails) > 0 {
			fmt.Printf(" - %s\n", strings.Join(nca.FailureDetails, "\n - "))
		}
	}
	return nil
}

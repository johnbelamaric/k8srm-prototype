package main

import (
	"fmt"
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/big"
	"strings"
)

// This file contains all the functions for scheduling.

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

// NodeResources methods

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

// ResourcePool methods

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
		result.Allocations = capacities
		break
	}

	if len(result.Allocations) == 0 {
		result.FailureSummary = fmt.Sprintf("pool %q: no resources with sufficient capacity", pool.Name)
		result.FailureDetails = failures
		return result
	}

	return result
}

// Resource methods

func (r *Resource) AllocateCapacity(rc ResourceClaim) ([]CapacityAllocation, string) {
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
		for _, capacity := range r.Allocations {
			flat = append(flat, capacity)
		}
	}
	*/

	var result []CapacityAllocation
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

// Capacity methods

func (c Capacity) AllocateRequest(cr CapacityRequest) (*CapacityAllocation, error) {
	if c.Counter != nil && cr.Counter != nil {
		if cr.Counter.Request <= c.Counter.Capacity {
			return &CapacityAllocation{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Counter:  &ResourceCounterRequest{cr.Counter.Request},
				},
			}, nil
		}
		return nil, nil
	}

	if c.Quantity != nil && cr.Quantity != nil {
		if cr.Quantity.Request.Cmp(c.Quantity.Capacity) <= 0 {
			return &CapacityAllocation{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Quantity: &ResourceQuantityRequest{cr.Quantity.Request},
				},
			}, nil
		}
		return nil, nil
	}

	if c.Block != nil && cr.Quantity != nil {
		realRequest := roundToBlock(cr.Quantity.Request, c.Block.Size)
		if realRequest.Cmp(c.Block.Capacity) <= 0 {
			return &CapacityAllocation{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Quantity: &ResourceQuantityRequest{realRequest},
				},
			}, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("invalid allocation request of %v from %v", cr, c)
}

func roundToBlock(q, size resource.Quantity) resource.Quantity {
	qi := qtoi(q)
	si := qtoi(size)
	zero := big.NewInt(0)
	remainder := big.NewInt(0)
	remainder.Rem(qi, si)
	if remainder.Cmp(zero) > 0 {
		qi.Add(qi, si).Sub(qi, remainder)
	}
	// canonicalize and return
	return resource.MustParse(resource.NewDecimalQuantity(*inf.NewDecBig(qi, inf.Scale(-1*resource.Nano)), q.Format).String())
}

// force to nano scale and return as int
func qtoi(q resource.Quantity) *big.Int {
	_, scale := q.AsCanonicalBytes(nil)
	d := q.AsDec()
	d.SetScale(inf.Scale(int32(resource.Nano) - scale))
	i := big.NewInt(0)
	i.SetString(d.String(), 10)
	return i
}

// NodeCapacityAllocation methods

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

// PoolCapacityAlloction methods

func (pca *PoolCapacityAllocation) Success() bool {
	return pca.FailureSummary == "" && len(pca.FailureDetails) == 0 && len(pca.Allocations) > 0
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

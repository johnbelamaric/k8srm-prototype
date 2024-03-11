package main

import (
	"fmt"
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/big"
	"sigs.k8s.io/yaml"
)

// This file contains all the functions for scheduling.

// SchedulePod finds the best available node that can accomodate the pod claim
func SchedulePod(available []NodeResources, pcc *PodCapacityClaim) *NodeCapacityAllocation {
	var best *NodeCapacityAllocation
	for _, nr := range available {
		nca := nr.AllocatePodCapacityClaim(pcc)

		fmt.Printf("%s: %d\n", nca.NodeName, nca.Score())

		if !nca.Success() {
			var unsatisfied []CapacityClaimAllocation
			for _, cca := range nca.CapacityClaimAllocations {
				if cca.Success() {
					continue
				}
				unsatisfied = append(unsatisfied, cca)
			}

			b, _ := yaml.Marshal(unsatisfied)
			fmt.Println(string(b))
			fmt.Println("---")

			continue
		}
		if best == nil || best.Score() < nca.Score() {
			best = &nca
		}
	}

	if best != nil {
		return best
	}

	return nil
}

// NodeResources methods

// AllocateForPod evaluates if a node can fit a pod claim, and if so, returns
// the allocation (including topology assignments) and a score.
// If not, returns the reason why the allocation is impossible.
func (nr *NodeResources) AllocatePodCapacityClaim(pcc *PodCapacityClaim) NodeCapacityAllocation {
	result := NodeCapacityAllocation{NodeName: nr.Name}

	result.CapacityClaimAllocations = append(result.CapacityClaimAllocations, nr.AllocateCapacityClaim(&pcc.PodClaim))

	for _, cc := range pcc.ContainerClaims {
		result.CapacityClaimAllocations = append(result.CapacityClaimAllocations, nr.AllocateCapacityClaim(&cc))
	}

	return result
}

func (nr *NodeResources) AllocateCapacityClaim(cc *CapacityClaim) CapacityClaimAllocation {
	result := CapacityClaimAllocation{ClaimName: cc.Name}

	for _, rc := range cc.Claims {
		rca := ResourceClaimAllocation{ClaimName: cc.Name}

		// find the best pool to satisfy each resource claim
		// TODO(johnbelamaric): allows splitting a single resource claim across multiple
		// pools
		// TODO(johnbelamaric): fix this so the as each claim is sastisfied,
		// we reduce the pool capacity. Right now if there are multiple claims
		// for the same pool, we could double-allocate
		var poolResults []*PoolCapacityAllocation
		var best *PoolCapacityAllocation

		// find the best pool that can satisfy the claim
		for _, pool := range nr.Pools {
			poolResult := pool.AllocateCapacity(rc)
			poolResults = append(poolResults, &poolResult)
			if !poolResult.Success() {
				continue
			}
			if best == nil || best.Score < poolResult.Score {
				best = &poolResult
			}
		}
		if best != nil {
			rca.PoolAllocations = append(rca.PoolAllocations, *best)
		} else {
			rca.FailureSummary = "no resource with sufficient capacity in any pool"
			for _, pca := range poolResults {
				rca.FailureDetails = append(rca.FailureDetails, pca.FailureReason())
			}
		}
		result.ResourceClaimAllocations = append(result.ResourceClaimAllocations, rca)
	}
	return result
}

// ResourcePool methods

// AllocateCapacity will evaluate a resource claim against the pool, and
// return the options for making those allocations against the pools resources.
func (pool *ResourcePool) AllocateCapacity(rc ResourceClaim) PoolCapacityAllocation {
	result := PoolCapacityAllocation{PoolName: pool.Name}

	if rc.Driver != "" && rc.Driver != pool.Driver {
		result.FailureSummary = "driver mismatch"
		return result
	}

	var failures []string

	// filter out resources that do not meet the constraints
	var resources []Resource
	for _, r := range pool.Resources {
		pass, err := r.MeetsConstraints(rc.Constraints, pool.Attributes)
		if err != nil {
			result.FailureSummary = fmt.Sprintf("error evaluating resource %q against constraints: %s", r.Name, err)
			return result
		}
		if !pass {
			failures = append(failures, fmt.Sprintf("%s: does not meet constraints", r.Name))
			continue
		}

		resources = append(resources, r)
	}

	if len(resources) == 0 {
		result.FailureSummary = fmt.Sprintf("no resources meet the constraints %v", rc.Constraints)
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

		//TODO(johnbelamaric): loop through all instead of using first, add scoring and splitting
		// across resources if possible
		result.Score = 1
		result.CapacityAllocations = capacities
		result.ResourceName = r.Name
		break
	}

	if len(result.CapacityAllocations) == 0 {
		result.FailureSummary = "no resources with sufficient capacity"
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
	for _, a := range nca.CapacityClaimAllocations {
		if !a.Success() {
			return false
		}
	}

	return true
}

func (nca *NodeCapacityAllocation) Score() int {
	if !nca.Success() {
		return 0
	}

	score := 0
	for _, a := range nca.CapacityClaimAllocations {
		score += a.Score()
	}

	return score
}

// CapacityClaimAllocation methods

func (cca *CapacityClaimAllocation) Success() bool {
	for _, a := range cca.ResourceClaimAllocations {
		if !a.Success() {
			return false
		}
	}

	return true
}

func (cca *CapacityClaimAllocation) Score() int {
	if !cca.Success() {
		return 0
	}

	score := 0
	for _, a := range cca.ResourceClaimAllocations {
		score += a.Score()
	}

	return score
}

// ResourceClaimAllocation methods

func (rca *ResourceClaimAllocation) Success() bool {
	if rca.FailureSummary != "" {
		return false
	}

	if len(rca.PoolAllocations) == 0 {
		return false
	}

	for _, a := range rca.PoolAllocations {
		if !a.Success() {
			return false
		}
	}

	return true
}

func (rca *ResourceClaimAllocation) Score() int {
	if !rca.Success() {
		return 0
	}

	score := 0
	for _, a := range rca.PoolAllocations {
		score += a.Score
	}

	return score
}

// PoolCapacityAlloction methods

func (pca *PoolCapacityAllocation) Success() bool {
	return pca.FailureSummary == "" && len(pca.FailureDetails) == 0 && len(pca.CapacityAllocations) > 0
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

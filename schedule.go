package main

import (
	"fmt"
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/big"
	"sort"
	"strings"
)

// This file contains all the functions for scheduling.

// SchedulePod finds the best available node that can accomodate the pod claim
// Note that for the prototype, no allocation state is kept across calls to this function,
// but since capacity values are often pointers, you really should start with a fresh
// NodeResources for testing
func SchedulePod(available []NodeResources, pcc PodCapacityClaim) *NodeAllocationResult {
	results, best := EvaluateNodesForPod(available, pcc)
	if best < 0 {
		return nil
	}

	return &results[best]
}

func EvaluateNodesForPod(available []NodeResources, pcc PodCapacityClaim) ([]NodeAllocationResult, int) {
	best := -1
	var results []NodeAllocationResult
	for i, nr := range available {
		results = append(results, nr.AllocatePodCapacityClaim(pcc))

		if !results[i].Success() {
			continue
		}
		if best < 0 || results[best].Score() < results[i].Score() {
			best = i
		}
	}

	return results, best
}

// NodeResources methods

// AllocatePodCapacityClaim evaluates the capacity claims for a pod.
func (nr *NodeResources) AllocatePodCapacityClaim(pcc PodCapacityClaim) NodeAllocationResult {
	result := NodeAllocationResult{NodeName: nr.Name}

	result.CapacityClaimResults = append(result.CapacityClaimResults, nr.AllocateCapacityClaim(&pcc.PodClaim))

	for _, cc := range pcc.ContainerClaims {
		result.CapacityClaimResults = append(result.CapacityClaimResults, nr.AllocateCapacityClaim(&cc))
	}

	return result
}

func (nr *NodeResources) AllocateCapacityClaim(cc *CapacityClaim) CapacityClaimResult {
	ccResult := CapacityClaimResult{ClaimName: cc.Name}

	for _, rc := range cc.Claims {
		rcResult := ResourceClaimResult{ClaimName: rc.Name}

		best := -1
		for i, pool := range nr.Pools {
			rcResult.PoolResults = append(rcResult.PoolResults, pool.AllocateCapacity(rc))
			if !rcResult.PoolResults[i].Success() {
				continue
			}
			if best < 0 || rcResult.PoolResults[best].Score() < rcResult.PoolResults[i].Score() {
				best = i
			}
		}

		rcResult.Best = best

		if best < 0 {
			rcResult.FailureReason = "no pool found that can satisfy the claim"
		} else {
			err := nr.Pools[best].ReduceCapacity(rcResult.PoolResults[best])
			if err != nil {
				rcResult.FailureReason = fmt.Sprintf("error trying to reduce pool capacity: %s", err)
			}
		}

		ccResult.ResourceClaimResults = append(ccResult.ResourceClaimResults, rcResult)
	}
	return ccResult
}

// ResourcePool methods

// AllocateCapacity will evaluate a resource claim against the pool, and
// return the options for making those allocations against the pools resources.
func (pool *ResourcePool) AllocateCapacity(rc ResourceClaim) PoolResult {
	result := PoolResult{PoolName: pool.Name, Best: -1}

	if rc.Driver != "" && rc.Driver != pool.Driver {
		result.FailureReason = fmt.Sprintf("pool driver %q mismatch claim driver %q", pool.Driver, rc.Driver)
		return result
	}

	best := -1
	// filter out resources that do not meet the constraints
	for i, r := range pool.Resources {
		rResult := ResourceResult{ResourceName: r.Name}
		pass, err := r.MeetsConstraints(rc.Constraints, pool.Attributes)
		if err != nil {
			rResult.FailureReason = fmt.Sprintf("error evaluating against constraints: %s", err)
			result.ResourceResults = append(result.ResourceResults, rResult)
			continue
		}
		if !pass {
			rResult.FailureReason = "does not meet constraints"
			result.ResourceResults = append(result.ResourceResults, rResult)
			continue
		}

		capacities, reason := r.AllocateCapacity(rc)
		if len(capacities) == 0 && reason == "" {
			reason = "unknown"
		}

		if reason != "" {
			rResult.FailureReason = reason
			result.ResourceResults = append(result.ResourceResults, rResult)
			continue
		}

		//TODO(johnbelamaric): add scoring
		rResult.Score = 100
		rResult.CapacityResults = capacities
		result.ResourceResults = append(result.ResourceResults, rResult)

		if best < 0 || result.ResourceResults[best].Score < rResult.Score {
			best = i
		}
	}

	result.Best = best

	if best < 0 {
		result.FailureReason = "no resources in pool with sufficient capacity"
	}

	return result
}

func (pool *ResourcePool) ReduceCapacity(pr PoolResult) error {
	if pool.Name != pr.PoolName {
		return fmt.Errorf("cannot reduce pool %q capacity using allocation from pool %q", pool.Name, pr.PoolName)
	}

	if pr.Best < 0 {
		return fmt.Errorf("cannot reduce pool %q capacity from unsatisfied result", pool.Name)
	}

	if len(pool.Resources) != len(pr.ResourceResults) {
		return fmt.Errorf("pool %q resources and resource result list differ in length", pool.Name)
	}

	return pool.Resources[pr.Best].ReduceCapacity(pr.ResourceResults[pr.Best].CapacityResults)
}

// Resource methods

// ReduceCapacity deducts the allocation from the resource so that subsequent
// requests take already allocated capacities into account. This is not how we
// would do it in the real model, because we want drivers to publish capacity without
// tracking allocations. But it's convenient in the prototype.
func (r *Resource) ReduceCapacity(allocations []CapacityResult) error {
	// Capacity allocations should contain enough information to do this

	// index our capacities by their unique topologies
	capMap := make(map[string]int)
	for i, capacity := range r.Capacities {
		capMap[capacity.capKey()] = i
	}

	for _, ca := range allocations {
		idx, ok := capMap[ca.capKey()]
		if !ok {
			return fmt.Errorf("allocated capacity %q not found in resource capacities", ca.capKey())
		}
		var err error
		r.Capacities[idx], err = r.Capacities[idx].reduce(ca.CapacityRequest)
		if err != nil {
			return err
		}
	}

	return nil
}

func (ca *CapacityResult) capKey() string {
	var keyList, topoList []string
	for _, ta := range ca.Topologies {
		topoList = append(topoList, fmt.Sprintf("%s=%s", ta.Type, ta.Name))
	}
	sort.Strings(topoList)
	keyList = append(keyList, ca.CapacityRequest.Capacity)
	keyList = append(keyList, topoList...)
	return strings.Join(keyList, ";")
}

func (c Capacity) capKey() string {
	topos := make(map[string]string)
	for _, t := range c.Topologies {
		topos[t.Type] = t.Name
	}

	var keyList, topoList []string
	for k, v := range topos {
		topoList = append(topoList, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(topoList)
	keyList = append(keyList, c.Name)
	keyList = append(keyList, topoList...)
	return strings.Join(keyList, ";")
}

func (r *Resource) AllocateCapacity(rc ResourceClaim) ([]CapacityResult, string) {
	var result []CapacityResult
	// index the capacities in the resource
	capacityMap := make(map[string][]Capacity)
	for _, c := range r.Capacities {
		capacityMap[c.Name] = append(capacityMap[c.Name], c)
	}

	// evaluate each claim capacity and see if we can satisfy it
	for _, cr := range rc.Capacities {
		availCap, ok := capacityMap[cr.Capacity]
		if !ok {
			return nil, fmt.Sprintf("no capacity %q present in resource %q", cr.Capacity, r.Name)
		}
		satisfied := false
		// TODO(johnbelamaric): implement splitting across topologies (implement GroupInResource)
		for i, capInTopo := range availCap {
			allocReq, err := capInTopo.AllocateRequest(cr)
			if err != nil {
				return nil, fmt.Sprintf("error evaluating capacity %q in resource %q: %s", cr.Capacity, r.Name, err)
			}
			if allocReq != nil {
				capacityMap[cr.Capacity][i], err = availCap[i].reduce(allocReq.CapacityRequest)
				if err != nil {
					return nil, fmt.Sprintf("err reducing capacity %q in resource %q: %s", cr.Capacity, r.Name, err)
				}
				result = append(result, *allocReq)
				satisfied = true
				break
			}
		}
		if !satisfied {
			return nil, fmt.Sprintf("insufficient capacity %q present in resource %q", cr.Capacity, r.Name)
		}
	}

	return result, ""
}

// Capacity methods

func (c Capacity) AllocateRequest(cr CapacityRequest) (*CapacityResult, error) {
	if c.Counter != nil && cr.Counter != nil {
		if cr.Counter.Request <= c.Counter.Capacity {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Counter:  &ResourceCounterRequest{cr.Counter.Request},
				},
				Topologies: c.topologyAssignments(),
			}, nil
		}
		return nil, nil
	}

	if c.Quantity != nil && cr.Quantity != nil {
		if cr.Quantity.Request.Cmp(c.Quantity.Capacity) <= 0 {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Quantity: &ResourceQuantityRequest{cr.Quantity.Request},
				},
				Topologies: c.topologyAssignments(),
			}, nil
		}
		return nil, nil
	}

	if c.Block != nil && cr.Quantity != nil {
		realRequest := roundToBlock(cr.Quantity.Request, c.Block.Size)
		if realRequest.Cmp(c.Block.Capacity) <= 0 {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Capacity: cr.Capacity,
					Quantity: &ResourceQuantityRequest{realRequest},
				},
				Topologies: c.topologyAssignments(),
			}, nil
		}
		return nil, nil
	}

	return nil, fmt.Errorf("request/capacity type mismatch")
}

func (c Capacity) topologyAssignments() []TopologyAssignment {
	var result []TopologyAssignment
	for _, t := range c.Topologies {
		result = append(result, TopologyAssignment{Type: t.Type, Name: t.Name})
	}

	return result
}

// reduce applies a CapacityRequest and returns a reduced Capacity. Note that
// this assumes the CapacityRequest is one that has been returned by
// AllocateCapacity and therefore does no validation. In particular,
// block sizes will not be honored; that should already have been done
func (c Capacity) reduce(cr CapacityRequest) (Capacity, error) {
	if cr.Capacity != c.Name {
		return Capacity{}, fmt.Errorf("cannot reduce capacity %q using request for %q", c.Name, cr.Capacity)
	}
	result := c
	if c.Counter != nil && cr.Counter != nil {
		result.Counter.Capacity -= cr.Counter.Request
		return result, nil
	}

	if c.Quantity != nil && cr.Quantity != nil {
		result.Quantity.Capacity.Sub(cr.Quantity.Request)
		// force caching of string value for test ease
		_ = result.Quantity.Capacity.String()
		return result, nil
	}

	if c.Block != nil && cr.Quantity != nil {
		result.Block.Capacity.Sub(cr.Quantity.Request)
		_ = result.Block.Capacity.String()
		return result, nil
	}

	return Capacity{}, fmt.Errorf("request/capacity type mismatch")
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

// NodeAllocationResult methods

func (nar *NodeAllocationResult) Success() bool {
	for _, ccr := range nar.CapacityClaimResults {
		if !ccr.Success() {
			return false
		}
	}

	return true
}

func (nar *NodeAllocationResult) Score() int {
	if !nar.Success() {
		return 0
	}

	score := 0
	for _, ccr := range nar.CapacityClaimResults {
		score += ccr.Score()
	}

	return score / len(nar.CapacityClaimResults)
}

func (nar *NodeAllocationResult) PrintSummary() {
	msg := "failed"
	if nar.Success() {
		msg = "succeeded"
	}

	fmt.Printf("node %q (%d): %s\n", nar.NodeName, nar.Score(), msg)

	for _, ccr := range nar.CapacityClaimResults {
		msg = "failed"
		if ccr.Success() {
			msg = "succeeded"
		}
		fmt.Printf("- capacity claim %q (%d): %s\n", ccr.ClaimName, ccr.Score(), msg)

		for _, rcr := range ccr.ResourceClaimResults {
			msg = rcr.FailureReason
			if rcr.Success() {
				msg = "succeeded"
			}
			fmt.Printf("  - resource claim %q (%d): %s\n", rcr.ClaimName, rcr.Score(), msg)

			for pri, pr := range rcr.PoolResults {
				msg = pr.FailureReason
				if pr.Success() {
					msg = "succeeded"
				}
				if pri == rcr.Best {
					msg = "best"
				}
				fmt.Printf("    - pool %q (%d): %s\n", pr.PoolName, pr.Score(), msg)
				for rri, rr := range pr.ResourceResults {
					msg = rr.FailureReason
					if rr.Success() {
						msg = "success"
					}
					if rri == pr.Best {
						msg = "best"
					}
					fmt.Printf("      - resource %q (%d): %s\n", rr.ResourceName, rr.Score, msg)
				}
			}
		}
	}
}

// CapacityClaimResult methods

func (ccr *CapacityClaimResult) Success() bool {
	for _, rcr := range ccr.ResourceClaimResults {
		if !rcr.Success() {
			return false
		}
	}

	return true
}

func (ccr *CapacityClaimResult) Score() int {
	if !ccr.Success() {
		return 0
	}

	score := 0
	for _, r := range ccr.ResourceClaimResults {
		score += r.Score()
	}

	return score / len(ccr.ResourceClaimResults)
}

// ResourceClaimResult methods

func (rcr *ResourceClaimResult) Success() bool {
	return rcr.Best >= 0
}

func (rcr *ResourceClaimResult) Score() int {
	if !rcr.Success() {
		return 0
	}

	return rcr.PoolResults[rcr.Best].Score()
}

// PoolResult methods

func (pr *PoolResult) Success() bool {
	return pr.Best >= 0
}

func (pr *PoolResult) Score() int {
	if !pr.Success() {
		return 0
	}

	return pr.ResourceResults[pr.Best].Score
}

// ResourceResult methods

func (rr *ResourceResult) Success() bool {
	return rr.Score > 0
}

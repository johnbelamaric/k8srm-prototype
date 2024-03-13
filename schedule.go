package main

import (
	"fmt"
	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
	"math/big"
	"sigs.k8s.io/yaml"
	"sort"
	"strings"
)

// This file contains all the functions for scheduling.

// SchedulePod finds the best available node that can accomodate the pod claim
// Note that for the prototype, no allocation state is kept across calls to this function,
// but since capacity values are often pointers, you really should start with a fresh
// NodeResources for testing
func SchedulePod(available []NodeResources, pcc PodCapacityClaim) *NodeCapacityAllocation {
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
func (nr *NodeResources) AllocatePodCapacityClaim(pcc PodCapacityClaim) NodeCapacityAllocation {
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
		// pools (implement AggregateInPool)
		var poolResults []*PoolCapacityAllocation
		var best *PoolCapacityAllocation
		var idx int

		// find the best pool that can satisfy the claim
		for i, pool := range nr.Pools {
			poolResult := pool.AllocateCapacity(rc)
			poolResults = append(poolResults, &poolResult)
			if !poolResult.Success() {
				continue
			}
			if best == nil || best.Score < poolResult.Score {
				best = &poolResult
				idx = i
			}
		}
		var err error
		if best != nil {
			err = nr.Pools[idx].ReduceCapacity(best)
			if err == nil {
				rca.PoolAllocations = append(rca.PoolAllocations, *best)
			}

		}

		if err != nil || best == nil {
			rca.FailureSummary = "no resource with sufficient capacity in any pool"
			if err != nil {
				rca.FailureSummary = err.Error()
			}
			for _, pca := range poolResults {
				rca.FailureDetails = append(rca.FailureDetails,
					fmt.Sprintf("%s: %s", pca.PoolName, pca.FailureReason()))
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

func (pool *ResourcePool) ReduceCapacity(pca *PoolCapacityAllocation) error {
	if pool.Name != pca.PoolName {
		return fmt.Errorf("cannot reduce pool %q capacity using allocation from pool %q", pool.Name, pca.PoolName)
	}

	// find the resource
	var r *Resource
	for i, ri := range pool.Resources {
		if ri.Name == pca.ResourceName {
			r = &pool.Resources[i]
			break
		}
	}

	if r == nil {
		return fmt.Errorf("could not find resource %q in pool %q", pca.ResourceName, pool.Name)
	}

	return r.ReduceCapacity(pca.CapacityAllocations)
}

// Resource methods

// ReduceCapacity deducts the allocation from the resource so that subsequent
// requests take already allocated capacities into account. This is not how we
// would do it in the real model, because we want drivers to publish capacity without
// tracking allocations. But it's convenient in the prototype.
func (r *Resource) ReduceCapacity(allocations []CapacityAllocation) error {
	// Capacity allocations should contain enough information to do this

	// index our capacities by their unique topologies
	capMap := make(map[string]int)
	for i, capacity := range r.Capacities {
		capMap[r.capKey(capacity)] = i
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

func (ca *CapacityAllocation) capKey() string {
	var keyList, topoList []string
	for _, ta := range ca.Topologies {
		topoList = append(topoList, fmt.Sprintf("%s=%s", ta.Type, ta.Name))
	}
	sort.Strings(topoList)
	keyList = append(keyList, ca.CapacityRequest.Capacity)
	keyList = append(keyList, topoList...)
	return strings.Join(keyList, ";")
}

func (r *Resource) capKey(capacity Capacity) string {
	topos := make(map[string]string)
	for _, t := range capacity.Topologies {
		topos[t.Type] = t.Name
	}

	var keyList, topoList []string
	for k, v := range topos {
		topoList = append(topoList, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(topoList)
	keyList = append(keyList, capacity.Name)
	keyList = append(keyList, topoList...)
	return strings.Join(keyList, ";")
}

func (r *Resource) AllocateCapacity(rc ResourceClaim) ([]CapacityAllocation, string) {
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
				Topologies: c.topologyAssignments(),
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
				Topologies: c.topologyAssignments(),
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

	return fmt.Sprintf("%s; %s", pca.FailureSummary, strings.Join(pca.FailureDetails, ", "))
}

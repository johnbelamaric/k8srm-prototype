package schedule

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
// NodeDevices for testing
func SchedulePod(available []NodeDevices, pcc PodCapacityClaim) *NodeAllocationResult {
	results, best := EvaluateNodesForPod(available, pcc)
	if best < 0 {
		return nil
	}

	return &results[best]
}

func EvaluateNodesForPod(available []NodeDevices, pcc PodCapacityClaim) ([]NodeAllocationResult, int) {
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

// NodeDevices methods

// AllocatePodCapacityClaim evaluates the capacity claims for a pod.
func (nr *NodeDevices) AllocatePodCapacityClaim(pcc PodCapacityClaim) NodeAllocationResult {
	result := NodeAllocationResult{NodeName: nr.Name}

	result.CapacityClaimResults = append(result.CapacityClaimResults, nr.AllocateCapacityClaim(&pcc.PodClaim))

	for _, cc := range pcc.ContainerClaims {
		result.CapacityClaimResults = append(result.CapacityClaimResults, nr.AllocateCapacityClaim(&cc))
	}

	return result
}

func (nr *NodeDevices) AllocateCapacityClaim(cc *CapacityClaim) CapacityClaimResult {
	ccResult := CapacityClaimResult{ClaimName: cc.Name}

	for _, rc := range cc.Claims {
		rcResult := DeviceClaimResult{ClaimName: rc.Name}

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

		ccResult.DeviceClaimResults = append(ccResult.DeviceClaimResults, rcResult)
	}
	return ccResult
}

// DevicePool methods

// AllocateCapacity will evaluate a device claim against the pool, and
// return the options for making those allocations against the pools devices.
func (pool *DevicePool) AllocateCapacity(rc DeviceClaim) PoolResult {
	result := PoolResult{PoolName: pool.Name, Best: -1}

	if rc.Spec.Driver != nil && *rc.Spec.Driver != "" && *rc.Spec.Driver != pool.Spec.Driver {
		result.FailureReason = fmt.Sprintf("pool driver %q mismatch claim driver %q", pool.Spec.Driver, *rc.Spec.Driver)
		return result
	}

	best := -1
	// filter out devices that do not meet the constraints
	for i, r := range pool.Spec.Devices {
		rResult := DeviceResult{DeviceName: r.Name}
		pass, err := r.MeetsConstraints(rc.Spec.Constraints, pool.Spec.Attributes)
		if err != nil {
			rResult.FailureReason = fmt.Sprintf("error evaluating against constraints: %s", err)
			result.DeviceResults = append(result.DeviceResults, rResult)
			continue
		}
		if !pass {
			rResult.FailureReason = "does not meet constraints"
			result.DeviceResults = append(result.DeviceResults, rResult)
			continue
		}

		capacities, reason := r.AllocateCapacity(rc)
		if len(capacities) == 0 && reason == "" {
			reason = "unknown"
		}

		if reason != "" {
			rResult.FailureReason = reason
			result.DeviceResults = append(result.DeviceResults, rResult)
			continue
		}

		//TODO(johnbelamaric): add scoring
		rResult.Score = 100
		rResult.CapacityResults = capacities
		result.DeviceResults = append(result.DeviceResults, rResult)

		if best < 0 || result.DeviceResults[best].Score < rResult.Score {
			best = i
		}
	}

	result.Best = best

	if best < 0 {
		result.FailureReason = "no devices in pool with sufficient capacity"
	}

	return result
}

func (pool *DevicePool) ReduceCapacity(pr PoolResult) error {
	if pool.Name != pr.PoolName {
		return fmt.Errorf("cannot reduce pool %q capacity using allocation from pool %q", pool.Name, pr.PoolName)
	}

	if pr.Best < 0 {
		return fmt.Errorf("cannot reduce pool %q capacity from unsatisfied result", pool.Name)
	}

	if len(pool.Spec.Devices) != len(pr.DeviceResults) {
		return fmt.Errorf("pool %q devices and device result list differ in length", pool.Name)
	}

	return pool.Spec.Devices[pr.Best].ReduceCapacity(pr.DeviceResults[pr.Best].CapacityResults)
}

// Device methods

// ReduceCapacity deducts the allocation from the device so that subsequent
// requests take already allocated capacities into account. This is not how we
// would do it in the real model, because we want drivers to publish capacity without
// tracking allocations. But it's convenient in the prototype.
func (r *Device) ReduceCapacity(allocations []CapacityResult) error {
	// Capacity allocations should contain enough information to do this

	// index our capacities by their unique topologies
	capMap := make(map[string]int)
	for i, capacity := range r.Resources {
		capMap[capacity.capKey()] = i
	}

	for _, ca := range allocations {
		idx, ok := capMap[ca.capKey()]
		if !ok {
			return fmt.Errorf("allocated capacity %q not found in device capacities", ca.capKey())
		}
		var err error
		r.Resources[idx], err = r.Resources[idx].reduce(ca.CapacityRequest)
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
	keyList = append(keyList, ca.CapacityRequest.Resource)
	keyList = append(keyList, topoList...)
	return strings.Join(keyList, ";")
}

func (c ResourceCapacity) capKey() string {
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

func (r *Device) AllocateCapacity(rc DeviceClaim) ([]CapacityResult, string) {
	var result []CapacityResult
	// index the capacities in the device. this results in an array per
	// capacity name, with the individual per-topology capacities as the
	// entries in the array
	capacityMap := make(map[string][]ResourceCapacity)
	for _, c := range r.Resources {
		capacityMap[c.Name] = append(capacityMap[c.Name], c)
	}

	// evaluate each claim capacity and see if we can satisfy it
	for _, cr := range rc.Spec.Requests {
		availCap, ok := capacityMap[cr.Resource]
		if !ok {
			return nil, fmt.Sprintf("no capacity %q present in device %q", cr.Resource, r.Name)
		}
		satisfied := false
		// TODO(johnbelamaric): currently ignores GroupInDevice value and assumes 'true'
		// TODO(johnbelamaric): splitting across topos should affect score
		unsatReq := cr
		for i, capInTopo := range availCap {
			allocReq, remainReq, err := capInTopo.AllocateRequest(unsatReq)
			if err != nil {
				return nil, fmt.Sprintf("error evaluating capacity %q in device %q: %s", cr.Resource, r.Name, err)
			}
			if allocReq != nil {
				capacityMap[cr.Resource][i], err = availCap[i].reduce(allocReq.CapacityRequest)
				if err != nil {
					return nil, fmt.Sprintf("err reducing capacity %q in device %q: %s", cr.Resource, r.Name, err)
				}
				result = append(result, *allocReq)
			}

			if remainReq == nil {
				satisfied = true
				break
			}

			unsatReq = *remainReq
		}
		if !satisfied {
			return nil, fmt.Sprintf("insufficient capacity %q present in device %q", cr.Resource, r.Name)
		}
	}

	return result, ""
}

// ResourceCapacity methods

func (c ResourceCapacity) AllocateRequest(cr CapacityRequest) (*CapacityResult, *CapacityRequest, error) {
	if c.Counter != nil && cr.Counter != nil {
		if cr.Counter.Request <= c.Counter.Capacity {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Counter:  &ResourceCounterRequest{cr.Counter.Request},
				},
				Topologies: c.topologyAssignments(),
			}, nil, nil
		}
		if c.Counter.Capacity == 0 {
			return nil, &cr, nil
		}
		return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Counter:  &ResourceCounterRequest{c.Counter.Capacity},
				},
				Topologies: c.topologyAssignments(),
			},
			&CapacityRequest{
				Resource: cr.Resource,
				Counter:  &ResourceCounterRequest{cr.Counter.Request - c.Counter.Capacity},
			},
			nil
	}

	if c.Quantity != nil && cr.Quantity != nil {
		if cr.Quantity.Request.Cmp(c.Quantity.Capacity) <= 0 {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Quantity: &ResourceQuantityRequest{cr.Quantity.Request},
				},
				Topologies: c.topologyAssignments(),
			}, nil, nil
		}
		if c.Quantity.Capacity.IsZero() {
			return nil, &cr, nil
		}
		remainder := cr.Quantity.Request
		remainder.Sub(c.Quantity.Capacity)
		return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Quantity: &ResourceQuantityRequest{c.Quantity.Capacity},
				},
				Topologies: c.topologyAssignments(),
			},
			&CapacityRequest{
				Resource: cr.Resource,
				Quantity: &ResourceQuantityRequest{remainder},
			},
			nil
	}

	if c.Block != nil && cr.Quantity != nil {
		realRequest := roundUpToBlock(cr.Quantity.Request, c.Block.Size)
		realCapacity := roundDownToBlock(c.Block.Capacity, c.Block.Size)
		if realRequest.Cmp(realCapacity) <= 0 {
			return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Quantity: &ResourceQuantityRequest{realRequest},
				},
				Topologies: c.topologyAssignments(),
			}, nil, nil
		}
		if c.Block.Capacity.Cmp(c.Block.Size) <= 0 {
			return nil, &cr, nil
		}
		remainder := realRequest
		remainder.Sub(realCapacity)
		return &CapacityResult{
				CapacityRequest: CapacityRequest{
					Resource: cr.Resource,
					Quantity: &ResourceQuantityRequest{realCapacity},
				},
				Topologies: c.topologyAssignments(),
			},
			&CapacityRequest{
				Resource: cr.Resource,
				Quantity: &ResourceQuantityRequest{remainder},
			},
			nil
	}

	if c.AccessMode != nil && cr.AccessMode != nil {
		return c.allocateAccessModeRequest(cr)
	}

	return nil, &cr, fmt.Errorf("request/capacity type mismatch (%v, %v)", c, cr)
}

func (c ResourceCapacity) allocateAccessModeRequest(cr CapacityRequest) (*CapacityResult, *CapacityRequest, error) {

	// upgrade the requested mode based on the capacity's configuration
	requestMode := cr.AccessMode.Request
	if requestMode == ReadOnlyShared && !c.AccessMode.AllowReadOnlyShared {
		requestMode = ReadWriteShared
	}

	if requestMode == ReadWriteShared && !c.AccessMode.AllowReadWriteShared {
		requestMode = WriteExclusive
	}

	if requestMode == WriteExclusive && !c.AccessMode.AllowWriteExclusive {
		requestMode = ReadWriteExclusive
	}

	blockers := 0
	switch requestMode {
	case ReadWriteExclusive:
		blockers += c.AccessMode.ReadOnlyShared
		blockers += c.AccessMode.ReadWriteShared
		blockers += c.AccessMode.WriteExclusive
		blockers += c.AccessMode.ReadWriteExclusive

	case WriteExclusive:
		blockers += c.AccessMode.ReadWriteShared
		blockers += c.AccessMode.WriteExclusive
		blockers += c.AccessMode.ReadWriteExclusive

	case ReadWriteShared:
		blockers += c.AccessMode.WriteExclusive
		blockers += c.AccessMode.ReadWriteExclusive

	case ReadOnlyShared:
		blockers += c.AccessMode.ReadWriteExclusive

	default:
		return nil, &cr, fmt.Errorf("invalid request access mode %q", requestMode)
	}

	if blockers > 0 {
		return nil, &cr, nil
	}

	return &CapacityResult{
		CapacityRequest: CapacityRequest{
			Resource:   cr.Resource,
			AccessMode: &ResourceAccessModeRequest{requestMode},
		},
		Topologies: c.topologyAssignments(),
	}, nil, nil
}

func (c ResourceCapacity) topologyAssignments() []TopologyAssignment {
	var result []TopologyAssignment
	for _, t := range c.Topologies {
		result = append(result, TopologyAssignment{Type: t.Type, Name: t.Name})
	}

	return result
}

// reduce applies a CapacityRequest and returns a reduced ResourceCapacity. Note that
// this assumes the CapacityRequest is one that has been returned by
// AllocateCapacity and therefore does no validation. In particular,
// block sizes will not be honored; that should already have been done
func (c ResourceCapacity) reduce(cr CapacityRequest) (ResourceCapacity, error) {
	if cr.Resource != c.Name {
		return ResourceCapacity{}, fmt.Errorf("cannot reduce capacity %q using request for %q", c.Name, cr.Resource)
	}
	result := c
	if c.Counter != nil && cr.Counter != nil {
		copied := *c.Counter
		result.Counter = &copied
		result.Counter.Capacity -= cr.Counter.Request
		return result, nil
	}

	if c.Quantity != nil && cr.Quantity != nil {
		copied := *c.Quantity
		result.Quantity = &copied
		result.Quantity.Capacity.Sub(cr.Quantity.Request)
		// force caching of string value for test ease
		_ = result.Quantity.Capacity.String()
		return result, nil
	}

	if c.Block != nil && cr.Quantity != nil {
		copied := *c.Block
		result.Block = &copied
		result.Block.Capacity.Sub(cr.Quantity.Request)
		_ = result.Block.Capacity.String()
		return result, nil
	}

	if c.AccessMode != nil && cr.AccessMode != nil {
		copied := *c.AccessMode
		result.AccessMode = &copied
		switch cr.AccessMode.Request {
		case ReadOnlyShared:
			result.AccessMode.ReadOnlyShared += 1
		case ReadWriteShared:
			result.AccessMode.ReadWriteShared += 1
		case WriteExclusive:
			result.AccessMode.WriteExclusive += 1
		case ReadWriteExclusive:
			result.AccessMode.ReadWriteExclusive += 1
		}
		return result, nil
	}

	return ResourceCapacity{}, fmt.Errorf("request/capacity type mismatch")
}

func roundUpToBlock(q, size resource.Quantity) resource.Quantity {
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

func roundDownToBlock(q, size resource.Quantity) resource.Quantity {
	qi := qtoi(q)
	si := qtoi(size)
	qi.Div(qi, si)
	qi.Mul(qi, si)

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

		for _, rcr := range ccr.DeviceClaimResults {
			msg = rcr.FailureReason
			if rcr.Success() {
				msg = "succeeded"
			}
			fmt.Printf("  - device claim %q (%d): %s\n", rcr.ClaimName, rcr.Score(), msg)

			for pri, pr := range rcr.PoolResults {
				msg = pr.FailureReason
				if pr.Success() {
					msg = "succeeded"
				}
				if pri == rcr.Best {
					msg = "best"
				}
				fmt.Printf("    - pool %q (%d): %s\n", pr.PoolName, pr.Score(), msg)
				for rri, rr := range pr.DeviceResults {
					msg = rr.FailureReason
					if rr.Success() {
						msg = "success"
					}
					if rri == pr.Best {
						msg = "best"
					}
					fmt.Printf("      - device %q (%d): %s\n", rr.DeviceName, rr.Score, msg)
				}
			}
		}
	}
}

// CapacityClaimResult methods

func (ccr *CapacityClaimResult) Success() bool {
	for _, rcr := range ccr.DeviceClaimResults {
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
	for _, r := range ccr.DeviceClaimResults {
		score += r.Score()
	}

	return score / len(ccr.DeviceClaimResults)
}

// DeviceClaimResult methods

func (rcr *DeviceClaimResult) Success() bool {
	return rcr.Best >= 0
}

func (rcr *DeviceClaimResult) Score() int {
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

	return pr.DeviceResults[pr.Best].Score
}

// DeviceResult methods

func (rr *DeviceResult) Success() bool {
	return rr.Score > 0
}

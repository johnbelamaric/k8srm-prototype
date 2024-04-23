package schedule

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

func dumpTestCase(tn string, claim PodCapacityClaim) {
	if os.Getenv("DUMP_TEST_CASES") != "y" {
		return
	}

	cleanup := func(r rune) rune {
		if r < 'a' || r > 'z' {
			return '-'
		}
		return r
	}
	file := "testdata/pod-" + strings.Map(cleanup, strings.ToLower(tn)) + ".yaml"

	b, _ := yaml.Marshal(claim)
	err := os.WriteFile(file, b, 0644)
	if err != nil {
		fmt.Printf("error saving file %q: %s\n", file, err)
	}
}

func TestSchedulePodForFoozer(t *testing.T) {
	testCases := map[string]struct {
		claim         PodCapacityClaim
		expectSuccess bool
	}{
		"single pod, container, cpu/mem, and foozer": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []DeviceClaim{
						genClaimFoozer("foozer", "1", "2Gi", 0),
					},
				},
			},
			expectSuccess: true,
		},
		"no foozer big enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []DeviceClaim{
						genClaimFoozer("foozer", "16", "32Gi", 0),
					},
				},
			},
			expectSuccess: false,
		},
	}

	for tn, tc := range testCases {
		capacity := GenCapShapeOne(2)
		t.Run(tn, func(t *testing.T) {
			dumpTestCase(tn, tc.claim)
			allocation := SchedulePod(capacity, tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
		})
	}
}

func TestSchedulePodForBigFoozer(t *testing.T) {
	testCases := map[string]struct {
		claim         PodCapacityClaim
		expectSuccess bool
	}{
		"single pod, container, cpu/mem, and foozer": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []DeviceClaim{
						genClaimFoozer("foozer", "1", "2Gi", 0),
					},
				},
			},
			expectSuccess: true,
		},
		"no foozer big enough": {
			claim: PodCapacityClaim{
				PodClaim: CapacityClaim{
					Name: "my-foozer-pod",
					Claims: []DeviceClaim{
						genClaimFoozer("foozer", "16", "32Gi", 0),
					},
				},
			},
			expectSuccess: true,
		},
	}

	for tn, tc := range testCases {
		capacity := GenCapShapeTwo(2, 4)
		t.Run(tn, func(t *testing.T) {
			dumpTestCase(tn, tc.claim)
			allocation := SchedulePod(capacity, tc.claim)
			require.Equal(t, tc.expectSuccess, allocation != nil)
		})
	}
}

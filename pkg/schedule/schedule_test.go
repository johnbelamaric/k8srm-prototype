package schedule

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/johnbelamaric/k8srm-prototype/pkg/api"
	"github.com/johnbelamaric/k8srm-prototype/pkg/gen"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/yaml"
)

func dumpTestClaims(tn string, claims []api.DeviceClaim) {
	if os.Getenv("DUMP_TEST_CASES") != "y" {
		return
	}

	cleanup := func(r rune) rune {
		if r < 'a' || r > 'z' {
			return '-'
		}
		return r
	}
	file := "testdata/claims-" + strings.Map(cleanup, strings.ToLower(tn)) + ".yaml"

	b, _ := yaml.Marshal(claims)
	err := os.WriteFile(file, b, 0644)
	if err != nil {
		fmt.Printf("error saving file %q: %s\n", file, err)
	}
}

func TestSelectNode(t *testing.T) {
	testCases := map[string]struct {
		claims        []api.DeviceClaim
		pools         []api.DevicePool
		expectSuccess bool
	}{
		"single-by-driver": {
			claims: []api.DeviceClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "myclaim",
						Namespace: "default",
					},
					Spec: api.DeviceClaimSpec{
						DeviceClass: "not-implemented-yet",
						Driver:      ptr("example.com-foozer"),
					},
				},
			},
			pools:         gen.GenShapeZero(2),
			expectSuccess: true,
		},
		"multiple-single-pool": {
			claims: []api.DeviceClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "myclaim",
						Namespace: "default",
					},
					Spec: api.DeviceClaimSpec{
						DeviceClass:    "not-implemented-yet",
						Driver:         ptr("example.com-foozer"),
						MinDeviceCount: ptr(2),
					},
				},
			},
			pools:         gen.GenShapeZero(2),
			expectSuccess: true,
		},
		"split-across-pools": {
			claims: []api.DeviceClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "myclaim",
						Namespace: "default",
					},
					Spec: api.DeviceClaimSpec{
						DeviceClass:    "not-implemented-yet",
						Driver:         ptr("example.com-foozer"),
						MinDeviceCount: ptr(4),
					},
				},
			},
			pools:         gen.GenShapeOne(2),
			expectSuccess: true,
		},
	}

	for tn, tc := range testCases {
		t.Run(tn, func(t *testing.T) {
			dumpTestClaims(tn, tc.claims)
			allocations, results := SelectNode(tc.claims, tc.pools)
			b, _ := yaml.Marshal(allocations)
			fmt.Println("ALLOCATIONS")
			fmt.Println("-----------")
			fmt.Println(string(b))
			fmt.Println("NODE RESULTS")
			fmt.Println("------------")
			b, _ = yaml.Marshal(results)
			fmt.Println(string(b))
			require.Equal(t, tc.expectSuccess, allocations != nil)
		})
	}
}

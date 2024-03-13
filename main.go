package main

import (
	"flag"
	"fmt"
	"os"
	"sigs.k8s.io/yaml"
)

var nodesFlag, claimFlag string

func init() {
	flag.StringVar(&nodesFlag, "nodes", "", "file with []NodeResources yaml")
	flag.StringVar(&claimFlag, "claim", "", "file with PodCapacityClaim yaml")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %s -nodes <file> -claim <file> pod\n", os.Args[0])
	flag.PrintDefaults()
}

func genExample() {
	nrs := genCapShapeZero(1)

	b, _ := yaml.Marshal(nrs)
	fmt.Println(string(b))
	fmt.Println("---")
	claim := PodCapacityClaim{
		PodClaim: CapacityClaim{
			Name:   "my-pod",
			Claims: []ResourceClaim{genClaimPod()},
		},
		ContainerClaims: []CapacityClaim{
			{
				Name:   "my-container-1",
				Claims: []ResourceClaim{genClaimContainer("7127m", "8Gi")},
			},
			{
				Name:   "my-container-2",
				Claims: []ResourceClaim{genClaimContainer("200m", "8Gi")},
			},
			{
				Name:   "my-container-3",
				Claims: []ResourceClaim{genClaimContainer("200m", "8Gi")},
			},
		},
	}

	b, _ = yaml.Marshal(claim)
	fmt.Println(string(b))
}

func unmarshalFile(file string, obj interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, obj)
}

func schedulePod(nodesFile, claimFile string) error {

	var nrs []NodeResources
	err := unmarshalFile(nodesFile, &nrs)
	if err != nil {
		return err
	}

	var claim PodCapacityClaim
	err = unmarshalFile(claimFile, &claim)
	if err != nil {
		return err
	}

	allocation := SchedulePod(nrs, claim)
	b, _ := yaml.Marshal(allocation)
	fmt.Println(string(b))

	return nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) != 1 {
		usage()
		os.Exit(1)
	}

	cmd := flag.Args()[0]
	switch cmd {
	case "gen-example":
		genExample()
		break
	case "pod":
		err := schedulePod(nodesFlag, claimFlag)
		if err != nil {
			fmt.Fprintf(flag.CommandLine.Output(), "error: %s\n", err)
			os.Exit(1)
		}
		break
	default:
		usage()
		os.Exit(1)
	}
}

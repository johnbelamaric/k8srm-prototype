package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/johnbelamaric/k8srm-prototype/pkg/old/schedule"

	"sigs.k8s.io/yaml"
)

var flagNodes, flagClaim string
var flagVerbose bool

func init() {
	flag.StringVar(&flagNodes, "nodes", "", "file with []NodeDevices yaml")
	flag.StringVar(&flagClaim, "claim", "", "file with PodCapacityClaim yaml")
	flag.BoolVar(&flagVerbose, "v", false, "verbose output")
	flag.Usage = usage
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "usage: %s -nodes <file> -claim <file> pod\n", os.Args[0])
	fmt.Fprintf(flag.CommandLine.Output(), "       %s gen-example <shape>\n", os.Args[0])
	flag.PrintDefaults()
}

func genCapacityExample(shape string) {
	var nrs []schedule.NodeDevices

	switch shape {
	case "0":
		nrs = schedule.GenCapShapeZero(1)
	case "1":
		nrs = schedule.GenCapShapeOne(1)
	case "2":
		nrs = schedule.GenCapShapeTwo(1, 2)
	case "3":
		nrs = schedule.GenCapShapeThree(1, 2)
	default:
		fmt.Printf("unknown shape %q\n", shape)
	}

	if nrs != nil {
		b, _ := yaml.Marshal(nrs)
		fmt.Println(string(b))
	}
}

func unmarshalFile(file string, obj interface{}) error {
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, obj)
}

func schedulePod(nodesFile, claimFile string) error {

	var nrs []schedule.NodeDevices
	err := unmarshalFile(nodesFile, &nrs)
	if err != nil {
		return err
	}

	var claim schedule.PodCapacityClaim
	err = unmarshalFile(claimFile, &claim)
	if err != nil {
		return err
	}

	results, best := schedule.EvaluateNodesForPod(nrs, claim)
	if best < 0 {
		fmt.Println("failed to satisfy the claim")
		if flagVerbose {
			for _, nar := range results {
				fmt.Println("-------------------------------")
				nar.PrintSummary()
			}
		}
	} else {
		fmt.Println("succeeded in satisfying the claim")
		fmt.Println("-------------------------------")
		if flagVerbose {
			b, _ := yaml.Marshal(results[best])
			fmt.Println(string(b))
		} else {
			results[best].PrintSummary()
		}
	}

	return nil
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(1)
	}

	switch args[0] {
	case "gen-example":
		shape := "0"
		if len(args) > 1 {
			shape = args[1]
		}
		genCapacityExample(shape)
		break
	case "pod":
		err := schedulePod(flagNodes, flagClaim)
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

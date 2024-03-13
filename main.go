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
	fmt.Fprintf(flag.CommandLine.Output(), "       %s gen-example <shape>\n", os.Args[0])
	flag.PrintDefaults()
}

func genCapacityExample(shape string) {
	var nrs []NodeResources

	switch shape {
	case "0":
		nrs = genCapShapeZero(1)
	case "1":
		nrs = genCapShapeOne(1)
	case "2":
		nrs = genCapShapeTwo(1, 2)
	case "3":
		nrs = genCapShapeThree(1, 2)
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

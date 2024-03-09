package main

import (
	"fmt"
	"sigs.k8s.io/yaml"
)

func main() {
	nrs := genCapShapeZero(4)
	nrs = append(nrs, genCapShapeOne(4)...)
	nrs = append(nrs, genCapShapeTwo(8, 2)...)
	nrs = append(nrs, genCapShapeThree(8, 4)...)

	b, _ := yaml.Marshal(nrs)
	fmt.Println(string(b))
}

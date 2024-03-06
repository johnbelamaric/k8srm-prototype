package main

import (
	"fmt"
	"sigs.k8s.io/yaml"
)

func main() {
	nrs := genShapeZero(4)
	nrs = append(nrs, genShapeOne(4)...)
	nrs = append(nrs, genShapeTwo(8, 2)...)
	nrs = append(nrs, genShapeThree(8, 4)...)

	b, _ := yaml.Marshal(nrs)
	fmt.Println(string(b))
}

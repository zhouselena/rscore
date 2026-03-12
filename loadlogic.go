package rscore

import (
	"log"
)

// SampleStruct description.
type SampleStruct struct {
	Name	string
	SampleId int64
	Toggle bool
	ExtraInfo any
	Args map[string]any
}

// description
func Load(nodespath string, edgespath string) (string, error) {

	if (nodespath == "") {
		nodespath = "public/templates/nodes.csv"
	}
	
	if (edgespath == "") {
		edgespath = "public/templates/edges.csv"
	}

	// logic
	str := "Nodes path: " + nodespath + "\n" + "Edges path: " + edgespath
	log.Printf("%s", str)

	return str, nil

}
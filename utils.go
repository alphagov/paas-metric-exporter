package main

import (
	"encoding/json"
	"fmt"
)

// MetricVars will contain the variables the tenant could use to compose their
// custom metric namespace.
type MetricVars struct {
	App          string
	GUID         string
	Index        string
	Job          string
	Metric       string
	Organisation string
	Space        string
}

func deb(v interface{}) {
	b, _ := json.Marshal(v)
	fmt.Printf("\n\n\n%s\n\n\n", string(b))
}

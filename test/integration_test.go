package test

import "flag"

var integrationTest bool

func init() {
	flag.BoolVar(&integrationTest, "integration", false, "run integration tests")
}

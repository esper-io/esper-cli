package main

import "testing"

func TestCurrentSpecContract(t *testing.T) {
	if issues := check("../../spec/openapi"); len(issues) != 0 {
		t.Fatalf("contract issues: %v", issues)
	}
}

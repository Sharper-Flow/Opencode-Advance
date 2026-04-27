package config

import "testing"

// TestValidate_PortRangeAllows6325 asserts the upper bound of the MCP port
// range is 6325 (matching Vision v1.1.1's MaxPort), not the historical 6300.
// Slot groups need this range to host pools and group ports.
func TestValidate_PortRangeAllows6325(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.edge-of-range]
port    = 6325
command = "echo"
`
	stack := mustParse(t, src)
	if err := stack.Validate(); err != nil {
		t.Fatalf("port 6325 should be in valid range; got: %v", err)
	}
}

// TestValidate_PortRangeRejects6326 asserts ports above 6325 still fail.
func TestValidate_PortRangeRejects6326(t *testing.T) {
	src := `
[meta]
version = "1.0.0"

[mcp.servers.over-range]
port    = 6326
command = "echo"
`
	stack := mustParse(t, src)
	err := stack.Validate()
	if err == nil {
		t.Fatal("port 6326 should be rejected; got nil")
	}
	if !ValidationErrorContains(err, "out of range") {
		t.Errorf("expected 'out of range' in error; got: %v", err)
	}
}

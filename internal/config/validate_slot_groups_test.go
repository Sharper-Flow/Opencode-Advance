package config

import "testing"

// TestValidate_SlotGroups covers all error paths for [mcp.slot_groups.<name>]:
//   - required fields (template, base_port, count, group_port)
//   - count >= 2 (Vision rejects single-slot pools)
//   - base_port + count - 1 within port range
//   - group_port within port range
//   - reject `port` on `defaults` (Vision derives slot ports from base_port)
//   - port collision: group_port vs declared servers
//   - port collision: slot ports vs declared servers
//   - port collision: across slot groups (group_port vs group_port, slot vs slot)
//   - template name collision: <template>-N vs declared servers.* keys
//   - happy path
func TestValidate_SlotGroups(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr bool
		wantSub string // substring expected in aggregated error
	}{
		{
			name: "happy path — minimal slot group",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6300
`,
			wantErr: false,
		},
		{
			name: "missing template",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
base_port  = 6301
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "template",
		},
		{
			name: "missing base_port",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "base_port",
		},
		{
			name: "missing group_port",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template  = "pw-slot"
base_port = 6301
count     = 4
`,
			wantErr: true,
			wantSub: "group_port",
		},
		{
			name: "count below 2",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 1
group_port = 6300
`,
			wantErr: true,
			wantSub: "count",
		},
		{
			name: "base_port + count exceeds maxPort",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6324
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "out of range",
		},
		{
			name: "group_port out of range (above maxPort)",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6326
`,
			wantErr: true,
			wantSub: "out of range",
		},
		{
			name: "defaults sets port (forbidden)",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6300

[mcp.slot_groups.pw.defaults]
port    = 6299
command = "echo"
`,
			wantErr: true,
			wantSub: "defaults.port",
		},
		{
			name: "group_port collides with declared server",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port    = 6300
command = "echo"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "port 6300",
		},
		{
			name: "slot port collides with declared server",
			src: `
[meta]
version = "1.0.0"

[mcp.servers.a]
port    = 6302
command = "echo"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "port 6302",
		},
		{
			name: "slot port collides across slot groups",
			src: `
[meta]
version = "1.0.0"

[mcp.slot_groups.pw1]
template   = "pw1-slot"
base_port  = 6301
count      = 4
group_port = 6300

[mcp.slot_groups.pw2]
template   = "pw2-slot"
base_port  = 6304
count      = 2
group_port = 6310
`,
			wantErr: true,
			wantSub: "port 6304",
		},
		{
			name: "template name collides with declared server",
			src: `
[meta]
version = "1.0.0"

[mcp.servers."pw-slot-2"]
port    = 6299
command = "echo"

[mcp.slot_groups.pw]
template   = "pw-slot"
base_port  = 6301
count      = 4
group_port = 6300
`,
			wantErr: true,
			wantSub: "pw-slot-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stack := mustParse(t, tt.src)
			err := stack.Validate()
			if tt.wantErr && err == nil {
				t.Fatalf("expected validation error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected nil, got: %v", err)
			}
			if tt.wantSub != "" && !ValidationErrorContains(err, tt.wantSub) {
				t.Errorf("expected substring %q in error; got: %v", tt.wantSub, err)
			}
		})
	}
}

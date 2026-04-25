package temporal

import "testing"

func TestParseNodeMajorVersion(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"v20.1.0", 20, false},
		{"v18.17.0", 18, false},
		{"v22-nightly", 22, false},
		{"v22.0.0-nightly", 22, false},
		{"v23.11.0", 23, false},
		{"malformed", 0, true},
		{"", 0, true},
		{"v", 0, true},
		{"vX.Y.Z", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseNodeMajorVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseNodeMajorVersion(%q) = %d, want error", tt.input, got)
				}
			} else {
				if err != nil {
					t.Errorf("parseNodeMajorVersion(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("parseNodeMajorVersion(%q) = %d, want %d", tt.input, got, tt.want)
				}
			}
		})
	}
}

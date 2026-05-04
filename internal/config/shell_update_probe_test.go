package config

import (
	"strings"
	"testing"
)

func TestParse_ShellAndUpdateProbeSectionsTyped(t *testing.T) {
	src := minimalTOML + `

[shell]
auto_refresh = false
auto_refresh_notice = "once"

[update_probe]
default = "warn"
timeout_per_plugin_ms = 2500
timeout_global_ms = 9000
cache_ttl_minutes = 7
`

	stack := mustParse(t, src)

	if _, ok := stack.DeferredSections["shell"]; ok {
		t.Fatal("shell should be typed, not deferred")
	}
	if _, ok := stack.DeferredSections["update_probe"]; ok {
		t.Fatal("update_probe should be typed, not deferred")
	}
	if stack.Shell.IsAutoRefreshEnabled() {
		t.Fatal("Shell.IsAutoRefreshEnabled() = true, want false")
	}
	if stack.Shell.AutoRefreshNotice != "once" {
		t.Fatalf("Shell.AutoRefreshNotice = %q, want once", stack.Shell.AutoRefreshNotice)
	}
	if stack.UpdateProbe.Default != "warn" {
		t.Fatalf("UpdateProbe.Default = %q, want warn", stack.UpdateProbe.Default)
	}
	if stack.UpdateProbe.TimeoutPerPluginMS != 2500 || stack.UpdateProbe.TimeoutGlobalMS != 9000 || stack.UpdateProbe.CacheTTLMinutes != 7 {
		t.Fatalf("UpdateProbe time/cache defaults parsed wrong: %+v", stack.UpdateProbe)
	}
}

func TestValidate_ShellAndUpdateProbeDefaults(t *testing.T) {
	stack := mustParse(t, minimalTOML)
	if err := stack.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}

	if !stack.Shell.IsAutoRefreshEnabled() {
		t.Fatal("Shell.IsAutoRefreshEnabled() = false, want true default")
	}
	if stack.Shell.AutoRefreshNotice != "off" {
		t.Fatalf("Shell.AutoRefreshNotice = %q, want off", stack.Shell.AutoRefreshNotice)
	}
	if stack.UpdateProbe.Default != "passive" {
		t.Fatalf("UpdateProbe.Default = %q, want passive", stack.UpdateProbe.Default)
	}
	if stack.UpdateProbe.TimeoutPerPluginMS != 3000 {
		t.Fatalf("TimeoutPerPluginMS = %d, want 3000", stack.UpdateProbe.TimeoutPerPluginMS)
	}
	if stack.UpdateProbe.TimeoutGlobalMS != 10000 {
		t.Fatalf("TimeoutGlobalMS = %d, want 10000", stack.UpdateProbe.TimeoutGlobalMS)
	}
	if stack.UpdateProbe.CacheTTLMinutes != 5 {
		t.Fatalf("CacheTTLMinutes = %d, want 5", stack.UpdateProbe.CacheTTLMinutes)
	}
}

func TestValidate_ShellAndUpdateProbeInvalidValues(t *testing.T) {
	tests := []struct {
		name    string
		extra   string
		wantSub string
	}{
		{
			name: "bad auto refresh notice",
			extra: `
[shell]
auto_refresh_notice = "loud"
`,
			wantSub: "shell.auto_refresh_notice",
		},
		{
			name: "bad update probe mode",
			extra: `
[update_probe]
default = "always"
`,
			wantSub: "update_probe.default",
		},
		{
			name: "bad per plugin timeout",
			extra: `
[update_probe]
timeout_per_plugin_ms = 0
`,
			wantSub: "update_probe.timeout_per_plugin_ms",
		},
		{
			name: "bad global timeout",
			extra: `
[update_probe]
timeout_global_ms = -1
`,
			wantSub: "update_probe.timeout_global_ms",
		},
		{
			name: "bad cache ttl",
			extra: `
[update_probe]
cache_ttl_minutes = 0
`,
			wantSub: "update_probe.cache_ttl_minutes",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stack := mustParse(t, minimalTOML+tc.extra)
			err := stack.Validate()
			if err == nil {
				t.Fatal("Validate() = nil, want error")
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("Validate() = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}

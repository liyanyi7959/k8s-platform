package ssh

import "testing"

func TestParseProbeOutput(t *testing.T) {
	result := parseProbeOutput("Linux\n6.8.0\nNAME=Ubuntu\nVERSION_ID=24.04\n---HW---\n4\n8192\n120\n")
	if result.OS != "Ubuntu" || result.OSVersion != "24.04" || result.Kernel != "6.8.0" {
		t.Fatalf("unexpected platform probe: %#v", result)
	}
	if result.CPUCores == nil || *result.CPUCores != 4 || result.MemoryMB == nil || *result.MemoryMB != 8192 || result.DiskGB == nil || *result.DiskGB != 120 {
		t.Fatalf("unexpected hardware probe: %#v", result)
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("a'b"); got != `'a'"'"'b'` {
		t.Fatalf("shell quote = %q", got)
	}
}

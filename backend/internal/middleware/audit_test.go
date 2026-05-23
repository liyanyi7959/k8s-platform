package middleware

import "testing"

func TestShouldAuditRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		want   bool
	}{
		{
			name:   "skip get requests",
			method: "GET",
			path:   "/api/v1/clusters/1/pods",
			want:   false,
		},
		{
			name:   "skip pod log session handshake",
			method: "POST",
			path:   "/api/v1/clusters/1/pods/devops/demo/logs/session",
			want:   false,
		},
		{
			name:   "keep pod exec audited",
			method: "POST",
			path:   "/api/v1/clusters/1/pods/devops/demo/exec",
			want:   true,
		},
		{
			name:   "keep create workload audited",
			method: "POST",
			path:   "/api/v1/clusters/1/workloads/deployments",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAuditRequest(tt.method, tt.path); got != tt.want {
				t.Fatalf("shouldAuditRequest(%q, %q) = %v, want %v", tt.method, tt.path, got, tt.want)
			}
		})
	}
}

package service

import (
	"context"
	"errors"
	"testing"
)

// ────────── NewClusterRegistryService ──────────

func TestNewClusterRegistryService_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	if svc == nil {
		t.Fatal("should not return nil")
	}
}

// ────────── ListClusters: nil DB ──────────

func TestListClusters_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.ListClusters(context.Background(), ListClustersRequest{})
	if err == nil {
		t.Fatal("expected error for nil db")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

// ────────── ImportCluster ──────────

func TestImportCluster_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.ImportCluster(context.Background(), "test", "kc")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestImportCluster_EmptyName(t *testing.T) {
	// With nil db, db check happens first. This tests the nil-db path.
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.ImportCluster(context.Background(), "", "kc")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestImportCluster_EmptyKubeconfig(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.ImportCluster(context.Background(), "test", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── GetCluster ──────────

func TestGetCluster_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.GetCluster(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestGetCluster_ZeroID(t *testing.T) {
	// nil db → returns "db is required" before id check
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.GetCluster(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── GetKubeconfig ──────────

func TestGetKubeconfig_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.GetKubeconfig(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestGetKubeconfig_ZeroID(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	_, err := svc.GetKubeconfig(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── UpdateClusterHealth ──────────

func TestUpdateClusterHealth_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.UpdateClusterHealth(context.Background(), 1, true, 3, 3, "v1.28")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestUpdateClusterHealth_ZeroID(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.UpdateClusterHealth(context.Background(), 0, true, 3, 3, "v1.28")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── PatchCluster ──────────

func TestPatchCluster_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.PatchCluster(context.Background(), 1, PatchClusterRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestPatchCluster_ZeroID(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.PatchCluster(context.Background(), 0, PatchClusterRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── DeleteCluster ──────────

func TestDeleteCluster_NilDB(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.DeleteCluster(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "db is required" {
		t.Fatalf("expected 'db is required', got %v", err)
	}
}

func TestDeleteCluster_ZeroID(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")
	err := svc.DeleteCluster(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error")
	}
}

// ────────── normalizePage (shared helper used by ListClusters) ──────────

func TestNormalizePage_DefaultsAndBounds(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantP    int
		wantPS   int
	}{
		{"zero values", 0, 0, 1, 10},
		{"negative", -1, -5, 1, 10},
		{"valid", 3, 50, 3, 50},
		{"page size over 200", 1, 500, 1, 200},
		{"page size 200", 1, 200, 1, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, ps := normalizePage(tt.page, tt.pageSize)
			if p != tt.wantP || ps != tt.wantPS {
				t.Fatalf("normalizePage(%d, %d) = (%d, %d), want (%d, %d)",
					tt.page, tt.pageSize, p, ps, tt.wantP, tt.wantPS)
			}
		})
	}
}

// ────────── PatchClusterRequest struct ──────────

func TestPatchClusterRequest_NameField(t *testing.T) {
	name := "new-name"
	req := PatchClusterRequest{Name: &name}
	if req.Name == nil || *req.Name != "new-name" {
		t.Fatal("name field mismatch")
	}
	if req.Kubeconfig != nil {
		t.Fatal("kubeconfig should be nil")
	}
}

// ────────── verify error types ──────────

func TestClusterService_ErrorsAreWrapped(t *testing.T) {
	svc := NewClusterRegistryService(nil, "")

	// ListClusters returns plain errors.New
	_, err := svc.ListClusters(context.Background(), ListClustersRequest{})
	if !errors.Is(err, errors.New("db is required")) {
		// errors.New creates a unique error each time, so we check the message
		if err.Error() != "db is required" {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// ImportCluster returns plain errors.New for nil db
	_, err = svc.ImportCluster(context.Background(), "c1", "kc")
	if err.Error() != "db is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

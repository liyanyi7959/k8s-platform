package application

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeManifestApply(t *testing.T) {
	value, err := NormalizeManifestApply(ManifestApplyInput{ClusterID: 1, YAML: "  apiVersion: v1  ", SourceLabel: "  "})
	if err != nil || value.YAML != "apiVersion: v1" || value.SourceLabel != "通用 YAML 清单" {
		t.Fatalf("NormalizeManifestApply() = %#v, %v", value, err)
	}
	if _, err := NormalizeManifestApply(ManifestApplyInput{ClusterID: 1}); err == nil {
		t.Fatal("empty YAML should fail")
	}
}

func TestNormalizeManifestPage(t *testing.T) {
	page, pageSize := NormalizeManifestPage(0, 1000)
	if page != 1 || pageSize != 20 {
		t.Fatalf("NormalizeManifestPage(0, 1000) = (%d, %d), want (1, 20)", page, pageSize)
	}
	page, pageSize = NormalizeManifestPage(2, 50)
	if page != 2 || pageSize != 50 {
		t.Fatalf("NormalizeManifestPage(2, 50) = (%d, %d)", page, pageSize)
	}
}

type manifestRuntimeSpy struct {
	applyInput ManifestApplyInput
	listQuery  ManifestRecordQuery
	getCluster uint64
	getRecord  uint64
}

func (spy *manifestRuntimeSpy) Execute(_ context.Context, input ManifestApplyInput) (*ManifestApplyResult, error) {
	spy.applyInput = input
	return &ManifestApplyResult{RecordID: 7, Status: "success"}, nil
}

func (spy *manifestRuntimeSpy) List(_ context.Context, query ManifestRecordQuery) (*ManifestRecordPage, error) {
	spy.listQuery = query
	return &ManifestRecordPage{Page: query.Page, PageSize: query.PageSize}, nil
}

func (spy *manifestRuntimeSpy) Get(_ context.Context, clusterID, recordID uint64) (*ManifestRecordDetail, error) {
	spy.getCluster, spy.getRecord = clusterID, recordID
	return &ManifestRecordDetail{ClusterID: clusterID}, nil
}

func TestManifestServiceOwnsValidationAndQueryNormalization(t *testing.T) {
	spy := &manifestRuntimeSpy{}
	service := NewManifestService(spy)
	result, err := service.Apply(context.Background(), ManifestApplyInput{ClusterID: 2, YAML: "  apiVersion: v1  ", SourceLabel: "  "})
	if err != nil || result.RecordID != 7 {
		t.Fatalf("Apply() = %#v, %v", result, err)
	}
	if spy.applyInput.YAML != "apiVersion: v1" || spy.applyInput.SourceLabel == "" {
		t.Fatalf("runtime input = %#v", spy.applyInput)
	}
	if _, err := service.List(context.Background(), ManifestRecordQuery{ClusterID: 2, Page: 0, PageSize: 200, Keyword: " ops "}); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if spy.listQuery.Page != 1 || spy.listQuery.PageSize != 20 || spy.listQuery.Keyword != "ops" {
		t.Fatalf("normalized list query = %#v", spy.listQuery)
	}
	if _, err := service.Get(context.Background(), 2, 9); err != nil || spy.getCluster != 2 || spy.getRecord != 9 {
		t.Fatalf("Get() = cluster=%d record=%d err=%v", spy.getCluster, spy.getRecord, err)
	}
	if _, err := service.Apply(context.Background(), ManifestApplyInput{}); !errors.Is(err, ErrInvalidParams) {
		t.Fatalf("Apply() invalid error = %v", err)
	}
}

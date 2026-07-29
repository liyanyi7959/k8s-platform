package application

import (
	"context"
	"strings"
	"time"
)

type MetricPoint struct { Timestamp int64 `json:"timestamp"`; Value float64 `json:"value"` }
type MetricsTrendQuery struct { ClusterID uint64; Target string; Name string; Namespace string; Metric string; Start time.Time; End time.Time; Step time.Duration }
type MetricsRuntime interface {
	NodeMetrics(context.Context,uint64) (any,error); PodMetrics(context.Context,uint64,string) (any,error)
	Source(context.Context,uint64) (any,error); Detect(context.Context,uint64) (any,error); Switch(context.Context,uint64,string) error
	Trend(context.Context,MetricsTrendQuery) (any,error); HealthCheck(context.Context,uint64) (any,error)
}
type MetricsService struct { runtime MetricsRuntime }
func NewMetricsService(runtime MetricsRuntime) *MetricsService { return &MetricsService{runtime:runtime} }
func (s *MetricsService) NodeMetrics(ctx context.Context, clusterID uint64) (any,error) { if err:=validateMetricsCluster(clusterID);err!=nil{return nil,err}; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.NodeMetrics(ctx,clusterID) }
func (s *MetricsService) PodMetrics(ctx context.Context, clusterID uint64, namespace string) (any,error) { if err:=validateMetricsCluster(clusterID);err!=nil{return nil,err}; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.PodMetrics(ctx,clusterID,strings.TrimSpace(namespace)) }
func (s *MetricsService) Source(ctx context.Context, clusterID uint64) (any,error) { if err:=validateMetricsCluster(clusterID);err!=nil{return nil,err}; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.Source(ctx,clusterID) }
func (s *MetricsService) Detect(ctx context.Context, clusterID uint64) (any,error) { if err:=validateMetricsCluster(clusterID);err!=nil{return nil,err}; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.Detect(ctx,clusterID) }
func (s *MetricsService) Switch(ctx context.Context, clusterID uint64, source string) error { if err:=validateMetricsCluster(clusterID);err!=nil{return err}; source=strings.ToLower(strings.TrimSpace(source)); if source!="auto"&&source!="prometheus"&&source!="metrics_server" {return ErrInvalidParams}; if s==nil||s.runtime==nil{return ErrConflict}; return s.runtime.Switch(ctx,clusterID,source) }
func (s *MetricsService) Trend(ctx context.Context, query MetricsTrendQuery) (any,error) { query.Target=strings.ToLower(strings.TrimSpace(query.Target)); query.Name=strings.TrimSpace(query.Name); query.Namespace=strings.TrimSpace(query.Namespace); query.Metric=strings.ToLower(strings.TrimSpace(query.Metric)); if query.ClusterID==0||query.Name==""||query.Start.IsZero()||query.End.IsZero()||query.Step<=0||query.End.Before(query.Start)||(query.Target!="node"&&query.Target!="pod")||(query.Metric!="cpu"&&query.Metric!="memory") { return nil,ErrInvalidParams }; if query.Target=="pod"&&query.Namespace=="" { return nil,ErrInvalidParams }; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.Trend(ctx,query) }
func (s *MetricsService) HealthCheck(ctx context.Context, clusterID uint64) (any,error) { if err:=validateMetricsCluster(clusterID);err!=nil{return nil,err}; if s==nil||s.runtime==nil{return nil,ErrConflict}; return s.runtime.HealthCheck(ctx,clusterID) }
func validateMetricsCluster(clusterID uint64) error { if clusterID==0{return ErrInvalidParams}; return nil }

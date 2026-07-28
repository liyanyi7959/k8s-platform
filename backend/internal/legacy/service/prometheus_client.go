// prometheus_client.go 封装 Prometheus HTTP API 的基础调用。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// prometheusResponse 是 Prometheus API 的标准响应结构。
type prometheusResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
}

// prometheusQueryResult 是瞬时查询结果。
type prometheusQueryResult struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	} `json:"result"`
}

// prometheusQueryRangeResult 是范围查询结果。
type prometheusQueryRangeResult struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Values [][]interface{}   `json:"values"`
	} `json:"result"`
}

// prometheusClient 提供对 Prometheus HTTP API 的访问。
type prometheusClient struct {
	baseURL string
	client  *http.Client
}

// newPrometheusClient 创建 Prometheus 客户端。
func newPrometheusClient(baseURL string) *prometheusClient {
	return &prometheusClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

// query 执行 PromQL 瞬时查询。
func (c *prometheusClient) query(ctx context.Context, expr string) (prometheusQueryResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return prometheusQueryResult{}, err
	}
	q := u.Query()
	q.Set("query", expr)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return prometheusQueryResult{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return prometheusQueryResult{}, err
	}
	defer resp.Body.Close()

	return decodeQueryResponse(resp.Body)
}

// queryRange 执行 PromQL 范围查询。
func (c *prometheusClient) queryRange(ctx context.Context, expr string, start, end time.Time, step time.Duration) (prometheusQueryRangeResult, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	q := u.Query()
	q.Set("query", expr)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	defer resp.Body.Close()

	return decodeQueryRangeResponse(resp.Body)
}

// health 检查 Prometheus 是否健康。
func (c *prometheusClient) health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/-/healthy", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("prometheus health check returned %d", resp.StatusCode)
	}
	return nil
}

func decodeQueryResponse(r io.Reader) (prometheusQueryResult, error) {
	var wrapper prometheusResponse
	if err := json.NewDecoder(r).Decode(&wrapper); err != nil {
		return prometheusQueryResult{}, err
	}
	if wrapper.Status != "success" {
		return prometheusQueryResult{}, fmt.Errorf("prometheus query error: %s", wrapper.Error)
	}
	var result prometheusQueryResult
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return prometheusQueryResult{}, err
	}
	return result, nil
}

func decodeQueryRangeResponse(r io.Reader) (prometheusQueryRangeResult, error) {
	var wrapper prometheusResponse
	if err := json.NewDecoder(r).Decode(&wrapper); err != nil {
		return prometheusQueryRangeResult{}, err
	}
	if wrapper.Status != "success" {
		return prometheusQueryRangeResult{}, fmt.Errorf("prometheus query error: %s", wrapper.Error)
	}
	var result prometheusQueryRangeResult
	if err := json.Unmarshal(wrapper.Data, &result); err != nil {
		return prometheusQueryRangeResult{}, err
	}
	return result, nil
}

func parsePrometheusValue(v interface{}) float64 {
	switch val := v.(type) {
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	}
	return 0
}

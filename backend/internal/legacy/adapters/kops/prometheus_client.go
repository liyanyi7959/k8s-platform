package kops

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

type prometheusResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType string          `json:"errorType"`
	Error     string          `json:"error"`
}

type prometheusQueryResult struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Value  []interface{}     `json:"value"`
	} `json:"result"`
}

type prometheusQueryRangeResult struct {
	ResultType string `json:"resultType"`
	Result     []struct {
		Metric map[string]string `json:"metric"`
		Values [][]interface{}   `json:"values"`
	} `json:"result"`
}

type prometheusClient struct {
	baseURL string
	client  *http.Client
}

func newPrometheusClient(baseURL string) *prometheusClient {
	return &prometheusClient{baseURL: baseURL, client: &http.Client{Timeout: 10 * time.Second}}
}

func (c *prometheusClient) query(ctx context.Context, expression string) (prometheusQueryResult, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query")
	if err != nil {
		return prometheusQueryResult{}, err
	}
	query := endpoint.Query()
	query.Set("query", expression)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return prometheusQueryResult{}, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return prometheusQueryResult{}, err
	}
	defer response.Body.Close()
	return decodeQueryResponse(response.Body)
}

func (c *prometheusClient) queryRange(ctx context.Context, expression string, start, end time.Time, step time.Duration) (prometheusQueryRangeResult, error) {
	endpoint, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	query := endpoint.Query()
	query.Set("query", expression)
	query.Set("start", strconv.FormatInt(start.Unix(), 10))
	query.Set("end", strconv.FormatInt(end.Unix(), 10))
	query.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return prometheusQueryRangeResult{}, err
	}
	defer response.Body.Close()
	return decodeQueryRangeResponse(response.Body)
}

func (c *prometheusClient) health(ctx context.Context) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/-/healthy", nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("prometheus health check returned %d", response.StatusCode)
	}
	return nil
}

func decodeQueryResponse(reader io.Reader) (prometheusQueryResult, error) {
	var envelope prometheusResponse
	if err := json.NewDecoder(reader).Decode(&envelope); err != nil {
		return prometheusQueryResult{}, err
	}
	if envelope.Status != "success" {
		return prometheusQueryResult{}, fmt.Errorf("prometheus query error: %s", envelope.Error)
	}
	var result prometheusQueryResult
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		return prometheusQueryResult{}, err
	}
	return result, nil
}

func decodeQueryRangeResponse(reader io.Reader) (prometheusQueryRangeResult, error) {
	var envelope prometheusResponse
	if err := json.NewDecoder(reader).Decode(&envelope); err != nil {
		return prometheusQueryRangeResult{}, err
	}
	if envelope.Status != "success" {
		return prometheusQueryRangeResult{}, fmt.Errorf("prometheus query error: %s", envelope.Error)
	}
	var result prometheusQueryRangeResult
	if err := json.Unmarshal(envelope.Data, &result); err != nil {
		return prometheusQueryRangeResult{}, err
	}
	return result, nil
}

func parsePrometheusValue(value interface{}) float64 {
	switch typed := value.(type) {
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}

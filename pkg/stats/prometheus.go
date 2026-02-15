package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type QueryRangeResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string         `json:"resultType"`
		Result     []MatrixSeries `json:"result"`
	} `json:"data"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

type MatrixSeries struct {
	Metric map[string]string `json:"metric"`
	Values [][]interface{}   `json:"values"`
}

type SeriesPointStats struct {
	Series      int
	TotalPoints int
	MinPoints   int
	MaxPoints   int
}

type CoverageStats struct {
	HasPoints        bool
	Earliest         time.Time
	Latest           time.Time
	ObservedDuration time.Duration
}

func NewClient(baseURL string, timeout time.Duration) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("stats base URL is empty")
	}

	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryRangeResponse, error) {
	u, err := c.QueryRangeURL(query, start, end, step)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request stats API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stats API returned %s", resp.Status)
	}

	var out QueryRangeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	if out.Status != "success" {
		return nil, fmt.Errorf("stats query failed (%s): %s", out.ErrorType, out.Error)
	}

	return &out, nil
}

func (c *Client) QueryRangeURL(query string, start, end time.Time, step time.Duration) (string, error) {
	u, err := url.Parse(c.baseURL + "/api/v1/query_range")
	if err != nil {
		return "", fmt.Errorf("build query URL: %w", err)
	}

	q := u.Query()
	q.Set("query", query)
	q.Set("start", strconv.FormatInt(start.Unix(), 10))
	q.Set("end", strconv.FormatInt(end.Unix(), 10))
	q.Set("step", strconv.Itoa(int(step.Seconds())))
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func AverageSeriesValue(resp *QueryRangeResponse) (float64, int, error) {
	if resp == nil {
		return 0, 0, fmt.Errorf("nil query response")
	}

	var sum float64
	var n int

	for _, series := range resp.Data.Result {
		for _, point := range series.Values {
			if len(point) < 2 {
				continue
			}

			raw, ok := point[1].(string)
			if !ok {
				continue
			}

			v, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				continue
			}

			sum += v
			n++
		}
	}

	if n == 0 {
		return 0, 0, fmt.Errorf("no datapoints in query response")
	}

	return sum / float64(n), n, nil
}

func GetSeriesPointStats(resp *QueryRangeResponse) SeriesPointStats {
	stats := SeriesPointStats{}
	if resp == nil {
		return stats
	}

	stats.Series = len(resp.Data.Result)
	if stats.Series == 0 {
		return stats
	}

	min := -1
	max := 0
	total := 0
	for _, series := range resp.Data.Result {
		n := len(series.Values)
		total += n
		if min == -1 || n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}

	stats.TotalPoints = total
	stats.MinPoints = min
	stats.MaxPoints = max
	return stats
}

func GetCoverageStats(resp *QueryRangeResponse) CoverageStats {
	stats := CoverageStats{}
	if resp == nil {
		return stats
	}

	var earliest time.Time
	var latest time.Time
	found := false

	for _, series := range resp.Data.Result {
		for _, point := range series.Values {
			if len(point) < 1 {
				continue
			}

			rawTS, ok := point[0].(float64)
			if !ok {
				continue
			}

			ts := time.Unix(int64(rawTS), 0)
			if !found || ts.Before(earliest) {
				earliest = ts
			}
			if !found || ts.After(latest) {
				latest = ts
			}
			found = true
		}
	}

	if !found {
		return stats
	}

	stats.HasPoints = true
	stats.Earliest = earliest
	stats.Latest = latest
	if latest.After(earliest) {
		stats.ObservedDuration = latest.Sub(earliest)
	}
	return stats
}

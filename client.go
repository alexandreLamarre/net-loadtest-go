package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/alitto/pond/v2"
)

type WorkloadConfig struct {
	ConcurrentRequests int
	Delay              time.Duration
}

type Client struct {
	auth    string
	wConfig WorkloadConfig
}

func NewClient(auth string, wConfig WorkloadConfig) *Client {
	return &Client{
		auth:    auth,
		wConfig: wConfig,
	}
}

type tuple struct {
	success bool
	code    int
	err     error
}

func (c *Client) runHttp(ctx context.Context, addr string) {
	httpClient := http.DefaultClient
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/http/ping", addr), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", c.auth)
	pool := pond.NewResultPool[tuple](c.wConfig.ConcurrentRequests)
	results := make(chan tuple, 2*c.wConfig.ConcurrentRequests)
	success := 0
	total := 0
	errorCodes := map[int]int{}
	errorMap := map[string]int{}
	defer close(results)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case res := <-results:
				total += 1
				if res.success {
					success += 1
				} else {
					if _, ok := errorCodes[res.code]; !ok {
						errorCodes[res.code] = 0
					}
					errorCodes[res.code] += 1

					if res.err != nil {
						strErr := res.err.Error()
						if _, ok := errorMap[strErr]; !ok {
							errorMap[strErr] = 0
						}
						errorMap[strErr] += 1
					}
				}
			}
		}
	}()

	go func() {
		t := time.NewTicker(time.Second * 2)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				slog.Default().With("success", success, "failure", total-success).Info("reporting results")
				logger := slog.Default()
				for code, count := range errorCodes {
					if code == -1 {
						logger = logger.With("network-errors", count)
					} else {
						logger = logger.With(fmt.Sprintf("%d-count", code), count)
					}
				}
				// logger.Info("error code details")
				newLogger := slog.Default()
				for err, count := range errorMap {
					newLogger.With(err, count)
				}
				// newLogger.Info("error details")
			}
		}
	}()
	slog.Default().With("delay", c.wConfig.Delay, "concurrency", c.wConfig.ConcurrentRequests).Info("starting client workload")
	tc := time.NewTicker(c.wConfig.Delay)
	defer tc.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.C:
			task := pool.Submit(func() tuple {
				ctx, ca := context.WithTimeout(pool.Context(), 60*time.Second)
				defer ca()
				req = req.WithContext(ctx)
				resp, err := httpClient.Do(req)
				if err != nil {
					// slog.Default().With("err", err).Error("could not complete request")
					return tuple{
						success: false,
						code:    -1,
						err:     err,
					}
				}
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					slog.Default().With("err", resp.StatusCode).Error("request failed")
					return tuple{
						success: false,
						err:     nil,
						code:    resp.StatusCode,
					}
				}
				return tuple{
					success: true,
					code:    http.StatusOK,
					err:     nil,
				}
			})
			go func() {
				res, err := task.Wait()
				if err != nil {
					slog.Default().With("err", err).Error("unhandled task pooling error")
				}
				select {
				case results <- res:
				default:
					slog.Default().Error("task result dropped, channel full")
				}
			}()
		}
	}
}

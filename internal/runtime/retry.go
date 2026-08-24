package runtime

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

type RetryPolicy struct {
	MaxAttempts int
	InitialWait time.Duration
	MaxWait     time.Duration
	Sleep       func(context.Context, time.Duration) error
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: 4,
		InitialWait: time.Second,
		MaxWait:     8 * time.Second,
		Sleep: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (policy RetryPolicy) delay(attempt int, response *http.Response) time.Duration {
	if response != nil {
		if seconds, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	delay := policy.InitialWait << attempt
	if delay > policy.MaxWait {
		return policy.MaxWait
	}
	return delay
}

func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
}

func retryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	default:
		return false
	}
}

package soehttp

import "strings"

// IsCircuitBreakerError 判断是否是熔断错误
// 当服务触发熔断保护时返回 true
func IsCircuitBreakerError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "fallback") ||
		strings.Contains(msg, "circuit open") ||
		strings.Contains(msg, "hystrix")
}

// IsTimeoutError 判断是否是超时错误
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context deadline exceeded")
}

// IsNetworkError 判断是否是网络错误
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable") ||
		strings.Contains(msg, "connection reset")
}

// IsHystrixMaxConcurrencyError 判断是否是超过最大并发限制的错误
func IsHystrixMaxConcurrencyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "max concurrency") ||
		strings.Contains(msg, "too many requests")
}

// GetErrorType 获取错误类型描述（用于日志和监控）
func GetErrorType(err error) string {
	if err == nil {
		return "success"
	}

	if IsCircuitBreakerError(err) {
		return "circuit_breaker"
	}
	if IsTimeoutError(err) {
		return "timeout"
	}
	if IsNetworkError(err) {
		return "network"
	}
	if IsHystrixMaxConcurrencyError(err) {
		return "max_concurrency"
	}

	return "business_error"
}

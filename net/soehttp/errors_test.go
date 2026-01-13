package soehttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestErrorDetection_Comprehensive 综合测试错误检测的准确性
func TestErrorDetection_Comprehensive(t *testing.T) {
	defer ResetConfigForTest()

	tests := []struct {
		name           string
		setupServer    func() *httptest.Server
		setupRemote    func(url string) SoeRemoteService
		expectError    bool
		expectCircuit  bool
		expectTimeout  bool
		expectNetwork  bool
		expectErrorMsg string
	}{
		{
			name: "正常请求 - 无错误",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					resp := SoeGoResponseVO{Code: 200, Msg: "ok", Data: "success"}
					json.NewEncoder(w).Encode(resp)
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				return NewRemote(RemoteOption{URL: url})
			},
			expectError:   false,
			expectCircuit: false,
		},
		{
			name: "业务错误（500） - 无熔断",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("Internal Server Error"))
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				return NewRemote(RemoteOption{URL: url})
			},
			expectError:    true,
			expectCircuit:  false,
			expectTimeout:  false,
			expectNetwork:  false,
			expectErrorMsg: "Internal Server Error",
		},
		{
			name: "熔断触发 - circuit open",
			setupServer: func() *httptest.Server {
				var count int32
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					atomic.AddInt32(&count, 1)
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("error"))
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				enable := true
				return NewRemote(RemoteOption{
					URL:           url,
					EnableHystrix: &enable,
					HystrixConfig: &HystrixConfig{
						Timeout:                1000,
						MaxConcurrentRequests:  100,
						ErrorPercentThreshold:  30,
						RequestVolumeThreshold: 3,
						SleepWindow:            1000,
					},
				})
			},
			expectError:   true,
			expectCircuit: true,
		},
		{
			name: "超时错误 - 无熔断",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(3 * time.Second)
					json.NewEncoder(w).Encode(SoeGoResponseVO{Code: 200})
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				return NewRemote(RemoteOption{
					URL:           url,
					TimeoutSecond: 1, // 1秒超时
				})
			},
			expectError:   true,
			expectCircuit: false,
			expectTimeout: true,
		},
		{
			name: "Context 超时",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(2 * time.Second)
					json.NewEncoder(w).Encode(SoeGoResponseVO{Code: 200})
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				ctx, _ := context.WithTimeout(context.Background(), 500*time.Millisecond)
				return NewRemote(RemoteOption{
					URL:     url,
					Context: ctx,
				})
			},
			expectError:   true,
			expectCircuit: false,
			expectTimeout: true,
		},
		{
			name: "熔断触发 - 带 fallback",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			setupRemote: func(url string) SoeRemoteService {
				enable := true
				return NewRemote(RemoteOption{
					URL:           url,
					EnableHystrix: &enable,
					HystrixConfig: &HystrixConfig{
						Timeout:                1000,
						ErrorPercentThreshold:  50,
						RequestVolumeThreshold: 3,
						SleepWindow:            1000,
					},
				})
			},
			expectError:   true,
			expectCircuit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			defer server.Close()

			remote := tt.setupRemote(server.URL)

			// 如果期望熔断，需要先触发熔断
			if tt.expectCircuit {
				for i := 0; i < 10; i++ {
					remote.Get(nil)
					time.Sleep(50 * time.Millisecond)
				}
			}

			// 执行实际测试
			_, err := remote.Get(nil)

			// 验证是否有错误
			if tt.expectError && err == nil {
				t.Errorf("期望有错误，但没有错误")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("期望无错误，但得到错误: %v", err)
				return
			}

			if err == nil {
				return
			}

			// 验证错误类型
			isCircuit := IsCircuitBreakerError(err)
			isTimeout := IsTimeoutError(err)
			isNetwork := IsNetworkError(err)
			errType := GetErrorType(err)

			t.Logf("错误信息: %v", err.Error())
			t.Logf("IsCircuitBreakerError: %v", isCircuit)
			t.Logf("IsTimeoutError: %v", isTimeout)
			t.Logf("IsNetworkError: %v", isNetwork)
			t.Logf("GetErrorType: %v", errType)

			// 验证熔断错误检测
			if tt.expectCircuit && !isCircuit {
				t.Errorf("期望检测到熔断错误，但 IsCircuitBreakerError 返回 false")
			}
			if !tt.expectCircuit && isCircuit {
				t.Errorf("不期望熔断错误，但 IsCircuitBreakerError 返回 true")
			}

			// 验证超时错误检测
			if tt.expectTimeout && !isTimeout {
				t.Errorf("期望检测到超时错误，但 IsTimeoutError 返回 false")
			}
			if !tt.expectTimeout && isTimeout && !tt.expectCircuit {
				// 注意：熔断可能包含 timeout 字样
				t.Errorf("不期望超时错误，但 IsTimeoutError 返回 true")
			}

			// 验证错误类型
			if tt.expectCircuit && errType != "circuit_breaker" {
				t.Errorf("期望错误类型为 circuit_breaker，实际: %s", errType)
			}
			if tt.expectTimeout && !tt.expectCircuit && errType != "timeout" {
				t.Errorf("期望错误类型为 timeout，实际: %s", errType)
			}
		})
	}
}

// TestErrorMessage_Actual 测试实际的错误消息格式
func TestErrorMessage_Actual(t *testing.T) {
	defer ResetConfigForTest()

	t.Run("查看实际的熔断错误消息", func(t *testing.T) {
		var count int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&count, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		enable := true
		remote := NewRemote(RemoteOption{
			URL:           server.URL,
			EnableHystrix: &enable,
			HystrixConfig: &HystrixConfig{
				Timeout:                1000,
				ErrorPercentThreshold:  50,
				RequestVolumeThreshold: 3,
				SleepWindow:            1000,
			},
		})

		// 触发熔断
		var errors []error
		for i := 0; i < 15; i++ {
			_, err := remote.Get(nil)
			if err != nil {
				errors = append(errors, err)
			}
			time.Sleep(50 * time.Millisecond)
		}

		t.Logf("总共收集了 %d 个错误", len(errors))
		t.Logf("服务器实际处理了 %d 个请求", atomic.LoadInt32(&count))

		// 打印所有错误消息
		for i, err := range errors {
			t.Logf("错误 %d: %v", i+1, err.Error())
			t.Logf("  IsCircuitBreakerError: %v", IsCircuitBreakerError(err))
			t.Logf("  GetErrorType: %v", GetErrorType(err))
		}
	})
}

// TestErrorType_EdgeCases 测试边界情况
func TestErrorType_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		expectType string
	}{
		{
			name:       "nil 错误",
			err:        nil,
			expectType: "success",
		},
		{
			name:       "包含 fallback 的错误",
			err:        errors.New("fallback failed"),
			expectType: "circuit_breaker",
		},
		{
			name:       "包含 circuit open 的错误",
			err:        errors.New("hystrix: circuit open"),
			expectType: "circuit_breaker",
		},
		{
			name:       "包含 hystrix 的错误",
			err:        errors.New("hystrix: timeout"),
			expectType: "circuit_breaker", // hystrix 关键字优先
		},
		{
			name:       "纯 timeout 错误",
			err:        errors.New("request timeout"),
			expectType: "timeout",
		},
		{
			name:       "context deadline exceeded",
			err:        errors.New("context deadline exceeded"),
			expectType: "timeout",
		},
		{
			name:       "connection refused",
			err:        errors.New("connection refused"),
			expectType: "network",
		},
		{
			name:       "max concurrency",
			err:        errors.New("hystrix: max concurrency"),
			expectType: "circuit_breaker", // 包含 hystrix，优先识别为熔断
		},
		{
			name:       "普通业务错误",
			err:        errors.New("invalid parameter"),
			expectType: "business_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType := GetErrorType(tt.err)
			if gotType != tt.expectType {
				t.Errorf("GetErrorType() = %v, 期望 %v", gotType, tt.expectType)
			}

			// 详细日志
			if tt.err != nil {
				t.Logf("错误: %v", tt.err)
				t.Logf("IsCircuitBreaker: %v", IsCircuitBreakerError(tt.err))
				t.Logf("IsTimeout: %v", IsTimeoutError(tt.err))
				t.Logf("IsNetwork: %v", IsNetworkError(tt.err))
				t.Logf("类型: %v", gotType)
			}
		})
	}
}

// TestErrorType_Priority 测试错误类型优先级
func TestErrorType_Priority(t *testing.T) {
	t.Run("hystrix timeout 应该被识别为熔断而非超时", func(t *testing.T) {
		err := errors.New("fallback failed with 'fallback'. run error was 'hystrix: timeout'")

		isCircuit := IsCircuitBreakerError(err)
		isTimeout := IsTimeoutError(err)
		errType := GetErrorType(err)

		t.Logf("错误: %v", err)
		t.Logf("IsCircuitBreaker: %v", isCircuit)
		t.Logf("IsTimeout: %v", isTimeout)
		t.Logf("类型: %v", errType)

		if errType != "circuit_breaker" {
			t.Errorf("期望类型为 circuit_breaker，实际: %s", errType)
		}
	})
}

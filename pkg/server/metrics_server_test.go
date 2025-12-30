package server

import (
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"mmemory/pkg/logger"
)

func init() {
	// 初始化logger以避免测试中的nil指针错误
	logger.Init("info", "text", "stdout", "")
}

func TestNewMetricsServer(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{
			name: "创建指标服务器端口9090",
			port: 9090,
		},
		{
			name: "创建指标服务器端口8080",
			port: 8080,
		},
		{
			name: "创建指标服务器端口3000",
			port: 3000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewMetricsServer(tt.port)
			if server == nil {
				t.Errorf("NewMetricsServer() 返回了 nil")
			}
			if server.server == nil {
				t.Errorf("NewMetricsServer() server字段为 nil")
			}
			expectedAddr := ""
			if server.server.Addr != expectedAddr {
				// 验证地址格式正确
				if server.server.Addr == "" {
					t.Errorf("NewMetricsServer() Addr为空")
				}
			}
		})
	}
}

func TestMetricsServer_Start(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "启动指标服务器",
			port:    19090, // 使用非标准端口避免冲突
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewMetricsServer(tt.port)

			err := server.Start()
			if (err != nil) != tt.wantErr {
				t.Errorf("Start() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// 等待服务器启动
			time.Sleep(100 * time.Millisecond)

			// 测试健康检查端点
			resp, err := http.Get("http://localhost:19090/health")
			if err != nil {
				t.Errorf("健康检查请求失败: %v", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("健康检查返回状态码: %d, want %d", resp.StatusCode, http.StatusOK)
				}
			}

			// 测试就绪检查端点
			resp, err = http.Get("http://localhost:19090/ready")
			if err != nil {
				t.Errorf("就绪检查请求失败: %v", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("就绪检查返回状态码: %d, want %d", resp.StatusCode, http.StatusOK)
				}
			}

			// 测试指标端点
			resp, err = http.Get("http://localhost:19090/metrics")
			if err != nil {
				t.Errorf("指标请求失败: %v", err)
			} else {
				defer resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("指标返回状态码: %d, want %d", resp.StatusCode, http.StatusOK)
				}

				// 读取响应内容
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("读取响应体失败: %v", err)
				}
				if len(body) == 0 {
					t.Errorf("指标响应体为空")
				}
			}

			// 停止服务器
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			server.Stop(ctx)
		})
	}
}

func TestMetricsServer_Stop(t *testing.T) {
	tests := []struct {
		name    string
		port    int
		wantErr bool
	}{
		{
			name:    "停止已启动的服务器",
			port:    19091,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewMetricsServer(tt.port)

			// 启动服务器
			err := server.Start()
			if err != nil {
				t.Errorf("启动服务器失败: %v", err)
				return
			}

			// 等待服务器启动
			time.Sleep(100 * time.Millisecond)

			// 停止服务器
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err = server.Stop(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Stop() error = %v, wantErr %v", err, tt.wantErr)
			}

			// 等待服务器完全停止
			time.Sleep(100 * time.Millisecond)

			// 验证服务器已停止
			_, err = http.Get("http://localhost:19091/health")
			if err == nil {
				t.Errorf("服务器未成功停止")
			}
		})
	}
}

func TestMetricsServer_Endpoints(t *testing.T) {
	server := NewMetricsServer(19092)

	// 启动服务器
	err := server.Start()
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	testCases := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "健康检查端点",
			path:           "/health",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "就绪检查端点",
			path:           "/ready",
			expectedStatus: http.StatusOK,
			expectedBody:   "Ready",
		},
		{
			name:           "指标端点",
			path:           "/metrics",
			expectedStatus: http.StatusOK,
			expectedBody:   "",
		},
		{
			name:           "不存在的端点",
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get("http://localhost:19092" + tc.path)
			if err != nil {
				t.Errorf("请求失败: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("状态码不匹配: got %d, want %d", resp.StatusCode, tc.expectedStatus)
			}

			if tc.expectedBody != "" {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Errorf("读取响应体失败: %v", err)
					return
				}
				bodyStr := string(body)
				if bodyStr != tc.expectedBody {
					t.Errorf("响应体不匹配: got %s, want %s", bodyStr, tc.expectedBody)
				}
			}
		})
	}
}

func TestMetricsServer_ConcurrentRequests(t *testing.T) {
	server := NewMetricsServer(19093)

	// 启动服务器
	err := server.Start()
	if err != nil {
		t.Fatalf("启动服务器失败: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Stop(ctx)
	}()

	// 等待服务器启动
	time.Sleep(100 * time.Millisecond)

	// 并发发送多个请求
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			resp, err := http.Get("http://localhost:19093/health")
			if err != nil {
				t.Errorf("并发请求失败: %v", err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Errorf("并发请求状态码错误: %d", resp.StatusCode)
				}
			}
			done <- true
		}()
	}

	// 等待所有请求完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestMetricsServer_Timeouts(t *testing.T) {
	server := NewMetricsServer(19094)

	// 验证超时配置
	if server.server.ReadTimeout != 10*time.Second {
		t.Errorf("ReadTimeout配置错误: got %v, want %v", server.server.ReadTimeout, 10*time.Second)
	}
	if server.server.WriteTimeout != 10*time.Second {
		t.Errorf("WriteTimeout配置错误: got %v, want %v", server.server.WriteTimeout, 10*time.Second)
	}
	if server.server.IdleTimeout != 60*time.Second {
		t.Errorf("IdleTimeout配置错误: got %v, want %v", server.server.IdleTimeout, 60*time.Second)
	}
}

func TestMetricsServer_StopWithoutStart(t *testing.T) {
	server := NewMetricsServer(19095)

	// 停止未启动的服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := server.Stop(ctx)
	// 不应该报错
	if err != nil {
		t.Errorf("停止未启动的服务器不应报错: %v", err)
	}
}

func TestMetricsServer_MultipleStartStop(t *testing.T) {
	server := NewMetricsServer(19096)

	// 多次启动和停止
	for i := 0; i < 3; i++ {
		err := server.Start()
		if err != nil {
			t.Errorf("第%d次启动失败: %v", i+1, err)
		}

		time.Sleep(50 * time.Millisecond)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = server.Stop(ctx)
		cancel()
		if err != nil {
			t.Errorf("第%d次停止失败: %v", i+1, err)
		}

		time.Sleep(50 * time.Millisecond)
	}
}
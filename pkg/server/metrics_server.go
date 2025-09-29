package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"mmemory/pkg/logger"
)

// MetricsServer 指标服务器
type MetricsServer struct {
	server *http.Server
}

// NewMetricsServer 创建指标服务器
func NewMetricsServer(port int) *MetricsServer {
	mux := http.NewServeMux()
	
	// 注册Prometheus指标处理器
	mux.Handle("/metrics", promhttp.Handler())
	
	// 添加健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	// 添加就绪检查端点
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Ready"))
	})
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return &MetricsServer{
		server: server,
	}
}

// Start 启动指标服务器
func (s *MetricsServer) Start() error {
	logger.Infof("📊 指标服务器启动，端口: %s", s.server.Addr)
	
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Errorf("指标服务器启动失败: %v", err)
		}
	}()
	
	return nil
}

// Stop 停止指标服务器
func (s *MetricsServer) Stop(ctx context.Context) error {
	logger.Info("📊 指标服务器停止")
	
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	
	return s.server.Shutdown(shutdownCtx)
}
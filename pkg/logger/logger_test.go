package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetLogger(t *testing.T) {
	tests := []struct {
		name string
		want *logrus.Logger
	}{
		{
			name: "获取logger实例",
			want: nil, // 我们只检查返回值不为nil
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetLogger()
			if got == nil {
				t.Errorf("GetLogger() 返回了 nil")
			}
		})
	}
}

func TestInit(t *testing.T) {
	// 保存原始logger
	originalLogger := Logger

	tests := []struct {
		name      string
		level     string
		format    string
		output    string
		filePath  string
		wantErr   bool
	}{
		{
			name:    "初始化JSON格式日志",
			level:   "info",
			format:  "json",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "初始化Text格式日志",
			level:   "debug",
			format:  "text",
			output:  "stdout",
			wantErr: false,
		},
		{
			name:    "初始化无效日志级别",
			level:   "invalid",
			format:  "text",
			output:  "stdout",
			wantErr: false, // 应该使用默认级别
		},
		{
			name:     "初始化文件输出",
			level:    "info",
			format:   "text",
			output:   "file",
			filePath: "/tmp/test_mmemory.log",
			wantErr:  false,
		},
		{
			name:     "文件输出无效路径",
			level:    "info",
			format:   "text",
			output:   "file",
			filePath: "/invalid/path/test.log",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.level, tt.format, tt.output, tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && Logger == nil {
				t.Errorf("Init() 后 Logger 为 nil")
			}
		})
	}

	// 清理测试文件
	os.Remove("/tmp/test_mmemory.log")

	// 恢复原始logger
	Logger = originalLogger
}

func TestDebug(t *testing.T) {
	// 初始化logger
	Init("debug", "text", "stdout", "")

	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "Debug日志",
			args: []interface{}{"测试", "debug", "日志"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Debug(tt.args...)
		})
	}
}

func TestDebugf(t *testing.T) {
	Init("debug", "text", "stdout", "")

	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Debugf格式化日志",
			format: "测试 %s %d",
			args:   []interface{}{"debug", 123},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Debugf(tt.format, tt.args...)
		})
	}
}

func TestInfo(t *testing.T) {
	Init("info", "text", "stdout", "")

	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "Info日志",
			args: []interface{}{"测试", "info", "日志"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Info(tt.args...)
		})
	}
}

func TestInfof(t *testing.T) {
	Init("info", "text", "stdout", "")

	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Infof格式化日志",
			format: "测试 %s %d",
			args:   []interface{}{"info", 456},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Infof(tt.format, tt.args...)
		})
	}
}

func TestWarn(t *testing.T) {
	Init("warn", "text", "stdout", "")

	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "Warn日志",
			args: []interface{}{"测试", "warn", "日志"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Warn(tt.args...)
		})
	}
}

func TestWarnf(t *testing.T) {
	Init("warn", "text", "stdout", "")

	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Warnf格式化日志",
			format: "测试 %s %d",
			args:   []interface{}{"warn", 789},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Warnf(tt.format, tt.args...)
		})
	}
}

func TestError(t *testing.T) {
	Init("error", "text", "stdout", "")

	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "Error日志",
			args: []interface{}{"测试", "error", "日志"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Error(tt.args...)
		})
	}
}

func TestErrorf(t *testing.T) {
	Init("error", "text", "stdout", "")

	tests := []struct {
		name   string
		format string
		args   []interface{}
	}{
		{
			name:   "Errorf格式化日志",
			format: "测试 %s %d",
			args:   []interface{}{"error", 999},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Errorf(tt.format, tt.args...)
		})
	}
}

func TestLoggerOutput(t *testing.T) {
	// 测试日志输出到buffer
	var buf bytes.Buffer
	Init("info", "text", "stdout", "")
	Logger.SetOutput(&buf)

	Infof("测试消息 %s", "内容")
	output := buf.String()

	if !strings.Contains(output, "测试消息") {
		t.Errorf("日志输出不包含预期内容: %s", output)
	}

	// 测试JSON格式
	buf.Reset()
	Init("info", "json", "stdout", "")
	Logger.SetOutput(&buf)

	Infof("测试JSON消息")
	output = buf.String()

	if !strings.Contains(output, "\"level\":\"info\"") {
		t.Errorf("JSON格式日志不包含level字段: %s", output)
	}
}

func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer

	// 测试debug级别
	Init("debug", "text", "stdout", "")
	Logger.SetOutput(&buf)
	buf.Reset()

	Debug("debug消息")
	Info("info消息")
	Warn("warn消息")
	Error("error消息")

	output := buf.String()
	if !strings.Contains(output, "debug消息") {
		t.Errorf("debug级别日志未输出")
	}

	// 测试info级别（不输出debug）
	Init("info", "text", "stdout", "")
	Logger.SetOutput(&buf)
	buf.Reset()

	Debug("debug消息")
	Info("info消息")
	output = buf.String()

	if strings.Contains(output, "debug消息") {
		t.Errorf("info级别不应输出debug日志")
	}
	if !strings.Contains(output, "info消息") {
		t.Errorf("info级别日志未输出")
	}
}

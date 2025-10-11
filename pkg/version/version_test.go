package version

import (
	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	if info.Version == "" {
		t.Error("Version should not be empty")
	}

	if info.GoVersion == "" {
		t.Error("GoVersion should not be empty")
	}

	if info.Platform == "" {
		t.Error("Platform should not be empty")
	}

	// Platform 格式应该是 "os/arch"
	if !strings.Contains(info.Platform, "/") {
		t.Errorf("Platform format should be 'os/arch', got: %s", info.Platform)
	}
}

func TestGetVersionString(t *testing.T) {
	// 测试默认情况
	Version = "v1.0.0"
	GitCommit = "unknown"
	versionStr := GetVersionString()
	if versionStr != "v1.0.0" {
		t.Errorf("Expected 'v1.0.0', got: %s", versionStr)
	}

	// 测试有Git提交的情况
	GitCommit = "abc123def456"
	versionStr = GetVersionString()
	expected := "v1.0.0-abc123d"
	if versionStr != expected {
		t.Errorf("Expected '%s', got: %s", expected, versionStr)
	}

	// 测试短Git提交
	GitCommit = "abc"
	versionStr = GetVersionString()
	if versionStr != "v1.0.0" {
		t.Errorf("Short commit should fallback to version only, got: %s", versionStr)
	}
}

func TestGetFullVersionString(t *testing.T) {
	Version = "v1.0.0"
	GitCommit = "abc123"
	GitBranch = "main"
	BuildTime = "2024-01-01T00:00:00Z"

	fullVersion := GetFullVersionString()

	// 检查是否包含关键信息
	if !strings.Contains(fullVersion, Version) {
		t.Error("Full version should contain version")
	}

	if !strings.Contains(fullVersion, GitCommit) {
		t.Error("Full version should contain git commit")
	}

	if !strings.Contains(fullVersion, GitBranch) {
		t.Error("Full version should contain git branch")
	}

	if !strings.Contains(fullVersion, "MMemory") {
		t.Error("Full version should contain app name")
	}
}

func TestFormatBuildTime(t *testing.T) {
	tests := []struct {
		name      string
		buildTime string
		wantEmpty bool
	}{
		{
			name:      "unknown build time",
			buildTime: "unknown",
			wantEmpty: false,
		},
		{
			name:      "valid RFC3339 time",
			buildTime: "2024-01-01T12:00:00Z",
			wantEmpty: false,
		},
		{
			name:      "invalid time format",
			buildTime: "invalid-time",
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			BuildTime = tt.buildTime
			result := FormatBuildTime()

			if result == "" && !tt.wantEmpty {
				t.Error("FormatBuildTime should not return empty string")
			}
		})
	}
}

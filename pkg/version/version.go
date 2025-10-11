package version

import (
	"fmt"
	"runtime"
	"time"
)

var (
	// Version 应用版本号，由构建时注入
	Version = "v0.4.0-dev"

	// GitCommit Git提交哈希，由构建时注入
	GitCommit = "unknown"

	// GitBranch Git分支名，由构建时注入
	GitBranch = "unknown"

	// BuildTime 构建时间，由构建时注入
	BuildTime = "unknown"

	// GoVersion Go版本
	GoVersion = runtime.Version()

	// Platform 运行平台
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// Info 版本信息结构
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetInfo 获取版本信息
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// GetVersionString 获取简短版本字符串
func GetVersionString() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}

// GetFullVersionString 获取完整版本字符串
func GetFullVersionString() string {
	info := GetInfo()
	return fmt.Sprintf(
		"MMemory %s\n"+
			"Git Commit: %s\n"+
			"Git Branch: %s\n"+
			"Build Time: %s\n"+
			"Go Version: %s\n"+
			"Platform: %s",
		info.Version,
		info.GitCommit,
		info.GitBranch,
		info.BuildTime,
		info.GoVersion,
		info.Platform,
	)
}

// FormatBuildTime 格式化构建时间
func FormatBuildTime() string {
	if BuildTime == "unknown" {
		return "unknown"
	}

	// 尝试解析构建时间
	t, err := time.Parse(time.RFC3339, BuildTime)
	if err != nil {
		return BuildTime
	}

	// 转换为北京时间并格式化
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return t.In(loc).Format("2006-01-02 15:04:05 MST")
}

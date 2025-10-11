# 版本信息管理

MMemory 项目实现了完整的版本信息管理系统，支持在编译时注入版本元数据，并通过命令和 API 查询版本信息。

## 功能特性

### 1. 版本包 (`pkg/version`)

提供版本信息管理功能：

- **版本号**: 语义化版本号（如 v0.3.0-dev）
- **Git信息**: 提交哈希、分支名
- **构建信息**: 构建时间、Go版本、运行平台

### 2. 构建时注入

通过 Makefile 在编译时自动注入版本信息：

```bash
# 使用默认版本构建
make build

# 指定版本号构建
make build VERSION=v1.0.0

# 查看版本信息
make version
```

版本信息通过 Go 的 `-ldflags` 注入：

```makefile
LDFLAGS=-X 'mmemory/pkg/version.Version=$(VERSION)' \
        -X 'mmemory/pkg/version.GitCommit=$(GIT_COMMIT)' \
        -X 'mmemory/pkg/version.GitBranch=$(GIT_BRANCH)' \
        -X 'mmemory/pkg/version.BuildTime=$(BUILD_TIME)'
```

### 3. Bot 命令

用户可以通过 Telegram Bot 查询版本信息：

```
/version
```

返回格式化的版本信息：

```
ℹ️ MMemory 版本信息

版本: v0.3.0-dev
Git提交: 3a1986d
Git分支: master
构建时间: 2025-10-11 16:58:40 CST
Go版本: go1.23.2
运行平台: darwin/arm64

🚀 MMemory - 你的智能提醒助手
```

### 4. 启动日志

程序启动时自动打印版本信息：

```
🚀 启动 MMemory v0.3.0-dev-3a1986d
📦 版本详情: Git=3a1986d, Branch=master, BuildTime=2025-10-11 16:58:40 CST
🖥️  运行环境: darwin/arm64 (go1.23.2)
```

## API 使用

### 获取版本信息

```go
import "mmemory/pkg/version"

// 获取完整版本信息结构
info := version.GetInfo()
fmt.Printf("Version: %s\n", info.Version)
fmt.Printf("Git Commit: %s\n", info.GitCommit)
fmt.Printf("Build Time: %s\n", info.BuildTime)

// 获取简短版本字符串
shortVersion := version.GetVersionString() // "v0.3.0-dev-3a1986d"

// 获取完整版本字符串（多行）
fullVersion := version.GetFullVersionString()

// 格式化构建时间（转换为北京时间）
buildTime := version.FormatBuildTime()
```

## 版本号规范

遵循语义化版本规范 (Semantic Versioning)：

- **主版本号 (Major)**: 不兼容的 API 变更
- **次版本号 (Minor)**: 向后兼容的功能新增
- **修订号 (Patch)**: 向后兼容的问题修复
- **预发布标识**: `-dev`, `-alpha`, `-beta`, `-rc1` 等

示例：
- `v0.3.0-dev`: 开发版本
- `v1.0.0`: 正式版本
- `v1.2.3-beta.1`: Beta 版本

## 构建示例

### 开发构建
```bash
make build
# 输出: bin/mmemory (包含 Git 信息)
```

### 发布构建
```bash
make build VERSION=v1.0.0
# 输出: bin/mmemory (版本号 v1.0.0)
```

### Docker 构建
Docker 镜像构建时也会自动注入版本信息：

```bash
make docker-build
```

Dockerfile 中使用构建参数：

```dockerfile
ARG VERSION=v0.3.0-dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=1 go build \
    -ldflags "-X mmemory/pkg/version.Version=${VERSION} \
              -X mmemory/pkg/version.GitCommit=${GIT_COMMIT} \
              -X mmemory/pkg/version.BuildTime=${BUILD_TIME}" \
    -o /app/mmemory ./cmd/bot
```

## CI/CD 集成

在 CI/CD 流程中自动设置版本号：

```bash
# GitHub Actions 示例
export VERSION=$(git describe --tags --always)
export GIT_COMMIT=$(git rev-parse --short HEAD)
export GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

make build VERSION=$VERSION
```

## 测试

运行版本包测试：

```bash
go test -v ./pkg/version/
```

测试覆盖：
- ✅ 版本信息获取
- ✅ 版本字符串格式化
- ✅ 构建时间格式化
- ✅ 边界情况处理

## 最佳实践

1. **开发环境**: 使用 `-dev` 后缀标识开发版本
2. **发布前**: 更新 Makefile 中的 `VERSION` 默认值
3. **Git 标签**: 为每个发布版本创建 Git 标签
4. **版本一致性**: 确保 `pkg/version/version.go` 中的默认版本与 Makefile 一致

## 相关文件

- `pkg/version/version.go`: 版本包实现
- `pkg/version/version_test.go`: 单元测试
- `internal/bot/handlers/message.go`: `/version` 命令实现
- `cmd/bot/main.go`: 启动日志集成
- `Makefile`: 构建脚本（版本注入）

## C3 阶段集成

版本管理功能已集成到 C3 阶段的功能中：

- ✅ Bot 命令: `/version`
- ✅ 启动日志显示完整版本信息
- ✅ 构建时自动注入 Git 元数据
- ✅ 单元测试覆盖

版本: v0.4.0-dev（C3+ 阶段开发版本 - 包含版本管理功能）

# 使用更新的Go镜像作为构建环境
FROM golang:1.21 AS builder

# 设置工作目录
WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-w -s' -o mmemory ./cmd/bot

# 使用兼容性更好的镜像运行应用（解决Alpine兼容性问题）
FROM debian:bookworm-slim

# 安装必要的运行时依赖
RUN apt-get update && apt-get install -y \
    ca-certificates \
    tzdata \
    && rm -rf /var/lib/apt/lists/*

# 设置时区为北京时间
ENV TZ=Asia/Shanghai

	# 创建非root用户
	RUN groupadd -r mmemory && useradd -r -g mmemory mmemory

# 创建工作目录和数据目录
WORKDIR /app
RUN mkdir -p /app/data && chown -R mmemory:mmemory /app

# 复制构建的二进制文件
COPY --from=builder /app/mmemory .

# 复制配置文件
COPY --from=builder /app/configs ./configs

# 设置文件所有者
RUN chown -R mmemory:mmemory /app

# 切换到非root用户
USER mmemory

# 暴露端口（如果需要健康检查接口）
EXPOSE 8080

# 设置环境变量
ENV ENVIRONMENT=production
ENV DATABASE_PATH=/app/data/mmemory.db

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD ps aux | grep mmemory | grep -v grep || exit 1

# 启动应用
CMD ["./mmemory"]
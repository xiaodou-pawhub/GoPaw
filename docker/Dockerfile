# GoPaw Runtime Image
#
# 使用预编译二进制构建镜像，不包含源码。
#
# 构建步骤：
#   1. 交叉编译 Linux 二进制：make build-linux
#   2. 构建 Docker 镜像：     docker build -t gopaw:latest .
#
# 或一步完成：make docker-build

FROM alpine:3.19

# ca-certificates: HTTPS 请求; tzdata: 时区支持
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 将本地预编译的 Linux 二进制复制进镜像
# 确保已先执行 make build-linux 生成 linux/amd64 的 gopaw 文件
COPY gopaw .
RUN chmod +x gopaw

# 运行时数据目录（config 通过 volume 挂载，不放进镜像）
RUN mkdir -p data logs

EXPOSE 8088

# data: SQLite 数据库、记忆文件等持久化数据
# logs: 应用日志
VOLUME ["/app/data", "/app/logs"]

CMD ["./gopaw", "start", "--config", "/app/config.yaml"]

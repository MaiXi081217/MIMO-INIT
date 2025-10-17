#!/bin/bash
# 自动打包资源、生成 SHA256、复制到内部目录并编译 Go 可执行文件
# 输出二进制为 bin/mimo-update

set -e

# -------------------------------
# 配置
# -------------------------------
SRC_DIR="file"
CONFIG_FILE="config.json"
OUT_DIR="resources"
OUT_FILE="$OUT_DIR/resources.tar.gz"
HASH_FILE="$OUT_DIR/resources.sha256"
DECOMPRESS_DIR="internal/decompress"
DECOMPRESS_TAR="$DECOMPRESS_DIR/resources.tar.gz"
DECOMPRESS_HASH="$DECOMPRESS_DIR/resources.sha256"

# -------------------------------
# 创建输出目录
# -------------------------------
mkdir -p "$OUT_DIR"
mkdir -p "$DECOMPRESS_DIR"
mkdir -p "bin"

# -------------------------------
# 校验源文件
# -------------------------------
if [ ! -d "$SRC_DIR" ]; then
    echo "源目录 $SRC_DIR 不存在！"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo "配置文件 $CONFIG_FILE 不存在！"
    exit 1
fi

# -------------------------------
# 打包资源文件
# -------------------------------
echo "正在打包资源文件..."
tar -czf "$OUT_FILE" "$SRC_DIR" "$CONFIG_FILE"

# -------------------------------
# 生成 SHA256 校验文件
# -------------------------------
echo "生成 SHA256 校验文件..."
HASH=$(sha256sum "$OUT_FILE" | awk '{print $1}')
echo "$HASH" > "$HASH_FILE"

# -------------------------------
# 复制到 internal/decompress 供 Go 嵌入
# -------------------------------
echo "复制资源到 $DECOMPRESS_DIR 供 Go 嵌入..."
cp "$OUT_FILE" "$DECOMPRESS_TAR"
cp "$HASH_FILE" "$DECOMPRESS_HASH"

# -------------------------------
# 编译 Go 程序
# -------------------------------
echo "开始编译 Go 程序..."
if ! go build -o bin/mimo-update main.go; then
    echo "出错：Go 编译失败！"
    exit 1
fi

echo "打包与编译完成！"
echo "资源压缩包    : $OUT_FILE"
echo "哈希文件      : $HASH_FILE"
echo "Go 可执行文件 : bin/mimo-update"

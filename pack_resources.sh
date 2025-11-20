#!/bin/bash
# 自动打包资源、生成 SHA256、复制到内部目录并编译 Go 可执行文件
# 输出二进制为 bin/mimo-update-<version>

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
    echo " 源目录 $SRC_DIR 不存在！"
    exit 1
fi

if [ ! -f "$CONFIG_FILE" ]; then
    echo " 配置文件 $CONFIG_FILE 不存在！"
    exit 1
fi

# -------------------------------
# 读取版本号
# -------------------------------
VERSION_FILE="$SRC_DIR/SPDK_for_MIMO/VERSION.json"
if [ -f "$VERSION_FILE" ]; then
    # 尝试使用 jq 提取 MIMO 字段作为版本号
    if command -v jq >/dev/null 2>&1; then
        VERSION=$(jq -r '.MIMO' "$VERSION_FILE" 2>/dev/null)
    else
        # 如果没有 jq，使用 grep 和 sed 提取版本号
        VERSION=$(grep -o '"MIMO"[[:space:]]*:[[:space:]]*"[^"]*"' "$VERSION_FILE" | sed -n 's/.*"MIMO"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -1)
    fi
    if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
        echo " 未找到 MIMO 版本号，使用默认 v0.0.0"
        VERSION="v0.0.0"
    else
        echo " 读取到版本号：$VERSION"
    fi
else
    echo " 未找到版本文件 $VERSION_FILE，使用默认版本号 v0.0.0"
    VERSION="v0.0.0"
fi


# -------------------------------
# 打包资源文件
# -------------------------------
echo " 正在打包资源文件..."
tar -chzf "$OUT_FILE" "$SRC_DIR" "$CONFIG_FILE"
echo " 打包完成：$OUT_FILE"

# -------------------------------
# 生成 SHA256 校验文件
# -------------------------------
echo " 生成 SHA256 校验文件..."
HASH=$(sha256sum "$OUT_FILE" | awk '{print $1}')
echo "$HASH" > "$HASH_FILE"
echo " SHA256 校验完成：$HASH_FILE"
echo " 校验值：$HASH"

# -------------------------------
# 复制到 internal/decompress 供 Go 嵌入
# -------------------------------
echo " 复制资源到 $DECOMPRESS_DIR 嵌入..."
cp "$OUT_FILE" "$DECOMPRESS_TAR"
cp "$HASH_FILE" "$DECOMPRESS_HASH"
echo " 复制完成"

# -------------------------------
# 编译 Go 程序（带版本号后缀）
# -------------------------------
OUTPUT_BIN="bin/mimo${VERSION}"
echo " 开始编译 Go 程序..."
if go build -o "$OUTPUT_BIN" main.go; then
    echo " Go 编译完成：$OUTPUT_BIN"
else
    echo " Go 编译失败！"
    exit 1
fi
# -------------------------------
# 打印最终结果
# -------------------------------
echo "打包与编译完成！"
echo "资源压缩包    : $OUT_FILE"
echo "哈希文件      : $HASH_FILE"
echo "Go 可执行文件 : $OUTPUT_BIN"
echo "版本号        : $VERSION"

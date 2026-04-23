#!/bin/bash
set -euo pipefail

# ============================================================
#  FrontLeaves Sync — 自升级脚本 (macOS / Linux)
#  这是玩家唯一需要下载的文件，放在与 .minecraft/ 同级目录下运行。
# ============================================================

SERVER_BASE="https://yggleaf.frontleaves.com/api/v1"
METADATA_URL="${SERVER_BASE}/sync/scripts/metadata"
DOWNLOAD_URL="${SERVER_BASE}/sync/download"
BINARY_NAME="frontleaves-sync"

# -------------------- 颜色定义 --------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${CYAN}${BOLD}[INFO]${NC} $*"; }
ok()    { echo -e "${GREEN}${BOLD}[OK]${NC} $*"; }
die()   { echo -e "${RED}${BOLD}[ERROR]${NC} $*"; exit 1; }

# -------------------- 检测平台 --------------------
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "${OS}" in
        darwin) OS="darwin" ;;
        linux)  OS="linux" ;;
        *)      die "不支持的操作系统: ${OS}" ;;
    esac

    case "${ARCH}" in
        x86_64 | amd64) ARCH="amd64" ;;
        arm64 | aarch64) ARCH="arm64" ;;
        *)               die "不支持的架构: ${ARCH}" ;;
    esac

    SUFFIX="${OS}-${ARCH}"
    TARGET_NAME="${BINARY_NAME}-${SUFFIX}"
}

# -------------------- 确定二进制路径（放到 .minecraft 同级的 update/ 目录） --------------------
resolve_binary() {
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

    # 从脚本目录向上查找 .minecraft/ 所在目录
    MC_PARENT=""
    for dir in "$SCRIPT_DIR" "$SCRIPT_DIR/.." "$SCRIPT_DIR/../.."; do
        if [ -d "${dir}/.minecraft" ]; then
            MC_PARENT="$(cd "$dir" && pwd)"
            break
        fi
    done

    if [ -z "$MC_PARENT" ]; then
        die "找不到 .minecraft/ 目录。
请将本脚本放在 MC 游戏目录下运行（与 .minecraft/ 同级或其子目录内）。"
    fi

    # 二进制放在 .minecraft 同级的 update/ 目录
    UPDATE_DIR="${MC_PARENT}/update"
    mkdir -p "$UPDATE_DIR"
    BINARY_PATH="${UPDATE_DIR}/${BINARY_NAME}"
}

# -------------------- 获取远程文件信息 --------------------
fetch_remote_info() {
    info "正在获取服务端文件列表..."

    if ! command -v curl &>/dev/null; then
        die "需要 curl 命令，请先安装。"
    fi

    RESPONSE="$(curl -fsSL "$METADATA_URL" 2>/dev/null)" \
        || die "无法连接服务端，请检查网络连接。"

    REMOTE_PATH=""
    REMOTE_HASH=""

    if command -v jq &>/dev/null; then
        local count
        count="$(echo "$RESPONSE" | jq -r '.data.total // 0')"
        for i in $(seq 0 $((count - 1))); do
            local name
            name="$(echo "$RESPONSE" | jq -r ".data.files[$i].name")"
            if [ "$name" = "$TARGET_NAME" ]; then
                REMOTE_PATH="$(echo "$RESPONSE" | jq -r ".data.files[$i].path")"
                REMOTE_HASH="$(echo "$RESPONSE" | jq -r ".data.files[$i].hash")"
                break
            fi
        done
    else
        # 无 jq 时逐行解析
        local idx=0
        local names=()
        local paths=()
        local hashes=()

        while IFS= read -r line; do
            name_val="$(echo "$line" | sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
            [ -n "$name_val" ] && names+=("$name_val")
        done <<< "$(echo "$RESPONSE" | grep '"name"')"

        while IFS= read -r line; do
            path_val="$(echo "$line" | sed -n 's/.*"path"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
            [ -n "$path_val" ] && paths+=("$path_val")
        done <<< "$(echo "$RESPONSE" | grep '"path"')"

        while IFS= read -r line; do
            hash_val="$(echo "$line" | sed -n 's/.*"hash"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
            [ -n "$hash_val" ] && hashes+=("$hash_val")
        done <<< "$(echo "$RESPONSE" | grep '"hash"')"

        for i in "${!names[@]}"; do
            if [ "${names[$i]}" = "$TARGET_NAME" ]; then
                REMOTE_PATH="${paths[$i]:-}"
                REMOTE_HASH="${hashes[$i]:-}"
                break
            fi
        done
    fi

    if [ -z "$REMOTE_PATH" ]; then
        die "服务端未找到 ${TARGET_NAME}，该平台可能尚未发布。"
    fi
}

# -------------------- 计算本地 SHA-256 --------------------
local_hash() {
    if [ -f "$BINARY_PATH" ] && command -v shasum &>/dev/null; then
        echo "sha256:$(shasum -a 256 "$BINARY_PATH" | awk '{print $1}')"
    else
        echo ""
    fi
}

# -------------------- 下载二进制 --------------------
download_binary() {
    local encoded_path
    encoded_path="$(printf '%s' "$REMOTE_PATH" | sed 's/\//%2F/g')"
    local url="${DOWNLOAD_URL}?path=${encoded_path}"
    local tmp_file

    tmp_file="$(mktemp "${BINARY_PATH}.XXXXXX")"

    info "正在下载 ${TARGET_NAME}..."
    info "下载地址: ${url}"

    if ! curl -fSL --progress-bar -o "$tmp_file" "$url"; then
        rm -f "$tmp_file"
        die "下载失败，请检查网络连接。"
    fi

    chmod +x "$tmp_file"

    # 哈希校验
    if [ -n "$REMOTE_HASH" ] && command -v shasum &>/dev/null; then
        local actual_hex
        actual_hex="$(shasum -a 256 "$tmp_file" | awk '{print $1}')"
        local expected_hex="${REMOTE_HASH#sha256:}"

        if [ "$actual_hex" != "$expected_hex" ]; then
            rm -f "$tmp_file"
            die "哈希校验失败！期望 ${REMOTE_HASH}，实际 sha256:${actual_hex}"
        fi
        ok "哈希校验通过"
    fi

    # 原子替换
    if [ -f "$BINARY_PATH" ]; then
        local backup="${BINARY_PATH}.bak"
        cp -f "$BINARY_PATH" "$backup" 2>/dev/null || true
        if ! mv -f "$tmp_file" "$BINARY_PATH"; then
            [ -f "$backup" ] && mv -f "$backup" "$BINARY_PATH"
            rm -f "$tmp_file"
            die "替换文件失败，请尝试: sudo bash $0"
        fi
        rm -f "$backup"
    else
        if ! mv -f "$tmp_file" "$BINARY_PATH"; then
            rm -f "$tmp_file"
            die "写入文件失败，请尝试: sudo bash $0"
        fi
    fi

    ok "下载完成: ${BINARY_PATH}"
}

# -------------------- 主流程 --------------------
main() {
    echo ""
    echo -e "${CYAN}${BOLD}  FrontLeaves Sync 自升级工具${NC}"
    echo ""

    detect_platform
    resolve_binary
    fetch_remote_info

    info "平台: ${SUFFIX}"

    local cur_hash
    cur_hash="$(local_hash)"

    if [ -f "$BINARY_PATH" ] && [ "$cur_hash" = "$REMOTE_HASH" ]; then
        ok "已是最新版本，无需更新。"
    else
        if [ -f "$BINARY_PATH" ]; then
            info "发现新版本，正在更新..."
        else
            info "首次运行，正在下载同步器..."
        fi
        echo ""
        download_binary
    fi

    echo ""
    ok "启动同步器..."
    echo ""
    exec "$BINARY_PATH"
}

main "$@"

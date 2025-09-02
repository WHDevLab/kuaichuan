#!/bin/bash
export PATH=$PATH:~/go/bin
# Golang Gin+Fyne应用Windows交叉编译脚本
# 使用方法: ./build_windows.sh -i /path/to/icon.png -n app_name
rm release/kuaichuan_windows_amd64.zip
# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # 无颜色

# 默认参数
ICON_PATH="/Users/apple/Downloads/logo-icon-white-bg.png"
APP_NAME="release/kuaichuan"
ARCH="amd64"

# 解析命令行参数
while getopts "i:n:a:h" opt; do
  case $opt in
    i) ICON_PATH="$OPTARG";;
    n) APP_NAME="$OPTARG";;
    a) ARCH="$OPTARG";;
    h)
      echo "用法: $0 -i /path/to/icon.png -n app_name -a [amd64|386]"
      exit 0
      ;;
    \?)
      echo "无效选项: -$OPTARG" >&2
      exit 1
      ;;
  esac
done

# 检查是否提供了图标路径
if [ -z "$ICON_PATH" ]; then
  echo -e "${RED}错误: 必须提供图标路径 (-i 参数)${NC}"
  exit 1
fi

# 检查图标文件是否存在
if [ ! -f "$ICON_PATH" ]; then
  echo -e "${RED}错误: 图标文件 '$ICON_PATH' 不存在${NC}"
  exit 1
fi

# 检查系统类型
OS=$(uname -s)
case $OS in
  Darwin)
    echo -e "${YELLOW}检测到macOS系统${NC}"
    # 检查brew是否安装
    if ! command -v brew &> /dev/null; then
      echo -e "${RED}错误: Homebrew未安装，请先安装Homebrew: https://brew.sh${NC}"
      exit 1
    fi
    ;;
  Linux)
    echo -e "${YELLOW}检测到Linux系统${NC}"
    # 检查apt是否安装
    if ! command -v apt &> /dev/null; then
      echo -e "${RED}错误: 仅支持基于Debian的系统 (如Ubuntu)${NC}"
      exit 1
    fi
    ;;
  *)
    echo -e "${RED}错误: 不支持的系统类型: $OS${NC}"
    exit 1
    ;;
esac

# 安装必要的依赖
echo -e "${YELLOW}正在安装必要的依赖...${NC}"
case $OS in
  Darwin)
    brew install mingw-w64 || true
    ;;
  Linux)
    sudo apt-get update
    sudo apt-get install -y mingw-w64 || true
    ;;
esac

# 安装Fyne工具
echo -e "${YELLOW}正在安装Fyne工具...${NC}"
go install fyne.io/tools/cmd/fyne@latest || true

# 设置环境变量
echo -e "${YELLOW}正在设置编译环境...${NC}"
export GOOS=windows
export GOARCH=$ARCH
export CGO_ENABLED=1

if [ "$ARCH" = "amd64" ]; then
  export CC=x86_64-w64-mingw32-gcc
  export CXX=x86_64-w64-mingw32-g++
  TARGET="windows"
else
  export CC=i686-w64-mingw32-gcc
  export CXX=i686-w64-mingw32-g++
  TARGET="windows/386"
fi

# 清理并获取依赖
echo -e "${YELLOW}正在清理并获取依赖...${NC}"
go clean
go mod tidy

# 编译应用
echo -e "${YELLOW}正在编译Windows应用...${NC}"
fyne package -os $TARGET -name $APP_NAME -icon $ICON_PATH

# 检查编译结果
if [ $? -eq 0 ]; then
  echo -e "${GREEN}编译成功!${NC}"
  echo -e "${GREEN}Windows可执行文件: ${APP_NAME}.exe${NC}"
else
  echo -e "${RED}编译失败!${NC}"
  exit 1
fi

# 可选: 创建压缩包
read -p "是否创建包含可执行文件的zip压缩包? (y/n): " create_zip
if [ "$create_zip" = "y" ]; then
  echo -e "${YELLOW}正在创建压缩包...${NC}"
  zip "${APP_NAME}_windows_${ARCH}.zip" "${APP_NAME}.exe"
  echo -e "${GREEN}压缩包已创建: ${APP_NAME}_windows_${ARCH}.zip${NC}"
fi

echo -e "${GREEN}编译过程全部完成!${NC}"
rm release/kuaichuan.exe
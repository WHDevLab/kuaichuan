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
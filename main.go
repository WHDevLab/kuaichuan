package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/skip2/go-qrcode"
)

// 应用状态
type AppState struct {
	UploadDir     binding.String
	ServerAddress binding.String
	ServerRunning binding.Bool
	StatusMessage binding.String
	Uploads       binding.UntypedList
	TotalUploads  binding.Int
	TotalSize     binding.String
	CurrentSpeed  binding.String
	Server        *AppServer
}

type MyApp struct {
}

func main() {
	// 创建应用
	a := app.New()
	// a.Settings().SetTheme(&customTheme{})
	w := a.NewWindow("快传-局域网文件共享")
	w.SetMaster()

	// 创建状态
	state := &AppState{
		UploadDir:     binding.NewString(),
		ServerAddress: binding.NewString(),
		ServerRunning: binding.NewBool(),
		StatusMessage: binding.NewString(),
		Uploads:       binding.NewUntypedList(),
		TotalUploads:  binding.NewInt(),
		TotalSize:     binding.NewString(),
		CurrentSpeed:  binding.NewString(),
	}
	// 设置默认上传目录
	// defaultDir := filepath.Join(os.Getenv("HOME"), "Uploads")
	// if runtime.GOOS == "windows" {
	// 	defaultDir = filepath.Join(os.Getenv("USERPROFILE"), "Uploads")
	// }
	// state.UploadDir.Set(defaultDir)

	// 加载配置
	loadConfig(state)

	// 创建UI
	content := createUI(w, state)
	w.SetContent(content)

	// 窗口关闭时保存配置
	w.SetOnClosed(state.Server.StopServer)

	// 显示窗口
	w.Resize(fyne.NewSize(800, 600))
	w.ShowAndRun()
}

// func showToast(message string, window fyne.Window) {
// 	// 创建 Toast 标签
// 	toast := widget.NewLabel(message)
// 	toast.Alignment = fyne.TextAlignCenter

// 	// 添加背景矩形
// 	background := canvas.NewRectangle(theme.PlaceHolderColor())
// 	background.CornerRadius = theme.Padding()

// 	// 创建容器
// 	toastContainer := container.NewMax(background, toast)
// 	toastContainer.Resize(fyne.NewSize(200, 40)) // 调整合适尺寸

// 	// 添加到 Overlay
// 	window.Canvas().Overlays().Add(toastContainer)

// 	// 3秒后自动移除
// 	time.AfterFunc(3*time.Second, func() {
// 		window.Canvas().Overlays().Remove(toastContainer)
// 	})
// }

func showToast(message string, win fyne.Window) {
	// 创建无按钮的自定义对话框
	toastContent := widget.NewLabel("")
	customDialog := dialog.NewCustom(message, "知道了", toastContent, win)

	// 显示并自动关闭
	customDialog.Show()
}
func createUI(window fyne.Window, state *AppState) fyne.CanvasObject {

	savePathLabel := widget.NewLabelWithData(state.UploadDir)

	selectDirBtn := widget.NewButton("选择共享文件夹", func() {
		// dialog.showfile
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}

			if uri != nil {
				state.UploadDir.Set(uri.Path())
				saveConfig(state)
				state.StatusMessage.Set("上传目录已更新")
				state.Server.StopServer()
			}
		}, window)
	})
	openBtn := widget.NewButton("打开", func() {
		uploadDir, _ := state.UploadDir.Get()
		if len(uploadDir) == 0 {
			showToast("请选择共享文件夹", window)
			return
		}
		openFolder(uploadDir)
	})
	// 服务器地址显示
	addressLabel := widget.NewLabelWithData(state.ServerAddress)
	// addressLabel.TextStyle = fyne.TextStyle{Bold: true}

	// 生成二维码
	qr, err := qrcode.New("http://"+state.Server.GetLocalIP()+":8000", qrcode.Medium)
	if err != nil {
		panic(err)
	}

	// 将二维码转为PNG
	var buf bytes.Buffer
	err = png.Encode(&buf, qr.Image(256))
	if err != nil {
		panic(err)
	}

	// 创建Fyne资源
	resource := fyne.NewStaticResource("qrcode.png", buf.Bytes())

	// 更新图片
	qrImage := canvas.NewImageFromResource(resource)
	qrImage.FillMode = canvas.ImageFillOriginal
	qrImage.Hide()
	// 服务器控制按钮
	serverBtn := widget.NewButton("开始共享", nil)
	c := canvas.NewText("", color.NRGBA{R: 255, G: 128, B: 0, A: 255})
	c.TextStyle = fyne.TextStyle{Bold: true}
	c.Alignment = fyne.TextAlignCenter
	n := container.NewHBox(
		widget.NewLabel("浏览器访问:"),
		addressLabel,
	)
	n.Hide()

	serverBtn.OnTapped = func() {
		uploadDir, _ := state.UploadDir.Get()
		if len(uploadDir) == 0 {
			showToast("请选择共享文件夹", window)
			return
		}
		serverRunning, _ := state.ServerRunning.Get()
		if serverRunning {
			state.Server.StopServer()
			state.ServerRunning.Set(false)
			serverBtn.SetText("开始共享")
			c.Text = ""
			qrImage.Hide()
			n.Hide()
			// state.StatusMessage.Set("服务器已停止")
		} else {

			// 启动服务器
			state.Server = NewAppServer(uploadDir)
			go state.Server.StartServer()
			state.ServerRunning.Set(true)
			serverBtn.SetText("停止共享")

			// 	// 更新服务器地址显示
			ip := state.Server.GetLocalIP()
			state.ServerAddress.Set(fmt.Sprintf("http://%s:8000", ip))
			// state.StatusMessage.Set("服务器已启动")
			c.Text = "正在共享"
			qrImage.Show()
			n.Show()

		}
	}
	// // 统计信息
	// statsPanel := container.NewGridWithColumns(3,
	// 	// createStatCard("今日上传", state.TotalUploads.StringWithFormat("%d 个文件")),
	// 	createStatCard("总上传大小", state.TotalSize),
	// 	createStatCard("上传速度", state.CurrentSpeed),
	// )

	// // 上传历史列表
	// uploadsList := widget.NewListWithData(
	// // state.Uploads,
	// // func() fyne.CanvasObject {
	// // 	return container.NewHBox(
	// // 		// widget.NewIcon(fyne.),
	// // 		widget.NewLabel("文件名"),
	// // 		widget.NewLabel("大小"),
	// // 		widget.NewLabel("上传时间"),
	// // 		widget.NewButton("下载", nil),
	// // 		widget.NewButton("删除", nil),
	// // 	)
	// // },
	// // func(i binding.DataItem, o fyne.CanvasObject) {
	// // 	hbox := o.(*fyne.Container)
	// // 	label := hbox.Objects[1].(*widget.Label)
	// // 	sizeLabel := hbox.Objects[2].(*widget.Label)
	// // 	timeLabel := hbox.Objects[3].(*widget.Label)
	// // 	downloadBtn := hbox.Objects[4].(*widget.Button)
	// // 	deleteBtn := hbox.Objects[5].(*widget.Button)

	// // 	var file FileInfo
	// // 	i.(binding.Untyped).Get(&file)

	// // 	label.SetText(file.Name)
	// // 	sizeLabel.SetText(formatFileSize(file.Size))
	// // 	timeLabel.SetText(file.UploadedAt)

	// // 	downloadBtn.OnTapped = func() {
	// // 		downloadFile(file.Name, window)
	// // 	}

	// // 	deleteBtn.OnTapped = func() {
	// // 		deleteFile(file.Name, state, window)
	// // 	}
	// // },
	// )

	container.NewPadded()

	// 主布局
	mainLayout := container.NewBorder(
		container.NewVBox(
			container.NewPadded(),
			container.NewHBox(
				widget.NewLabel("共享文件夹:"),
				savePathLabel,
				selectDirBtn,
				openBtn,
			),
			container.NewPadded(),
			serverBtn,

			container.NewCenter(n),
			qrImage,
			c,
		),
		nil,
		nil,
		nil,
		// container.NewVBox(
		// 	widget.NewLabel("上传历史"),
		// 	statsPanel,
		// ),
	)

	return mainLayout
}
func openFolder(dir string) {
	// 根据操作系统选择不同的命令打开文件夹
	var cmd *exec.Cmd
	if fyne.CurrentDevice().IsMobile() {
		// 移动设备上的实现可能需要使用特定的API
		log.Println("移动设备上打开文件夹功能可能受限")
		return
	} else {
		// 桌面系统
		switch os := runtime.GOOS; os {
		case "darwin":
			cmd = exec.Command("open", dir)
		case "windows":
			dir = filepath.Clean(dir)
			cmd = exec.Command("explorer", dir)
		case "linux":
			cmd = exec.Command("xdg-open", dir)
		default:
			log.Printf("不支持的操作系统: %s", os)
			return
		}
	}

	// 执行命令打开文件夹
	if err := cmd.Run(); err != nil {
		log.Printf("无法打开文件夹: %v", err)
	}
}

// func createStatCard(title string, value binding.String) fyne.CanvasObject {
// 	return container.NewBorder(
// 		widget.NewLabel(title),
// 		nil,
// 		nil,
// 		nil,
// 		widget.NewLabelWithData(value),
// 	)
// }

// func formatFileSize(bytes int64) string {
// 	if bytes == 0 {
// 		return "0 B"
// 	}

// 	units := []string{"B", "KB", "MB", "GB", "TB"}
// 	unitIndex := 0

// 	for bytes >= 1024 && unitIndex < len(units)-1 {
// 		bytes /= 1024
// 		unitIndex++
// 	}

// 	return fmt.Sprintf("%.2f %s", float64(bytes), units[unitIndex])
// }

// // 服务器相关功能
// var server *http.Server

// func startServer(uploadDir string, state *AppState) {
// 	// 创建上传目录
// 	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
// 		if err := os.MkdirAll(uploadDir, 0755); err != nil {
// 			log.Fatalf("无法创建上传目录: %v", err)
// 		}
// 	}

// 	// 创建文件服务器
// 	mux := http.NewServeMux()
// 	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte(`
// 			<!DOCTYPE html>
// 			<html>
// 			<head>
// 				<title>文件上传服务器</title>
// 			</head>
// 			<body>
// 				<h1>文件上传服务器</h1>
// 				<form action="/upload" method="post" enctype="multipart/form-data">
// 					<input type="file" name="file" multiple>
// 					<input type="submit" value="上传">
// 				</form>
// 			</body>
// 			</html>
// 		`))
// 	})

// 	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
// 		if r.Method != http.MethodPost {
// 			http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
// 			return
// 		}

// 		// 解析表单数据
// 		if err := r.ParseMultipartForm(32 << 20); // 32MB
// 		err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}

// 		// 获取文件
// 		file, handler, err := r.FormFile("file")
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusBadRequest)
// 			return
// 		}
// 		defer file.Close()

// 		// 安全处理文件名
// 		filename := sanitizeFilename(handler.Filename)
// 		dstPath := filepath.Join(uploadDir, filename)

// 		// 创建保存文件
// 		dst, err := os.Create(dstPath)
// 		if err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}
// 		defer dst.Close()

// 		// 复制文件内容
// 		if _, err := io.Copy(dst, file); err != nil {
// 			http.Error(w, err.Error(), http.StatusInternalServerError)
// 			return
// 		}

// 		// 更新上传历史
// 		fileInfo := FileInfo{
// 			Name:       filename,
// 			Size:       handler.Size,
// 			UploadedAt: time.Now().Format("2006-01-02 15:04:05"),
// 		}

// 		if err := updateHistory(fileInfo, uploadDir); err != nil {
// 			log.Printf("更新历史记录失败: %v", err)
// 		}

// 		// 返回成功响应
// 		w.Write([]byte("文件上传成功"))

// 		// 刷新上传列表
// 		refreshUploads(state)
// 	})

// 	// 启动服务器
// 	server = &http.Server{
// 		Addr:    ":8000",
// 		Handler: mux,
// 	}

// 	log.Println("服务器启动在 :8000")
// 	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 		log.Fatalf("服务器启动失败: %v", err)
// 	}
// }

// func stopServer() {
// 	if server != nil {
// 		log.Println("正在关闭服务器...")
// 		if err := server.Shutdown(context.Background()); err != nil {
// 			log.Printf("关闭服务器失败: %v", err)
// 		}
// 		log.Println("服务器已关闭")
// 	}
// }

// func refreshUploads(state *AppState) {
// 	uploadDir, _ := state.UploadDir.Get()
// 	history, err := readHistory(uploadDir)
// 	if err != nil {
// 		log.Printf("读取历史记录失败: %v", err)
// 		return
// 	}

// 	// 更新列表数据
// 	items := make([]interface{}, len(history))
// 	for i, file := range history {
// 		items[i] = file
// 	}

// 	state.Uploads.Set(items)

// 	// 更新统计信息
// 	state.TotalUploads.Set(len(history))

// 	var totalSize int64
// 	for _, file := range history {
// 		totalSize += file.Size
// 	}

// 	state.TotalSize.Set(formatFileSize(totalSize))
// }

// func downloadFile(filename string, window fyne.Window) {
// 	uploadDir, _ := AppState.UploadDir.Get()
// 	filePath := filepath.Join(uploadDir, filename)

// 	// 检查文件是否存在
// 	if _, err := os.Stat(filePath); os.IsNotExist(err) {
// 		dialog.ShowError(fmt.Errorf("文件不存在"), window)
// 		return
// 	}

// 	// 打开文件下载对话框
// 	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
// 		if err != nil {
// 			dialog.ShowError(err, window)
// 			return
// 		}

// 		if writer == nil {
// 			return // 用户取消了下载
// 		}

// 		defer writer.Close()

// 		// 读取源文件
// 		src, err := os.Open(filePath)
// 		if err != nil {
// 			dialog.ShowError(err, window)
// 			return
// 		}
// 		defer src.Close()

// 		// 复制文件内容
// 		if _, err := io.Copy(writer, src); err != nil {
// 			dialog.ShowError(err, window)
// 			return
// 		}

// 		dialog.ShowInformation("下载完成", "文件下载成功", window)
// 	}, window)

// 	saveDialog.SetFileName(filename)
// 	saveDialog.Show()
// }

// func deleteFile(filename string, state *AppState, window fyne.Window) {
// 	uploadDir, _ := state.UploadDir.Get()
// 	filePath := filepath.Join(uploadDir, filename)

// 	// 确认对话框
// 	dialog.ShowConfirm("确认删除", fmt.Sprintf("确定要删除文件 %s 吗?", filename), func(confirmed bool) {
// 		if confirmed {
// 			// 删除文件
// 			if err := os.Remove(filePath); err != nil {
// 				dialog.ShowError(err, window)
// 				return
// 			}

// 			// 从历史记录中移除
// 			if err := removeFromHistory(filename, uploadDir); err != nil {
// 				log.Printf("从历史记录中删除文件失败: %v", err)
// 			}

// 			// 刷新上传列表
// 			refreshUploads(state)

// 			// 显示成功消息
// 			dialog.ShowInformation("删除成功", "文件已成功删除", window)
// 		}
// 	}, window)
// }

// // 历史记录管理
// func historyFilePath(uploadDir string) string {
// 	return filepath.Join(uploadDir, "history.json")
// }

// func readHistory(uploadDir string) ([]FileInfo, error) {
// 	historyPath := historyFilePath(uploadDir)

// 	// 检查文件是否存在
// 	if _, err := os.Stat(historyPath); os.IsNotExist(err) {
// 		return []FileInfo{}, nil
// 	}

// 	// 读取文件内容
// 	data, err := os.ReadFile(historyPath)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// 解析JSON
// 	var history []FileInfo
// 	if err := json.Unmarshal(data, &history); err != nil {
// 		return nil, err
// 	}

// 	return history, nil
// }

// func updateHistory(fileInfo FileInfo, uploadDir string) error {
// 	history, err := readHistory(uploadDir)
// 	if err != nil {
// 		return err
// 	}

// 	// 添加新文件到历史记录
// 	history = append([]FileInfo{fileInfo}, history...)

// 	// 限制历史记录数量
// 	if len(history) > 100 {
// 		history = history[:100]
// 	}

// 	// 保存历史记录
// 	data, err := json.Marshal(history)
// 	if err != nil {
// 		return err
// 	}

// 	return os.WriteFile(historyFilePath(uploadDir), data, 0644)
// }

// func removeFromHistory(filename string, uploadDir string) error {
// 	history, err := readHistory(uploadDir)
// 	if err != nil {
// 		return err
// 	}

// 	// 过滤掉要删除的文件
// 	var newHistory []FileInfo
// 	for _, file := range history {
// 		if file.Name != filename {
// 			newHistory = append(newHistory, file)
// 		}
// 	}

// 	// 保存更新后的历史记录
// 	data, err := json.Marshal(newHistory)
// 	if err != nil {
// 		return err
// 	}

// 	return os.WriteFile(historyFilePath(uploadDir), data, 0644)
// }

// 配置管理
func configFilePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "file-upload-server-config.json")
	}

	return filepath.Join(configDir, "file-upload-server", "config.json")
}

func loadConfig(state *AppState) {
	configPath := configFilePath()
	print(configPath)

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("读取配置文件失败: %v", err)
		return
	}

	// 解析JSON
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return
	}

	// 应用配置
	if uploadDir, ok := config["uploadDir"].(string); ok {
		state.UploadDir.Set(uploadDir)
	}
}

func saveConfig(state *AppState) {
	configPath := configFilePath()

	// 创建配置目录
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Printf("创建配置目录失败: %v", err)
			return
		}
	}

	// 获取当前配置
	uploadDir, _ := state.UploadDir.Get()

	// 保存配置
	config := map[string]interface{}{
		"uploadDir": uploadDir,
	}

	data, err := json.Marshal(config)
	if err != nil {
		log.Printf("序列化配置失败: %v", err)
		return
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		log.Printf("保存配置文件失败: %v", err)
		return
	}
}

// // 安全处理文件名
// func sanitizeFilename(filename string) string {
// 	// 移除路径分隔符
// 	filename = strings.ReplaceAll(filename, "/", "_")
// 	filename = strings.ReplaceAll(filename, "\\", "_")

// 	// 移除其他可能的危险字符
// 	filename = strings.ReplaceAll(filename, "..", "_")
// 	filename = strings.ReplaceAll(filename, ":", "_")
// 	filename = strings.ReplaceAll(filename, "*", "_")
// 	filename = strings.ReplaceAll(filename, "?", "_")
// 	filename = strings.ReplaceAll(filename, "\"", "_")
// 	filename = strings.ReplaceAll(filename, "<", "_")
// 	filename = strings.ReplaceAll(filename, ">", "_")
// 	filename = strings.ReplaceAll(filename, "|", "_")

// 	return filename
// }

// // 自定义主题
// type customTheme struct{}

// func (c *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
// 	if name == "111" {
// 		return color.RGBA{R: 22, G: 93, B: 255, A: 255}
// 	}
// 	return fyne.CurrentApp().Settings().Theme().Color(name, variant)
// }

// func (c *customTheme) Font(style fyne.TextStyle) fyne.Resource {
// 	return fyne.CurrentApp().Settings().Theme().Font(style)
// }

// func (c *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
// 	return fyne.CurrentApp().Settings().Theme().Icon(name)
// }

// func (c *customTheme) Size(name fyne.ThemeSizeName) float32 {
// 	return fyne.CurrentApp().Settings().Theme().Size(name)
// }

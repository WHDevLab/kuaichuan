package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// 文件信息结构
type FileInfo struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	UploadedAt string `json:"uploaded_at"`
}

type FileItem struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// 配置
const (
	port = ":8000"
)

var server *http.Server

type AppServer struct {
	UploadDir string
}

func NewAppServer(uploadDir string) *AppServer {
	s := &AppServer{
		UploadDir: uploadDir,
	}
	return s
}

func (t *AppServer) StopServer() {
	if server != nil {
		log.Println("正在关闭服务器...")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Printf("关闭服务器失败: %v", err)
		}
		log.Println("服务器已关闭")
	}
}

func (t *AppServer) StartServer() {

	mux := http.NewServeMux()
	// 注册路由
	mux.HandleFunc("/", t.serveIndex)
	mux.HandleFunc("/get-ip", t.getIPHandler)
	mux.HandleFunc("/api/upload", t.uploadHandler)
	mux.HandleFunc("/download/", t.downloadHandler)
	mux.HandleFunc("/upload", t.upload)
	mux.HandleFunc("/api/files", t.getFileList)
	// mux.HandleFunc("/delete/", deleteHandler)
	// mux.HandleFunc("/delete-all", deleteAllHandler)
	// mux.HandleFunc("/history", historyHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("."))))

	// 启动服务器
	log.Printf("服务器运行在端口 %s", port)
	log.Printf("访问地址: http://%s%s", t.GetLocalIP(), port)

	server = &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}

	log.Println("服务器启动在 :8000")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// 首页处理函数
func (t *AppServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	// 写入响应内容
	if _, err := w.Write([]byte(resourceListHtml.StaticContent)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}
	// http.ServeFile(w, r, filepath.Join("static", "list.html"))
}

// 首页处理函数
func (t *AppServer) upload(w http.ResponseWriter, r *http.Request) {
	// http.ServeFile(w, r, filepath.Join("static", "upload.html"))
	// 写入响应内容
	if _, err := w.Write([]byte(resourceUploadHtml.StaticContent)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}
}

// 获取本地IP地址
func (t *AppServer) GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		// 检查IP地址是否为IPv4且不是回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// 获取IP地址的处理函数
func (t *AppServer) getIPHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"ip": t.GetLocalIP(),
	})
}

// 文件上传处理函数
func (t *AppServer) uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	// 解析表单数据
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 获取文件
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 安全处理文件名，防止路径遍历攻击
	filename := filepath.Base(handler.Filename)
	safeFilename := t.sanitizeFilename(filename)
	dstPath := filepath.Join(t.UploadDir, safeFilename)

	// 创建保存文件
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 复制文件内容
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 获取文件大小
	fileInfo, err := os.Stat(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 更新上传历史
	fileInfoItem := FileInfo{
		Name:       safeFilename,
		Size:       fileInfo.Size(),
		UploadedAt: time.Now().Format("2006-01-02 15:04:05"),
	}
	if err := t.updateHistory(fileInfoItem); err != nil {
		log.Printf("更新历史记录失败: %v", err)
		// 即使历史记录更新失败，也返回成功状态
	}

	// 返回成功响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"message": "文件上传成功",
		"name":    safeFilename,
		"code":    200,
	})
}

// 文件下载处理函数
func (t *AppServer) downloadHandler(w http.ResponseWriter, r *http.Request) {
	queryParams := r.URL.Query()

	path := queryParams.Get("path")
	// 获取文件名
	filename := path
	if filename == "" {
		http.Error(w, "缺少文件名", http.StatusBadRequest)
		return
	}

	// 安全处理文件名，防止路径遍历攻击
	// safeFilename := t.sanitizeFilename(filename)
	safeFilename, _ := url.QueryUnescape(filename)
	filePath := filepath.Join(t.UploadDir, safeFilename)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "文件不存在", http.StatusNotFound)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", safeFilename))
	w.Header().Set("Content-Type", "application/octet-stream")

	// 发送文件
	http.ServeFile(w, r, filePath)
}

// // 文件删除处理函数
// func deleteHandler(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodDelete {
// 		http.Error(w, "方法不允许", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	// 获取文件名
// 	filename := strings.TrimPrefix(r.URL.Path, "/delete/")
// 	if filename == "" {
// 		http.Error(w, "缺少文件名", http.StatusBadRequest)
// 		return
// 	}

// 	// 安全处理文件名，防止路径遍历攻击
// 	safeFilename := sanitizeFilename(filename)
// 	filePath := filepath.Join(uploadDir, safeFilename)

// 	// 检查文件是否存在
// 	if _, err := os.Stat(filePath); os.IsNotExist(err) {
// 		http.Error(w, "文件不存在", http.StatusNotFound)
// 		return
// 	}

// 	// 删除文件
// 	if err := os.Remove(filePath); err != nil {
// 		http.Error(w, "删除文件失败", http.StatusInternalServerError)
// 		return
// 	}

// 	// 更新历史记录
// 	if err := removeFromHistory(safeFilename); err != nil {
// 		log.Printf("从历史记录中删除文件失败: %v", err)
// 		// 即使历史记录更新失败，也返回成功状态
// 	}

// 	// 返回成功响应
// 	w.Header().Set("Content-Type", "application/json")
// 	json.NewEncoder(w).Encode(map[string]string{
// 		"status":  "success",
// 		"message": "文件已删除",
// 	})
// }

// 安全处理文件名，防止路径遍历攻击
func (t *AppServer) sanitizeFilename(filename string) string {
	// 移除路径分隔符
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// 移除其他可能的危险字符
	filename = strings.ReplaceAll(filename, "..", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "*", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "\"", "_")
	filename = strings.ReplaceAll(filename, "<", "_")
	filename = strings.ReplaceAll(filename, ">", "_")
	filename = strings.ReplaceAll(filename, "|", "_")

	return filename
}

// 历史记录文件路径
func (t *AppServer) historyFilePath() string {
	return filepath.Join(t.UploadDir, "history.json")
}

// 读取上传历史
func (t *AppServer) readHistory() ([]FileInfo, error) {
	// 检查历史文件是否存在
	if _, err := os.Stat(t.historyFilePath()); os.IsNotExist(err) {
		return []FileInfo{}, nil
	}

	// 读取历史文件
	data, err := os.ReadFile(t.historyFilePath())
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var history []FileInfo
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, err
	}

	return history, nil
}

// 更新上传历史
func (t *AppServer) updateHistory(fileInfo FileInfo) error {
	// 读取现有历史
	history, err := t.readHistory()
	if err != nil {
		return err
	}

	// 添加新文件信息到开头
	history = append([]FileInfo{fileInfo}, history...)

	// 限制历史记录数量为100
	if len(history) > 100 {
		history = history[:100]
	}

	// 保存历史记录
	data, err := json.Marshal(history)
	if err != nil {
		return err
	}

	return os.WriteFile(t.historyFilePath(), data, 0644)
}

func (t *AppServer) getFileList(w http.ResponseWriter, r *http.Request) {
	// var files []string

	// err := filepath.WalkDir(t.UploadDir, func(path string, d os.DirEntry, err error) error {
	// 	if err != nil {
	// 		// 处理访问错误（如权限不足）
	// 		fmt.Printf("访问路径 %s 失败: %v\n", path, err)
	// 		return nil // 忽略错误继续遍历
	// 	}
	// 	if !d.IsDir() {
	// 		files = append(files, path)
	// 	}
	// 	return nil
	// })

	// if err != nil {
	// 	fmt.Printf("遍历目录失败: %v\n", err)
	// 	return
	// }

	path := r.URL.Query().Get("path")
	var items []FileItem

	entries, err := os.ReadDir(t.UploadDir + path)
	if err != nil {
		fmt.Printf("读取目录失败: %v\n", err)
		return
	}

	for _, entry := range entries {
		itemType := "file"
		if entry.IsDir() {
			itemType = "folder"
		}

		items = append(items, FileItem{
			Name: entry.Name(),
			Type: itemType,
		})
	}

	json.NewEncoder(w).Encode(map[string]any{
		"message": "文件上传成功",
		"list":    items,
		"code":    200,
	})
}

// // 从历史记录中移除文件
// func removeFromHistory(filename string) error {
// 	// 读取现有历史
// 	history, err := readHistory()
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

// 	return os.WriteFile(historyFilePath(), data, 0644)
// }

// // 清空历史记录
// func clearHistory() error {
// 	// 保存空的历史记录
// 	data, err := json.Marshal([]FileInfo{})
// 	if err != nil {
// 		return err
// 	}

// 	return os.WriteFile(historyFilePath(), data, 0644)
// }

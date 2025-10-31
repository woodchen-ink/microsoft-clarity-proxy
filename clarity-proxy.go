package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	// 环境变量配置
	proxyDomain string // 代理域名，如 https://analytics.czl.net
	listenPort  string // 监听端口，默认 8081

	// Clarity 域名映射
	clarityHosts = map[string]string{
		"/ms/t.js":  "https://www.clarity.ms/tag/",     // 主脚本
		"/ms/j/":    "https://scripts.clarity.ms/",     // 核心脚本
		"/ms/i.gif": "https://c.clarity.ms/c.gif",      // 统计图片
		"/ms/c/":    "https://c.clarity.ms/",           // CDN 资源
		"/ms/p":     "https://k.clarity.ms/collect",    // 数据收集
	}

	// URL 替换映射（会在 init 中根据 proxyDomain 动态生成）
	urlReplacements map[string]string
)

func init() {
	// 读取环境变量
	proxyDomain = getEnv("PROXY_DOMAIN", "https://analytics.czl.net")
	listenPort = getEnv("LISTEN_PORT", "8081")

	// 确保 PROXY_DOMAIN 不以 / 结尾
	proxyDomain = strings.TrimSuffix(proxyDomain, "/")

	// 确保 LISTEN_PORT 以 : 开头
	if !strings.HasPrefix(listenPort, ":") {
		listenPort = ":" + listenPort
	}

	// 动态生成 URL 替换映射
	urlReplacements = map[string]string{
		"https://c.clarity.ms/c.gif":       proxyDomain + "/ms/i.gif",
		"https://scripts.clarity.ms/":      proxyDomain + "/ms/j/",
		"https://a.clarity.ms/collect":     proxyDomain + "/ms/p",
		"https://k.clarity.ms/collect":     proxyDomain + "/ms/p",
		"https://c.clarity.ms/":            proxyDomain + "/ms/c/",
		"\"https://k.clarity.ms/collect\"": "\"" + proxyDomain + "/ms/p\"",
		"\"https://a.clarity.ms/collect\"": "\"" + proxyDomain + "/ms/p\"",
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	http.HandleFunc("/", proxyHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("===========================================")
	log.Printf("Microsoft Clarity 代理服务器")
	log.Printf("===========================================")
	log.Printf("监听端口: %s", listenPort)
	log.Printf("代理域名: %s", proxyDomain)
	log.Printf("健康检查: %s/health", listenPort)
	log.Printf("===========================================")
	log.Printf("环境变量配置:")
	log.Printf("  PROXY_DOMAIN  - 代理域名 (默认: https://analytics.czl.net)")
	log.Printf("  LISTEN_PORT   - 监听端口 (默认: 8081)")
	log.Printf("===========================================")

	if err := http.ListenAndServe(listenPort, nil); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// 健康检查端点
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","service":"clarity-proxy","domain":"%s"}`, proxyDomain)
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	// 处理 OPTIONS 预检请求
	if r.Method == http.MethodOptions {
		handleCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 路由到对应的 Clarity 端点
	targetURL, needReplace := getTargetURL(r)
	if targetURL == "" {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// 创建代理请求
	proxyReq, err := createProxyRequest(r, targetURL)
	if err != nil {
		log.Printf("创建代理请求失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不自动跟随重定向
		},
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		log.Printf("代理请求失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("读取响应体失败: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 如果是 JavaScript 文件，替换 URL
	if needReplace && isJavaScript(resp.Header.Get("Content-Type")) {
		body = replaceURLs(body)
	}

	// 设置 CORS 头（完全控制，不使用上游的）
	handleCORS(w, r)

	// 复制其他响应头（排除 CORS 和 Content-Length 相关的）
	for key, values := range resp.Header {
		// 跳过 CORS 相关的头和 Content-Length（因为我们可能修改了内容）
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "access-control-") || lowerKey == "content-length" {
			continue
		}
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 如果是 JavaScript，确保 Content-Type 正确
	if needReplace {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}

	// 写入状态码和响应体
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

	log.Printf("%s %s -> %s [%d]", r.Method, r.URL.Path, targetURL, resp.StatusCode)
}

// 获取目标 URL
func getTargetURL(r *http.Request) (string, bool) {
	path := r.URL.Path

	// 主脚本 /ms/t.js?id=xxx
	if path == "/ms/t.js" {
		id := r.URL.Query().Get("id")
		if id == "" {
			return "", false
		}
		return clarityHosts["/ms/t.js"] + id, true
	}

	// 核心脚本 /ms/j/0.8.38/clarity.js
	if strings.HasPrefix(path, "/ms/j/") {
		scriptPath := strings.TrimPrefix(path, "/ms/j/")
		return clarityHosts["/ms/j/"] + scriptPath, true
	}

	// 统计图片 /ms/i.gif
	if path == "/ms/i.gif" {
		return clarityHosts["/ms/i.gif"], false
	}

	// CDN 资源 /ms/c/*
	if strings.HasPrefix(path, "/ms/c/") {
		cdnPath := strings.TrimPrefix(path, "/ms/c/")
		targetURL := clarityHosts["/ms/c/"] + cdnPath
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}
		return targetURL, false
	}

	// 数据收集 /ms/p
	if path == "/ms/p" {
		return clarityHosts["/ms/p"], false
	}

	return "", false
}

// 创建代理请求
func createProxyRequest(r *http.Request, targetURL string) (*http.Request, error) {
	// 读取原始请求体
	var body io.Reader
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(bodyBytes)
	}

	// 解析目标 URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	// 创建新请求
	proxyReq, err := http.NewRequest(r.Method, targetURL, body)
	if err != nil {
		return nil, err
	}

	// 复制请求头
	for key, values := range r.Header {
		// 跳过一些不需要的头
		if key == "Host" {
			continue
		}
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// 设置正确的 Host
	proxyReq.Host = parsedURL.Host
	proxyReq.Header.Set("Host", parsedURL.Host)

	// 设置真实 IP
	if clientIP := r.Header.Get("X-Real-IP"); clientIP != "" {
		proxyReq.Header.Set("X-Real-IP", clientIP)
	} else if clientIP := r.RemoteAddr; clientIP != "" {
		proxyReq.Header.Set("X-Real-IP", strings.Split(clientIP, ":")[0])
	}

	// 设置 X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		proxyReq.Header.Set("X-Forwarded-For", xff)
	}

	// 设置 Referer
	if referer := r.Header.Get("Referer"); referer != "" {
		proxyReq.Header.Set("Referer", referer)
	}

	return proxyReq, nil
}

// 处理 CORS 头
func handleCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*" // 如果没有 Origin，使用通配符（但不推荐）
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "DNT, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Range, Authorization")
	w.Header().Set("Access-Control-Max-Age", "1728000")
}

// 判断是否为 JavaScript
func isJavaScript(contentType string) bool {
	contentType = strings.ToLower(contentType)
	return strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "application/x-javascript") ||
		strings.Contains(contentType, "text/javascript")
}

// 替换 URL
func replaceURLs(content []byte) []byte {
	result := string(content)
	for old, new := range urlReplacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return []byte(result)
}

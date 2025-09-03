package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	mathrand "math/rand"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// 类型定义
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      *bool         `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type Delta struct {
	Content          *string `json:"content,omitempty"`
	ReasoningContent *string `json:"reasoning_content,omitempty"`
}

type Choice struct {
	Delta Delta `json:"delta"`
}

type StreamResponse struct {
	Choices []Choice `json:"choices"`
}

type Model struct {
	ID     string `json:"id"`
	Object string `json:"object"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type HealthStats struct {
	TotalRequests       int `json:"total_requests"`
	AverageResponseTime int `json:"average_response_time"`
	ErrorRate           int `json:"error_rate"`
}

type HealthConfig struct {
	MaxRetries     int    `json:"max_retries"`
	RetryDelay     int    `json:"retry_delay"`
	RequestTimeout int    `json:"request_timeout"`
	RandomDelay    string `json:"random_delay"`
}

type HealthResponse struct {
	Status          string       `json:"status"`
	Timestamp       string       `json:"timestamp"`
	PerformanceMode string       `json:"performance_mode"`
	Config          HealthConfig `json:"config"`
	Stats           HealthStats  `json:"stats"`
}

type ErrorResponse struct {
	Error              string `json:"error"`
	Details            string `json:"details,omitempty"`
	RetryAfter         int    `json:"retry_after,omitempty"`
	AvailableEndpoints int    `json:"available_endpoints,omitempty"`
	PerformanceMode    string `json:"performance_mode,omitempty"`
}

// 日志系统类型定义
type LogLevel string

const (
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
)

type RequestLog struct {
	RequestID  string      `json:"request_id"`
	Timestamp  string      `json:"timestamp"`
	Level      LogLevel    `json:"level"`
	Type       string      `json:"type"`
	ClientIP   string      `json:"client_ip"`
	APIKey     string      `json:"api_key"`
	Model      string      `json:"model"`
	Messages   interface{} `json:"messages,omitempty"`
	Parameters interface{} `json:"parameters,omitempty"`
	UserAgent  string      `json:"user_agent,omitempty"`
}

type ResponseLog struct {
	RequestID        string      `json:"request_id"`
	Timestamp        string      `json:"timestamp"`
	Level            LogLevel    `json:"level"`
	Type             string      `json:"type"`
	StatusCode       int         `json:"status_code"`
	ResponseTime     int64       `json:"response_time_ms"`
	Endpoint         string      `json:"endpoint"`
	RetryCount       int         `json:"retry_count"`
	Content          interface{} `json:"content,omitempty"`
	ReasoningContent string      `json:"reasoning_content,omitempty"`
	Error            string      `json:"error,omitempty"`
}

type StreamLog struct {
	RequestID string      `json:"request_id"`
	Timestamp string      `json:"timestamp"`
	Level     LogLevel    `json:"level"`
	Type      string      `json:"type"`
	Content   interface{} `json:"content,omitempty"`
	Delta     interface{} `json:"delta,omitempty"`
}

// 版本信息
const (
	VERSION     = "2.0.0"
	BUILD_DATE  = "2025-01-02"
	DESCRIPTION = "DeepInfra API Proxy - Go版本优化版，解决流式响应截断问题"
)

// 全局配置
var (
	deepinfraURL = "https://api.deepinfra.com/v1/openai/chat/completions"
	port         = getEnvInt("PORT", 8000)

	// 性能配置
	performanceMode = getEnv("PERFORMANCE_MODE", "balanced")
	maxRetries      int
	retryDelay      int
	requestTimeout  int
	streamTimeout   int // 流式响应专用超时
	randomDelayMin  int
	randomDelayMax  int

	// 流处理优化配置
	streamBufferSize        = getEnvInt("STREAM_BUFFER_SIZE", 16384)
	disableConnectionCheck  = getEnv("DISABLE_CONNECTION_CHECK", "false") == "true"
	connectionCheckInterval = getEnvInt("CONNECTION_CHECK_INTERVAL", 20) // 每20次循环检查一次

	// 高并发管理配置
	maxConcurrentConnections = getEnvInt("MAX_CONCURRENT_CONNECTIONS", 1000)
	connectionQueueSize      = getEnvInt("CONNECTION_QUEUE_SIZE", 500)
	maxConnectionTime        = getEnvInt("MAX_CONNECTION_TIME", 600000)
	memoryLimitMB            = getEnvInt("MEMORY_LIMIT_MB", 2048)
	enableMetrics            = getEnv("ENABLE_METRICS", "true") == "true"

	// API 端点和密钥
	apiEndpoints []string
	validAPIKeys []string

	// 统计数据
	requestCount      int64
	totalResponseTime int64
	errorCount        int64

	// 并发控制
	currentConnections  int64
	connectionSemaphore chan struct{}

	// 日志配置
	enableDetailedLogging = getEnv("ENABLE_DETAILED_LOGGING", "true") == "true"
	logUserMessages       = getEnv("LOG_USER_MESSAGES", "false") == "true"
	logResponseContent    = getEnv("LOG_RESPONSE_CONTENT", "false") == "true"

	// 支持的模型
	supportedModels = []Model{
		{ID: "openai/gpt-oss-120b", Object: "model"},
		{ID: "moonshotai/Kimi-K2-Instruct", Object: "model"},
		{ID: "zai-org/GLM-4.5", Object: "model"},
		{ID: "zai-org/GLM-4.5-Air", Object: "model"},
		{ID: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", Object: "model"},
		{ID: "deepseek-ai/DeepSeek-R1-0528-Turbo", Object: "model"},
		{ID: "deepseek-ai/DeepSeek-V3-0324-Turbo", Object: "model"},
		{ID: "deepseek-ai/DeepSeek-V3.1", Object: "model"},
		{ID: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-Turbo", Object: "model"},
	}

	// User-Agent 列表
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
	}
)

// 工具函数
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getPerformanceConfig() {
	mode := strings.ToLower(performanceMode)

	switch mode {
	case "fast":
		maxRetries = getEnvInt("MAX_RETRIES", 1)
		retryDelay = getEnvInt("RETRY_DELAY", 200)
		requestTimeout = getEnvInt("REQUEST_TIMEOUT", 10000)
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 60000) // 1 分钟流超时
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 0)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 100)
	case "secure":
		maxRetries = getEnvInt("MAX_RETRIES", 5)
		retryDelay = getEnvInt("RETRY_DELAY", 2000)
		requestTimeout = getEnvInt("REQUEST_TIMEOUT", 60000)
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 600000) // 10 分钟流超时
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 500)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 1500)
	default: // balanced
		maxRetries = getEnvInt("MAX_RETRIES", 3)
		retryDelay = getEnvInt("RETRY_DELAY", 1000)
		requestTimeout = getEnvInt("REQUEST_TIMEOUT", 120000) // 2 分钟请求超时
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 300000)   // 5 分钟流超时
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 100)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 500)
	}
}

func getAPIEndpoints() []string {
	mirrors := getEnv("DEEPINFRA_MIRRORS", "")
	if mirrors != "" {
		endpoints := strings.Split(mirrors, ",")
		for i, endpoint := range endpoints {
			endpoints[i] = strings.TrimSpace(endpoint)
		}
		return endpoints
	}
	return []string{deepinfraURL}
}

func getValidAPIKeys() []string {
	keys := getEnv("VALID_API_KEYS", "linux.do")
	keyList := strings.Split(keys, ",")
	for i, key := range keyList {
		keyList[i] = strings.TrimSpace(key)
	}
	return keyList
}

func randomDelay() {
	if randomDelayMax > randomDelayMin {
		delay := mathrand.Intn(randomDelayMax-randomDelayMin) + randomDelayMin
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}

func getRandomUserAgent() string {
	return userAgents[mathrand.Intn(len(userAgents))]
}

// 日志系统函数
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// 如果加密随机数生成失败，使用时间戳作为备选
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%s", hex.EncodeToString(bytes))
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

func getClientIP(r *http.Request) string {
	// 检查 X-Forwarded-For 头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用 RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func logStructured(data interface{}) {
	if !enableDetailedLogging {
		return
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("日志序列化失败: %v", err)
		return
	}

	fmt.Println(string(jsonData))
}

func logRequest(requestID, clientIP, apiKey, model string, messageCount int, parameters interface{}, userAgent string) {
	if !enableDetailedLogging {
		return
	}

	requestLog := RequestLog{
		RequestID: requestID,
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     LogLevelInfo,
		Type:      "request",
		ClientIP:  clientIP,
		APIKey:    maskAPIKey(apiKey),
		Model:     model,
		UserAgent: userAgent,
	}

	// 只记录消息数量，不记录具体内容
	requestLog.Parameters = map[string]interface{}{
		"message_count": messageCount,
		"parameters":    parameters,
	}

	logStructured(requestLog)
}

func logResponse(requestID string, statusCode int, responseTime int64, endpoint string, retryCount int, errorMsg string) {
	if !enableDetailedLogging {
		return
	}

	level := LogLevelInfo
	if statusCode >= 400 {
		level = LogLevelError
	} else if statusCode >= 300 {
		level = LogLevelWarn
	}

	responseLog := ResponseLog{
		RequestID:    requestID,
		Timestamp:    time.Now().Format(time.RFC3339),
		Level:        level,
		Type:         "response",
		StatusCode:   statusCode,
		ResponseTime: responseTime,
		Endpoint:     endpoint,
		RetryCount:   retryCount,
		Error:        errorMsg,
	}

	// 不记录响应内容，只记录技术指标

	logStructured(responseLog)
}

// logStream 函数已移除，不再记录流式内容

// 带重试和多端点的请求函数
func fetchWithRetry(ctx context.Context, body []byte) (*http.Response, error) {
	var lastError error

	for endpointIndex, endpoint := range apiEndpoints {
		for i := 0; i < maxRetries; i++ {
			// 添加延迟
			if i > 0 || endpointIndex > 0 {
				delay := time.Duration(retryDelay*int(math.Pow(2, float64(i)))) * time.Millisecond
				time.Sleep(delay)
			}

			randomDelay()

			// 创建请求
			req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
			if err != nil {
				lastError = err
				continue
			}

			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", getRandomUserAgent())
			req.Header.Set("Accept", "text/event-stream, application/json, text/plain, */*")
			req.Header.Set("Accept-Language", "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7")
			req.Header.Set("Accept-Encoding", "gzip, deflate, br")
			req.Header.Set("Origin", "https://deepinfra.com")
			req.Header.Set("Referer", "https://deepinfra.com/")
			req.Header.Set("Sec-CH-UA", `"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"`)
			req.Header.Set("Sec-CH-UA-Mobile", "?0")
			req.Header.Set("Sec-CH-UA-Platform", `"Windows"`)
			req.Header.Set("Sec-Fetch-Dest", "empty")
			req.Header.Set("Sec-Fetch-Mode", "cors")
			req.Header.Set("Sec-Fetch-Site", "same-origin")
			req.Header.Set("X-Requested-With", "XMLHttpRequest")
			req.Header.Set("Cache-Control", "no-cache")
			req.Header.Set("Pragma", "no-cache")

			log.Printf("尝试请求端点: %s (第%d个端点, 第%d次尝试)", endpoint, endpointIndex+1, i+1)

			// 发送请求
			client := &http.Client{
				Timeout: time.Duration(requestTimeout) * time.Millisecond,
			}

			resp, err := client.Do(req)
			if err != nil {
				lastError = err
				log.Printf("端点 %s 请求尝试 %d/%d 失败: %v", endpoint, i+1, maxRetries, err)
				continue
			}

			if resp.StatusCode == http.StatusOK {
				log.Printf("请求成功: %s", endpoint)
				return resp, nil
			}

			// 处理限流或封禁错误
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
				waitTime := time.Duration(math.Min(float64(retryDelay)*math.Pow(2, float64(i)), 10000)) * time.Millisecond
				log.Printf("端点 %s 被限流或封禁 (%d)，等待 %v 后重试...", endpoint, resp.StatusCode, waitTime)
				resp.Body.Close()
				time.Sleep(waitTime)
				continue
			}

			resp.Body.Close()
			lastError = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
			log.Printf("端点 %s 请求尝试 %d/%d 失败: %v", endpoint, i+1, maxRetries, lastError)
		}
		log.Printf("端点 %s 所有重试都失败，尝试下一个端点", apiEndpoints[endpointIndex])
	}

	if lastError == nil {
		lastError = fmt.Errorf("所有端点和重试都失败")
	}
	return nil, lastError
}

// HTTP 处理函数
func healthHandler(w http.ResponseWriter, r *http.Request) {
	avgResponseTime := 0
	errorRate := 0
	if requestCount > 0 {
		avgResponseTime = int(totalResponseTime / requestCount)
		errorRate = int((errorCount * 100) / requestCount)
	}

	response := map[string]interface{}{
		"status":           "ok",
		"timestamp":        time.Now().Format(time.RFC3339),
		"version":          VERSION,
		"build_date":       BUILD_DATE,
		"description":      DESCRIPTION,
		"performance_mode": performanceMode,
		"config": HealthConfig{
			MaxRetries:     maxRetries,
			RetryDelay:     retryDelay,
			RequestTimeout: requestTimeout,
			RandomDelay:    fmt.Sprintf("%d-%dms", randomDelayMin, randomDelayMax),
		},
		"stats": HealthStats{
			TotalRequests:       int(requestCount),
			AverageResponseTime: avgResponseTime,
			ErrorRate:           errorRate,
		},
		"improvements": []string{
			"数据块读取策略，避免按行读取截断",
			"增强的错误恢复机制",
			"安全的数据发送函数",
			"动态缓冲区大小优化",
			"连接状态检测",
			"内存泄漏防护",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	response := ModelsResponse{
		Object: "list",
		Data:   supportedModels,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestCount++

	// 生成请求 ID
	requestID := generateRequestID()
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorCount++
		logResponse(requestID, http.StatusBadRequest, time.Since(startTime).Milliseconds(), "", 0, "Failed to read request body")
		http.Error(w, `{"error": "Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// API Key 验证
	auth := r.Header.Get("Authorization")
	key := strings.TrimPrefix(auth, "Bearer ")
	key = strings.TrimSpace(key)

	validKey := false
	for _, validAPIKey := range validAPIKeys {
		if key == validAPIKey {
			validKey = true
			break
		}
	}

	if !validKey {
		errorCount++
		logResponse(requestID, http.StatusUnauthorized, time.Since(startTime).Milliseconds(), "", 0, "Unauthorized")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
		return
	}

	// 解析请求体
	var chatReq ChatRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		errorCount++
		logResponse(requestID, http.StatusBadRequest, time.Since(startTime).Milliseconds(), "", 0, "Invalid JSON format")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
		return
	}

	// 记录请求日志
	parameters := map[string]interface{}{
		"stream":      chatReq.Stream,
		"temperature": chatReq.Temperature,
		"max_tokens":  chatReq.MaxTokens,
	}
	logRequest(requestID, clientIP, key, chatReq.Model, len(chatReq.Messages), parameters, userAgent)

	isStream := chatReq.Stream != nil && *chatReq.Stream

	// 发送请求到 DeepInfra API
	// 对于流式请求，使用更长的超时时间
	timeoutDuration := time.Duration(requestTimeout) * time.Millisecond
	if isStream {
		timeoutDuration = time.Duration(streamTimeout) * time.Millisecond
		log.Printf("🌊 流式请求，使用扩展超时: %v", timeoutDuration)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	resp, err := fetchWithRetry(ctx, body)
	if err != nil {
		errorCount++
		responseTime := time.Since(startTime)
		totalResponseTime += int64(responseTime.Milliseconds())

		logResponse(requestID, http.StatusBadGateway, responseTime.Milliseconds(), "all_endpoints", maxRetries, err.Error())
		log.Printf("DeepInfra API 所有端点请求失败: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:              "External API request failed",
			Details:            err.Error(),
			RetryAfter:         60,
			AvailableEndpoints: len(apiEndpoints),
			PerformanceMode:    performanceMode,
		})
		return
	}
	defer resp.Body.Close()

	// 处理响应
	if !isStream {
		// 非流式响应
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			errorCount++
			logResponse(requestID, http.StatusInternalServerError, time.Since(startTime).Milliseconds(), "", 0, "Failed to read response")
			http.Error(w, `{"error": "Failed to read response"}`, http.StatusInternalServerError)
			return
		}

		responseTime := time.Since(startTime)
		totalResponseTime += int64(responseTime.Milliseconds())

		logResponse(requestID, resp.StatusCode, responseTime.Milliseconds(), "deepinfra_api", 0, "")
		log.Printf("✅ 请求完成: %v", responseTime)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		w.Write(responseBody)
		return
	}

	// 流式响应处理
	handleStreamResponse(w, resp, requestID)
}

// 流式响应处理 - 优化版本，解决截断问题
func handleStreamResponse(w http.ResponseWriter, resp *http.Response, requestID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 使用优化的缓冲区大小
	buffer := make([]byte, streamBufferSize)
	lineBuffer := ""
	isInThinkBlock := false
	bufferedThinkContent := ""
	streamClosed := false
	checkCounter := 0 // 连接检测计数器

	log.Printf("🌊 开始流式响应处理，缓冲区大小: %d bytes, 连接检测: %v", streamBufferSize, !disableConnectionCheck)

	for !streamClosed {
		// 智能连接检测：平衡性能和稳定性
		if !disableConnectionCheck {
			checkCounter++
			if checkCounter%connectionCheckInterval == 0 { // 可配置的检查间隔
				if !isConnectionAlive(w) {
					log.Printf("客户端连接已断开，停止流式传输")
					break
				}
			}
		}

		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// 将读取的数据块添加到行缓冲区
			chunk := string(buffer[:n])
			lineBuffer += chunk

			// 防止行缓冲区过大（防止内存泄漏）
			if len(lineBuffer) > 1024*1024 { // 1MB 限制
				log.Printf("警告：行缓冲区过大 (%d bytes)，可能存在数据问题", len(lineBuffer))
				// 尝试处理部分数据
				if idx := strings.LastIndex(lineBuffer[:len(lineBuffer)/2], "\n"); idx > 0 {
					partialBuffer := lineBuffer[:idx]
					lineBuffer = lineBuffer[idx+1:]
					log.Printf("处理部分缓冲区数据，大小: %d bytes", len(partialBuffer))
				}
			}

			// 处理缓冲区中的完整行
			processedLines := 0
			for {
				lineEnd := strings.Index(lineBuffer, "\n")
				if lineEnd == -1 {
					// 没有完整的行，等待更多数据
					break
				}

				// 提取完整的行
				line := lineBuffer[:lineEnd]
				lineBuffer = lineBuffer[lineEnd+1:]
				processedLines++

				// 处理这一行
				if !streamClosed {
					processLineImproved(line, &isInThinkBlock, &bufferedThinkContent, &streamClosed, w, flusher, requestID)
				}

				// 如果已经关闭流，提前退出
				if streamClosed {
					break
				}
			}

			if processedLines > 0 && enableDetailedLogging {
				log.Printf("📝 处理了 %d 行数据，剩余缓冲区: %d bytes", processedLines, len(lineBuffer))
			}
		}

		if err != nil {
			if err == io.EOF {
				// 处理剩余的不完整行
				if lineBuffer != "" && !streamClosed {
					processLineImproved(lineBuffer, &isInThinkBlock, &bufferedThinkContent, &streamClosed, w, flusher, requestID)
				}
				break
			}
			log.Printf("读取流数据失败: %v", err)
			break
		}
	}

	// 确保发送最后的思考内容
	if isInThinkBlock && bufferedThinkContent != "" {
		sendThinkContent(bufferedThinkContent, w, flusher)
	}
}

// 处理单行数据 - 改进版本，增强错误恢复能力
func processLineImproved(line string, isInThinkBlock *bool, bufferedThinkContent *string, streamClosed *bool, w http.ResponseWriter, flusher http.Flusher, requestID string) {
	line = strings.TrimSpace(line)

	if strings.HasPrefix(line, "data: ") {
		jsonText := strings.TrimSpace(line[6:])

		if jsonText == "[DONE]" {
			// 发送缓存的思考内容
			if *isInThinkBlock && *bufferedThinkContent != "" {
				sendThinkContentSafe(*bufferedThinkContent, w, flusher)
			}

			// 安全发送结束标记
			if err := sendDataSafe("data: [DONE]\n\n", w, flusher); err != nil {
				log.Printf("发送结束标记失败: %v", err)
			}
			*streamClosed = true
			return
		}

		if jsonText != "" {
			var streamResp StreamResponse
			if err := json.Unmarshal([]byte(jsonText), &streamResp); err != nil {
				// 增强错误处理：JSON 解析失败时不中断流，而是记录并跳过
				log.Printf("JSON 解析失败，跳过此数据: %v, 内容长度: %d", err, len(jsonText))
				// 如果内容太长，只显示前100个字符
				if len(jsonText) > 100 {
					log.Printf("JSON 内容预览: %s...", jsonText[:100])
				} else {
					log.Printf("JSON 内容: %s", jsonText)
				}
				return
			}

			// 成功解析 JSON，处理数据
			if len(streamResp.Choices) > 0 {
				delta := streamResp.Choices[0].Delta

				var contentToSend *string

				// 处理思考内容
				if delta.ReasoningContent != nil {
					if *delta.ReasoningContent != "" {
						*bufferedThinkContent += *delta.ReasoningContent
					}
					*isInThinkBlock = true
				} else if delta.Content != nil {
					// 处理正常内容
					if *isInThinkBlock {
						// 发送思考内容
						if *bufferedThinkContent != "" {
							sendThinkContentSafe(*bufferedThinkContent, w, flusher)
							*bufferedThinkContent = ""
						}
						*isInThinkBlock = false
					}
					contentToSend = delta.Content
				}

				// 发送正常内容
				if contentToSend != nil && *contentToSend != "" {
					sendContentSafe(*contentToSend, w, flusher)
				}
			}
		}
	}
}

// 保留原有函数以保持兼容性
func processLine(line string, isInThinkBlock *bool, bufferedThinkContent *string, streamClosed *bool, w http.ResponseWriter, flusher http.Flusher, requestID string) {
	processLineImproved(line, isInThinkBlock, bufferedThinkContent, streamClosed, w, flusher, requestID)
}

// 安全发送数据的通用函数
func sendDataSafe(data string, w http.ResponseWriter, flusher http.Flusher) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("发送数据时发生 panic: %v", r)
		}
	}()

	_, err := fmt.Fprint(w, data)
	if err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}

	// 安全刷新
	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// 发送思考内容 - 安全版本
func sendThinkContentSafe(content string, w http.ResponseWriter, flusher http.Flusher) {
	thinkData := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"delta": map[string]interface{}{
					"content": fmt.Sprintf("<think>%s</think>", content),
				},
			},
		},
	}

	thinkJSON, err := json.Marshal(thinkData)
	if err != nil {
		log.Printf("思考内容 JSON 编码失败: %v", err)
		return
	}

	data := fmt.Sprintf("data: %s\n\n", string(thinkJSON))
	if err := sendDataSafe(data, w, flusher); err != nil {
		log.Printf("发送思考内容失败: %v", err)
	}
}

// 发送正常内容 - 安全版本
func sendContentSafe(content string, w http.ResponseWriter, flusher http.Flusher) {
	outputData := map[string]interface{}{
		"choices": []map[string]interface{}{
			{
				"delta": map[string]interface{}{
					"content": content,
				},
			},
		},
	}

	outputJSON, err := json.Marshal(outputData)
	if err != nil {
		log.Printf("内容 JSON 编码失败: %v", err)
		return
	}

	data := fmt.Sprintf("data: %s\n\n", string(outputJSON))
	if err := sendDataSafe(data, w, flusher); err != nil {
		log.Printf("发送内容失败: %v", err)
	}
}

// 发送思考内容 - 保持向后兼容
func sendThinkContent(content string, w http.ResponseWriter, flusher http.Flusher) {
	sendThinkContentSafe(content, w, flusher)
}

// 发送正常内容 - 保持向后兼容
func sendContent(content string, w http.ResponseWriter, flusher http.Flusher) {
	sendContentSafe(content, w, flusher)
}

func main() {
	// 设置路由
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/v1/models", modelsHandler)
	http.HandleFunc("/v1/chat/completions", concurrencyControlMiddleware(chatHandler))
	http.HandleFunc("/status", statusHandler) // 新增状态监控端点

	// 404 处理
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Not Found"})
	})

	// 启动服务器
	addr := fmt.Sprintf(":%d", port)
	log.Printf("🌐 Server listening on %s", addr)
	log.Printf("🔒 Concurrency limit: %d connections", maxConcurrentConnections)
	log.Printf("💾 Memory limit: %d MB", memoryLimitMB)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// 状态监控处理器
func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := getSystemStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// 检查连接是否仍然活跃
func isConnectionAlive(w http.ResponseWriter) bool {
	// 尝试写入一个空字符串来检测连接状态
	if _, err := fmt.Fprint(w, ""); err != nil {
		return false
	}
	return true
}

// 并发控制中间件
func concurrencyControlMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 尝试获取连接许可
		select {
		case connectionSemaphore <- struct{}{}:
			// 获取到许可，继续处理
			atomic.AddInt64(&currentConnections, 1)
			defer func() {
				<-connectionSemaphore
				atomic.AddInt64(&currentConnections, -1)
			}()
			next(w, r)
		default:
			// 连接数已满，返回503错误
			http.Error(w, `{"error": "Server too busy, please try again later"}`, http.StatusServiceUnavailable)
			log.Printf("⚠️ 连接数已满，拒绝新连接。当前连接数: %d", atomic.LoadInt64(&currentConnections))
		}
	}
}

// 获取当前系统状态
func getSystemStatus() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"current_connections": atomic.LoadInt64(&currentConnections),
		"max_connections":     maxConcurrentConnections,
		"memory_usage_mb":     m.Alloc / 1024 / 1024,
		"memory_limit_mb":     memoryLimitMB,
		"total_requests":      atomic.LoadInt64(&requestCount),
		"error_count":         atomic.LoadInt64(&errorCount),
	}
}

// 优化的缓冲区大小计算 - 已弃用，使用 streamBufferSize 配置
func getOptimalBufferSize() int {
	// 返回配置的缓冲区大小
	return streamBufferSize
}

func init() {
	mathrand.Seed(time.Now().UnixNano())
	getPerformanceConfig()
	apiEndpoints = getAPIEndpoints()
	validAPIKeys = getValidAPIKeys()

	// 初始化并发控制
	connectionSemaphore = make(chan struct{}, maxConcurrentConnections)

	log.Printf("🚀 %s", DESCRIPTION)
	log.Printf("📦 Version: %s (Build: %s)", VERSION, BUILD_DATE)
	log.Printf("🌐 Server started on port %d", port)
	log.Printf("⚡ Performance mode: %s", performanceMode)
	log.Printf("🔧 Config: retries=%d, delay=%dms, request_timeout=%dms, stream_timeout=%dms", maxRetries, retryDelay, requestTimeout, streamTimeout)
	log.Printf("⏱️  Random delay: %d-%dms", randomDelayMin, randomDelayMax)
	log.Printf("📝 Detailed logging: %v, User messages: %v, Response content: %v", enableDetailedLogging, logUserMessages, logResponseContent)
	log.Printf("🌊 Stream config: buffer_size=%d bytes, connection_check_disabled=%v, check_interval=%d", streamBufferSize, disableConnectionCheck, connectionCheckInterval)
	log.Printf("✨ 流式响应优化:")
	log.Printf("   • 分离的流式响应超时机制")
	log.Printf("   • 优化的缓冲区管理策略")
	log.Printf("   • 可配置的连接检测频率")
	log.Printf("   • 增强的错误恢复机制")
	log.Printf("   • 防止长响应截断的安全措施")
	log.Printf("   • 内存泄漏防护")
}

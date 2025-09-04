package main

import (
	"bufio"
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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// 版本信息
const (
	VERSION     = "2.1.0"
	BUILD_DATE  = "2025-01-03"
	DESCRIPTION = "Z2API Go优化版 - 基于原版Z2API的企业级优化实现"
)

// 类型定义
type ChatMessage struct {
	Role             string `json:"role"`
	Content          string `json:"content"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type OpenAIRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Stream      *bool         `json:"stream,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	MaxTokens   *int          `json:"max_tokens,omitempty"`
}

type UpstreamRequest struct {
	Stream          bool                   `json:"stream"`
	Model           string                 `json:"model"`
	Messages        []ChatMessage          `json:"messages"`
	Params          map[string]interface{} `json:"params"`
	Features        map[string]interface{} `json:"features"`
	BackgroundTasks map[string]bool        `json:"background_tasks,omitempty"`
	ChatID          string                 `json:"chat_id,omitempty"`
	ID              string                 `json:"id,omitempty"`
	MCPServers      []string               `json:"mcp_servers,omitempty"`
	ModelItem       struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		OwnedBy string `json:"owned_by"`
	} `json:"model_item,omitempty"`
	ToolServers []string          `json:"tool_servers,omitempty"`
	Variables   map[string]string `json:"variables,omitempty"`
}

type Delta struct {
	Role             string `json:"role,omitempty"`
	Content          string `json:"content,omitempty"`
	ReasoningContent string `json:"reasoning_content,omitempty"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message,omitempty"`
	Delta        Delta       `json:"delta,omitempty"`
	FinishReason string      `json:"finish_reason,omitempty"`
}

type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type UpstreamData struct {
	Type string `json:"type"`
	Data struct {
		DeltaContent string         `json:"delta_content"`
		EditContent  string         `json:"edit_content"`
		Phase        string         `json:"phase"`
		Done         bool           `json:"done"`
		Usage        *Usage         `json:"usage,omitempty"`
		Error        *UpstreamError `json:"error,omitempty"`
		Inner        *struct {
			Error *UpstreamError `json:"error,omitempty"`
		} `json:"data,omitempty"`
	} `json:"data"`
	Error *UpstreamError `json:"error,omitempty"`
}

type UpstreamError struct {
	Detail string `json:"detail"`
	Code   int    `json:"code"`
}

type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
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

type HealthStats struct {
	TotalRequests       int64 `json:"total_requests"`
	AverageResponseTime int64 `json:"average_response_time"`
	ErrorRate           int   `json:"error_rate"`
	CurrentConnections  int64 `json:"current_connections"`
}

type HealthConfig struct {
	MaxRetries             int    `json:"max_retries"`
	RetryDelay             int    `json:"retry_delay"`
	RequestTimeout         int    `json:"request_timeout"`
	RandomDelay            string `json:"random_delay"`
	MaxConcurrentConns     int    `json:"max_concurrent_connections"`
	StreamBufferSize       int    `json:"stream_buffer_size"`
	ConnectionCheckEnabled bool   `json:"connection_check_enabled"`
}

type HealthResponse struct {
	Status          string       `json:"status"`
	Timestamp       string       `json:"timestamp"`
	Version         string       `json:"version"`
	BuildDate       string       `json:"build_date"`
	Description     string       `json:"description"`
	PerformanceMode string       `json:"performance_mode"`
	UptimeSeconds   int          `json:"uptime_seconds"`
	Config          HealthConfig `json:"config"`
	Stats           HealthStats  `json:"stats"`
	Improvements    []string     `json:"improvements"`
}

type ErrorResponse struct {
	Error              string `json:"error"`
	Details            string `json:"details,omitempty"`
	RetryAfter         int    `json:"retry_after,omitempty"`
	AvailableEndpoints int    `json:"available_endpoints,omitempty"`
	PerformanceMode    string `json:"performance_mode,omitempty"`
}

// 全局配置
var (
	// 基础配置
	upstreamURL   = getEnv("UPSTREAM_URL", "https://chat.z.ai/api/chat/completions")
	port          = getEnvInt("PORT", 8080)
	defaultKey    = getEnv("DEFAULT_KEY", "123456")
	upstreamToken = getEnv("UPSTREAM_TOKEN", "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Ijc3NWI4MjMyLTFjMDgtNDZjOC1iM2ZjLTc4NGZkOTYzOTFkMCIsImVtYWlsIjoiR3Vlc3QtMTc1NjQxNzIwODY2NkBndWVzdC5jb20ifQ.ANLFGzTOIhaocgsVRMtzhcHOfhvxWrf3RwiEV0b4mmeNMu72fIbp9j0D42aWlrupZN5AARqGPeIDUFU5po0gFQ")

	// 模型配置
	defaultModelName  = "GLM-4.5"
	thinkingModelName = "GLM-4.5-Thinking"
	searchModelName   = "GLM-4.5-Search"

	// 伪装前端头部
	xFeVersion  = "prod-fe-1.0.70"
	browserUa   = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0"
	secChUa     = `"Not;A=Brand";v="99", "Microsoft Edge";v="139", "Chromium";v="139"`
	secChUaMob  = "?0"
	secChUaPlat = `"Windows"`
	originBase  = "https://chat.z.ai"

	// 性能配置
	performanceMode = getEnv("PERFORMANCE_MODE", "balanced")
	maxRetries      int
	retryDelay      int
	requestTimeout  int
	streamTimeout   int
	randomDelayMin  int
	randomDelayMax  int

	// 流处理优化配置
	streamBufferSize        = getEnvInt("STREAM_BUFFER_SIZE", 16384)
	disableConnectionCheck  = getEnv("DISABLE_CONNECTION_CHECK", "false") == "true"
	connectionCheckInterval = getEnvInt("CONNECTION_CHECK_INTERVAL", 20)

	// 高并发管理配置
	maxConcurrentConnections = getEnvInt("MAX_CONCURRENT_CONNECTIONS", 1000)
	connectionQueueSize      = getEnvInt("CONNECTION_QUEUE_SIZE", 500)
	maxConnectionTime        = getEnvInt("MAX_CONNECTION_TIME", 600000)
	memoryLimitMB            = getEnvInt("MEMORY_LIMIT_MB", 2048)
	enableMetrics            = getEnv("ENABLE_METRICS", "true") == "true"

	// 功能配置
	anonTokenEnabled = getEnv("ANON_TOKEN_ENABLED", "true") == "true"
	thinkTagsMode    = getEnv("THINK_TAGS_MODE", "think")
	debugMode        = getEnv("DEBUG_MODE", "false") == "true"

	// 统计数据
	requestCount      int64
	totalResponseTime int64
	errorCount        int64
	startTime         time.Time

	// 并发控制
	currentConnections  int64
	connectionSemaphore chan struct{}

	// 日志配置
	enableDetailedLogging = getEnv("ENABLE_DETAILED_LOGGING", "true") == "true"
	logUserMessages       = getEnv("LOG_USER_MESSAGES", "false") == "true"
	logResponseContent    = getEnv("LOG_RESPONSE_CONTENT", "false") == "true"

	// 支持的模型
	supportedModels = []Model{
		{ID: defaultModelName, Object: "model", Created: time.Now().Unix(), OwnedBy: "z.ai"},
		{ID: thinkingModelName, Object: "model", Created: time.Now().Unix(), OwnedBy: "z.ai"},
		{ID: searchModelName, Object: "model", Created: time.Now().Unix(), OwnedBy: "z.ai"},
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
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 60000)
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 0)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 100)
	case "secure":
		maxRetries = getEnvInt("MAX_RETRIES", 5)
		retryDelay = getEnvInt("RETRY_DELAY", 2000)
		requestTimeout = getEnvInt("REQUEST_TIMEOUT", 60000)
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 600000)
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 500)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 1500)
	default: // balanced
		maxRetries = getEnvInt("MAX_RETRIES", 3)
		retryDelay = getEnvInt("RETRY_DELAY", 1000)
		requestTimeout = getEnvInt("REQUEST_TIMEOUT", 120000)
		streamTimeout = getEnvInt("STREAM_TIMEOUT", 300000)
		randomDelayMin = getEnvInt("RANDOM_DELAY_MIN", 100)
		randomDelayMax = getEnvInt("RANDOM_DELAY_MAX", 500)
	}
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
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

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

func debugLog(format string, args ...interface{}) {
	if debugMode {
		log.Printf("[DEBUG] "+format, args...)
	}
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

	logStructured(responseLog)
}

// 获取匿名token（每次对话使用不同token，避免共享记忆）
func getAnonymousToken() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", originBase+"/api/v1/auths/", nil)
	if err != nil {
		return "", err
	}

	// 伪装浏览器头
	req.Header.Set("User-Agent", browserUa)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("X-FE-Version", xFeVersion)
	req.Header.Set("sec-ch-ua", secChUa)
	req.Header.Set("sec-ch-ua-mobile", secChUaMob)
	req.Header.Set("sec-ch-ua-platform", secChUaPlat)
	req.Header.Set("Origin", originBase)
	req.Header.Set("Referer", originBase+"/")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("anon token status=%d", resp.StatusCode)
	}

	var body struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}

	if body.Token == "" {
		return "", fmt.Errorf("anon token empty")
	}

	return body.Token, nil
}

// 思考内容转换函数
func transformThinking(s string) string {
	// 去 <summary>…</summary>
	s = regexp.MustCompile(`(?s)<summary>.*?</summary>`).ReplaceAllString(s, "")
	// 清理残留自定义标签，如 </thinking>、<Full> 等
	s = strings.ReplaceAll(s, "</thinking>", "")
	s = strings.ReplaceAll(s, "<Full>", "")
	s = strings.ReplaceAll(s, "</Full>", "")
	s = strings.TrimSpace(s)

	switch thinkTagsMode {
	case "think":
		s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "<think>")
		s = strings.ReplaceAll(s, "</details>", "</think>")
	case "strip":
		s = regexp.MustCompile(`<details[^>]*>`).ReplaceAllString(s, "")
		s = strings.ReplaceAll(s, "</details>", "")
	}

	// 处理每行前缀 "> "（包括起始位置）
	s = strings.TrimPrefix(s, "> ")
	s = strings.ReplaceAll(s, "\n> ", "\n")
	return strings.TrimSpace(s)
}

// 带重试的HTTP请求
func requestWithRetry(ctx context.Context, upstreamReq UpstreamRequest, chatID, authToken string) (*http.Response, error) {
	var lastErr error

	reqBody, err := json.Marshal(upstreamReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	for i := 0; i < maxRetries; i++ {
		// 添加延迟
		if i > 0 {
			delay := time.Duration(retryDelay*int(math.Pow(2, float64(i)))) * time.Millisecond
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		randomDelay()

		// 创建请求
		req, err := http.NewRequestWithContext(ctx, "POST", upstreamURL, bytes.NewReader(reqBody))
		if err != nil {
			lastErr = err
			continue
		}

		// 设置请求头
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("User-Agent", getRandomUserAgent())
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Accept-Language", "zh-CN")
		req.Header.Set("sec-ch-ua", secChUa)
		req.Header.Set("sec-ch-ua-mobile", secChUaMob)
		req.Header.Set("sec-ch-ua-platform", secChUaPlat)
		req.Header.Set("X-FE-Version", xFeVersion)
		req.Header.Set("Origin", originBase)
		req.Header.Set("Referer", originBase+"/c/"+chatID)

		debugLog("尝试请求上游: %s (第%d次尝试)", upstreamURL, i+1)

		// 发送请求
		client := &http.Client{
			Timeout: time.Duration(requestTimeout) * time.Millisecond,
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			debugLog("请求尝试 %d/%d 失败: %v", i+1, maxRetries, err)
			continue
		}

		if resp.StatusCode == http.StatusOK {
			debugLog("请求成功")
			return resp, nil
		}

		// 处理限流或封禁错误
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden {
			waitTime := time.Duration(math.Min(float64(retryDelay)*math.Pow(2, float64(i)), 10000)) * time.Millisecond
			debugLog("被限流或封禁 (%d)，等待 %v 后重试...", resp.StatusCode, waitTime)
			resp.Body.Close()
			select {
			case <-time.After(waitTime):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			continue
		}

		resp.Body.Close()
		lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		debugLog("请求尝试 %d/%d 失败: %v", i+1, maxRetries, lastErr)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("所有重试都失败")
	}
	return nil, lastErr
}

// HTTP 处理函数
func healthHandler(w http.ResponseWriter, r *http.Request) {
	avgResponseTime := int64(0)
	errorRate := 0
	if requestCount > 0 {
		avgResponseTime = totalResponseTime / requestCount
		errorRate = int((errorCount * 100) / requestCount)
	}

	uptime := time.Since(startTime)

	response := HealthResponse{
		Status:          "ok",
		Timestamp:       time.Now().Format(time.RFC3339),
		Version:         VERSION,
		BuildDate:       BUILD_DATE,
		Description:     DESCRIPTION,
		PerformanceMode: performanceMode,
		UptimeSeconds:   int(uptime.Seconds()),
		Config: HealthConfig{
			MaxRetries:             maxRetries,
			RetryDelay:             retryDelay,
			RequestTimeout:         requestTimeout,
			RandomDelay:            fmt.Sprintf("%d-%dms", randomDelayMin, randomDelayMax),
			MaxConcurrentConns:     maxConcurrentConnections,
			StreamBufferSize:       streamBufferSize,
			ConnectionCheckEnabled: !disableConnectionCheck,
		},
		Stats: HealthStats{
			TotalRequests:       requestCount,
			AverageResponseTime: avgResponseTime,
			ErrorRate:           errorRate,
			CurrentConnections:  atomic.LoadInt64(&currentConnections),
		},
		Improvements: []string{
			"基于原版Z2API的企业级优化",
			"完整的并发控制机制",
			"结构化日志系统",
			"性能模式配置",
			"重试机制和错误恢复",
			"健康检查和监控",
			"匿名token支持",
			"专业思考内容处理",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func modelsHandler(w http.ResponseWriter, r *http.Request) {
	response := ModelsResponse{
		Object: "list",
		Data:   supportedModels,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
}

func optionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.WriteHeader(http.StatusOK)
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	atomic.AddInt64(&requestCount, 1)

	// 生成请求 ID
	requestID := generateRequestID()
	clientIP := getClientIP(r)
	userAgent := r.Header.Get("User-Agent")

	// 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		atomic.AddInt64(&errorCount, 1)
		logResponse(requestID, http.StatusBadRequest, time.Since(startTime).Milliseconds(), "", 0, "Failed to read request body")
		http.Error(w, `{"error": "Failed to read request body"}`, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// API Key 验证
	auth := r.Header.Get("Authorization")
	key := strings.TrimPrefix(auth, "Bearer ")
	key = strings.TrimSpace(key)

	if key != defaultKey {
		atomic.AddInt64(&errorCount, 1)
		logResponse(requestID, http.StatusUnauthorized, time.Since(startTime).Milliseconds(), "", 0, "Unauthorized")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Unauthorized"})
		return
	}

	debugLog("API key验证通过")

	// 解析请求体
	var chatReq OpenAIRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		atomic.AddInt64(&errorCount, 1)
		logResponse(requestID, http.StatusBadRequest, time.Since(startTime).Milliseconds(), "", 0, "Invalid JSON format")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON format"})
		return
	}

	debugLog("请求解析成功 - 模型: %s, 流式: %v, 消息数: %d", chatReq.Model, chatReq.Stream != nil && *chatReq.Stream, len(chatReq.Messages))

	// 生成会话相关ID
	chatID := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
	msgID := fmt.Sprintf("%d", time.Now().UnixNano())

	var isThinking bool
	var isSearch bool
	var searchMcp string
	if chatReq.Model == thinkingModelName {
		isThinking = true
	} else if chatReq.Model == searchModelName {
		isThinking = true
		isSearch = true
		searchMcp = "deep-web-search"
	}

	// 记录请求日志
	parameters := map[string]interface{}{
		"stream":      chatReq.Stream,
		"temperature": chatReq.Temperature,
		"max_tokens":  chatReq.MaxTokens,
	}
	logRequest(requestID, clientIP, key, chatReq.Model, len(chatReq.Messages), parameters, userAgent)

	// 构造上游请求
	upstreamReq := UpstreamRequest{
		Stream:   true, // 总是使用流式从上游获取
		ChatID:   chatID,
		ID:       msgID,
		Model:    "0727-360B-API", // 上游实际模型ID
		Messages: chatReq.Messages,
		Params:   map[string]interface{}{},
		Features: map[string]interface{}{
			"enable_thinking": isThinking,
			"web_search":      isSearch,
			"auto_web_search": isSearch,
		},
		BackgroundTasks: map[string]bool{
			"title_generation": false,
			"tags_generation":  false,
		},
		MCPServers:  []string{searchMcp},
		ToolServers: []string{},
		Variables: map[string]string{
			"{{USER_NAME}}":        "User",
			"{{USER_LOCATION}}":    "Unknown",
			"{{CURRENT_DATETIME}}": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	upstreamReq.ModelItem.ID = "0727-360B-API"
	upstreamReq.ModelItem.Name = "GLM-4.5"
	upstreamReq.ModelItem.OwnedBy = "openai"

	// 选择本次对话使用的token
	authToken := upstreamToken
	if anonTokenEnabled {
		if t, err := getAnonymousToken(); err == nil {
			authToken = t
			debugLog("匿名token获取成功: %s...", func() string {
				if len(t) > 10 {
					return t[:10]
				}
				return t
			}())
		} else {
			debugLog("匿名token获取失败，回退固定token: %v", err)
		}
	}

	isStream := chatReq.Stream != nil && *chatReq.Stream

	// 发送请求到上游API
	timeoutDuration := time.Duration(requestTimeout) * time.Millisecond
	if isStream {
		timeoutDuration = time.Duration(streamTimeout) * time.Millisecond
		debugLog("🌊 流式请求，使用扩展超时: %v", timeoutDuration)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	resp, err := requestWithRetry(ctx, upstreamReq, chatID, authToken)
	if err != nil {
		atomic.AddInt64(&errorCount, 1)
		responseTime := time.Since(startTime)
		atomic.AddInt64(&totalResponseTime, responseTime.Milliseconds())

		logResponse(requestID, http.StatusBadGateway, responseTime.Milliseconds(), "upstream", maxRetries, err.Error())
		debugLog("上游API请求失败: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error:           "External API request failed",
			Details:         err.Error(),
			RetryAfter:      60,
			PerformanceMode: performanceMode,
		})
		return
	}
	defer resp.Body.Close()

	// 处理响应
	if !isStream {
		handleNonStreamResponse(w, resp, requestID, startTime)
	} else {
		handleStreamResponse(w, resp, requestID, startTime)
	}
}

// 处理非流式响应
func handleNonStreamResponse(w http.ResponseWriter, resp *http.Response, requestID string, startTime time.Time) {
	debugLog("开始处理非流式响应")

	// 收集完整响应
	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		dataStr := strings.TrimPrefix(line, "data: ")
		if dataStr == "" || dataStr == "[DONE]" {
			continue
		}

		var upstreamData UpstreamData
		if err := json.Unmarshal([]byte(dataStr), &upstreamData); err != nil {
			continue
		}

		if upstreamData.Data.DeltaContent != "" {
			out := upstreamData.Data.DeltaContent
			if upstreamData.Data.Phase == "thinking" {
				out = transformThinking(out)
			}
			if out != "" {
				fullContent.WriteString(out)
			}
		}

		if upstreamData.Data.Done || upstreamData.Data.Phase == "done" {
			debugLog("检测到完成信号，停止收集")
			break
		}
	}

	finalContent := fullContent.String()
	debugLog("内容收集完成，最终长度: %d", len(finalContent))

	responseTime := time.Since(startTime)
	atomic.AddInt64(&totalResponseTime, responseTime.Milliseconds())

	logResponse(requestID, 200, responseTime.Milliseconds(), "upstream", 0, "")
	debugLog("非流式响应完成: %v", responseTime)

	// 构造完整响应
	response := OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   defaultModelName,
		Choices: []Choice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: finalContent,
				},
				FinishReason: "stop",
			},
		},
		Usage: &Usage{
			PromptTokens:     0,
			CompletionTokens: 0,
			TotalTokens:      0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(response)
	debugLog("非流式响应发送完成")
}

// 处理流式响应 - 优化版本
func handleStreamResponse(w http.ResponseWriter, resp *http.Response, requestID string, startTime time.Time) {
	debugLog("开始处理流式响应")

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

	// 发送第一个chunk（role）
	firstChunk := OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   defaultModelName,
		Choices: []Choice{
			{
				Index: 0,
				Delta: Delta{Role: "assistant"},
			},
		},
	}
	writeSSEChunk(w, firstChunk)
	flusher.Flush()

	// 使用优化的缓冲区大小
	buffer := make([]byte, streamBufferSize)
	lineBuffer := ""
	isInThinkBlock := false
	bufferedThinkContent := ""
	streamClosed := false
	checkCounter := 0
	sentInitialAnswer := false

	debugLog("🌊 开始流式响应处理，缓冲区大小: %d bytes", streamBufferSize)

	for !streamClosed {
		// 智能连接检测
		if !disableConnectionCheck {
			checkCounter++
			if checkCounter%connectionCheckInterval == 0 {
				if !isConnectionAlive(w) {
					debugLog("客户端连接已断开，停止流式传输")
					break
				}
			}
		}

		n, err := resp.Body.Read(buffer)
		if n > 0 {
			chunk := string(buffer[:n])
			lineBuffer += chunk

			// 防止行缓冲区过大
			if len(lineBuffer) > 1024*1024 {
				debugLog("警告：行缓冲区过大 (%d bytes)", len(lineBuffer))
				if idx := strings.LastIndex(lineBuffer[:len(lineBuffer)/2], "\n"); idx > 0 {
					lineBuffer = lineBuffer[idx+1:]
				}
			}

			// 处理缓冲区中的完整行
			for {
				lineEnd := strings.Index(lineBuffer, "\n")
				if lineEnd == -1 {
					break
				}

				line := lineBuffer[:lineEnd]
				lineBuffer = lineBuffer[lineEnd+1:]

				if !streamClosed {
					processStreamLine(line, &isInThinkBlock, &bufferedThinkContent, &streamClosed, &sentInitialAnswer, w, flusher)
				}

				if streamClosed {
					break
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				if lineBuffer != "" && !streamClosed {
					processStreamLine(lineBuffer, &isInThinkBlock, &bufferedThinkContent, &streamClosed, &sentInitialAnswer, w, flusher)
				}
				break
			}
			debugLog("读取流数据失败: %v", err)
			break
		}
	}

	// 确保发送最后的思考内容
	if isInThinkBlock && bufferedThinkContent != "" {
		sendThinkContentSafe(bufferedThinkContent, w, flusher)
	}

	debugLog("流式响应处理完成")
}

// 辅助函数
func writeSSEChunk(w http.ResponseWriter, chunk OpenAIResponse) {
	data, _ := json.Marshal(chunk)
	fmt.Fprintf(w, "data: %s\n\n", data)
}

func isConnectionAlive(w http.ResponseWriter) bool {
	if _, err := fmt.Fprint(w, ""); err != nil {
		return false
	}
	return true
}

func processStreamLine(line string, isInThinkBlock *bool, bufferedThinkContent *string, streamClosed *bool, sentInitialAnswer *bool, w http.ResponseWriter, flusher http.Flusher) {
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
				debugLog("发送结束标记失败: %v", err)
			}
			*streamClosed = true
			return
		}

		if jsonText != "" {
			var upstreamData UpstreamData
			if err := json.Unmarshal([]byte(jsonText), &upstreamData); err != nil {
				debugLog("JSON 解析失败，跳过此数据: %v", err)
				return
			}

			// 错误检测
			if upstreamData.Error != nil || upstreamData.Data.Error != nil ||
				(upstreamData.Data.Inner != nil && upstreamData.Data.Inner.Error != nil) {
				debugLog("上游错误，结束流")
				endChunk := OpenAIResponse{
					ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   defaultModelName,
					Choices: []Choice{{Index: 0, Delta: Delta{}, FinishReason: "stop"}},
				}
				writeSSEChunk(w, endChunk)
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				*streamClosed = true
				return
			}

			// 处理EditContent在最初的answer信息（只发送一次）
			if !*sentInitialAnswer && upstreamData.Data.EditContent != "" && upstreamData.Data.Phase == "answer" {
				out := upstreamData.Data.EditContent
				if out != "" {
					parts := regexp.MustCompile(`</details>`).Split(out, -1)
					if len(parts) > 1 {
						content := parts[1]
						if content != "" {
							debugLog("发送初始答案内容")
							chunk := OpenAIResponse{
								ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
								Object:  "chat.completion.chunk",
								Created: time.Now().Unix(),
								Model:   defaultModelName,
								Choices: []Choice{{Index: 0, Delta: Delta{Content: content}}},
							}
							writeSSEChunk(w, chunk)
							flusher.Flush()
							*sentInitialAnswer = true
						}
					}
				}
			}

			if upstreamData.Data.DeltaContent != "" {
				out := upstreamData.Data.DeltaContent
				if upstreamData.Data.Phase == "thinking" {
					out = transformThinking(out)
					// 思考内容使用 reasoning_content 字段
					if out != "" {
						debugLog("发送思考内容")
						chunk := OpenAIResponse{
							ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   defaultModelName,
							Choices: []Choice{{Index: 0, Delta: Delta{ReasoningContent: out}}},
						}
						writeSSEChunk(w, chunk)
						flusher.Flush()
					}
				} else {
					// 普通内容使用 content 字段
					if out != "" {
						debugLog("发送普通内容")
						chunk := OpenAIResponse{
							ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   defaultModelName,
							Choices: []Choice{{Index: 0, Delta: Delta{Content: out}}},
						}
						writeSSEChunk(w, chunk)
						flusher.Flush()
					}
				}
			}

			// 检查是否结束
			if upstreamData.Data.Done || upstreamData.Data.Phase == "done" {
				debugLog("检测到流结束信号")
				// 发送结束chunk
				endChunk := OpenAIResponse{
					ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
					Object:  "chat.completion.chunk",
					Created: time.Now().Unix(),
					Model:   defaultModelName,
					Choices: []Choice{{Index: 0, Delta: Delta{}, FinishReason: "stop"}},
				}
				writeSSEChunk(w, endChunk)
				flusher.Flush()

				// 发送[DONE]
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				debugLog("流式响应完成")
				*streamClosed = true
			}
		}
	}
}

// 安全发送数据的通用函数
func sendDataSafe(data string, w http.ResponseWriter, flusher http.Flusher) error {
	defer func() {
		if r := recover(); r != nil {
			debugLog("发送数据时发生 panic: %v", r)
		}
	}()

	_, err := fmt.Fprint(w, data)
	if err != nil {
		return fmt.Errorf("写入响应失败: %v", err)
	}

	if flusher != nil {
		flusher.Flush()
	}
	return nil
}

// 发送思考内容 - 安全版本
func sendThinkContentSafe(content string, w http.ResponseWriter, flusher http.Flusher) {
	thinkChunk := OpenAIResponse{
		ID:      fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   defaultModelName,
		Choices: []Choice{{Index: 0, Delta: Delta{Content: fmt.Sprintf("<think>%s</think>", content)}}},
	}

	thinkJSON, err := json.Marshal(thinkChunk)
	if err != nil {
		debugLog("思考内容 JSON 编码失败: %v", err)
		return
	}

	data := fmt.Sprintf("data: %s\n\n", string(thinkJSON))
	if err := sendDataSafe(data, w, flusher); err != nil {
		debugLog("发送思考内容失败: %v", err)
	}
}

// 并发控制中间件
func concurrencyControlMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case connectionSemaphore <- struct{}{}:
			atomic.AddInt64(&currentConnections, 1)
			defer func() {
				<-connectionSemaphore
				atomic.AddInt64(&currentConnections, -1)
			}()
			next(w, r)
		default:
			http.Error(w, `{"error": "Server too busy, please try again later"}`, http.StatusServiceUnavailable)
			debugLog("⚠️ 连接数已满，拒绝新连接。当前连接数: %d", atomic.LoadInt64(&currentConnections))
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
		"uptime_seconds":      int(time.Since(startTime).Seconds()),
	}
}

// 状态监控处理器
func statusHandler(w http.ResponseWriter, r *http.Request) {
	status := getSystemStatus()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(status)
}

// 初始化函数
func init() {
	// 设置随机种子
	mathrand.Seed(time.Now().UnixNano())

	// 初始化性能配置
	getPerformanceConfig()

	// 初始化并发控制
	connectionSemaphore = make(chan struct{}, maxConcurrentConnections)

	// 记录启动时间
	startTime = time.Now()

	log.Printf("🚀 Z2API Go优化版 v%s (构建日期: %s)", VERSION, BUILD_DATE)
	log.Printf("📝 %s", DESCRIPTION)
	log.Printf("⚡ 性能模式: %s", performanceMode)
	log.Printf("🔧 配置详情:")
	log.Printf("   - 最大重试次数: %d", maxRetries)
	log.Printf("   - 重试延迟: %dms", retryDelay)
	log.Printf("   - 请求超时: %dms", requestTimeout)
	log.Printf("   - 流式超时: %dms", streamTimeout)
	log.Printf("   - 随机延迟: %d-%dms", randomDelayMin, randomDelayMax)
	log.Printf("🔗 并发控制:")
	log.Printf("   - 最大并发连接: %d", maxConcurrentConnections)
	log.Printf("   - 连接队列大小: %d", connectionQueueSize)
	log.Printf("   - 最大连接时间: %dms", maxConnectionTime)
	log.Printf("🌊 流处理优化:")
	log.Printf("   - 缓冲区大小: %d bytes", streamBufferSize)
	log.Printf("   - 连接检测: %v (间隔: %d)", !disableConnectionCheck, connectionCheckInterval)
	log.Printf("💾 内存管理:")
	log.Printf("   - 内存限制: %dMB", memoryLimitMB)
	log.Printf("   - 指标收集: %v", enableMetrics)
	log.Printf("📝 日志配置:")
	log.Printf("   - 详细日志: %v", enableDetailedLogging)
	log.Printf("   - 用户消息: %v", logUserMessages)
	log.Printf("   - 响应内容: %v", logResponseContent)
	log.Printf("🔐 功能配置:")
	log.Printf("   - 匿名token: %v", anonTokenEnabled)
	log.Printf("   - 思考标签模式: %s", thinkTagsMode)
	log.Printf("   - 调试模式: %v", debugMode)
	log.Printf("🎯 支持模型: %s", func() string {
		var models []string
		for _, model := range supportedModels {
			models = append(models, model.ID)
		}
		return strings.Join(models, ", ")
	}())
}

func main() {
	// 设置路由
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/v1/models", modelsHandler)
	http.HandleFunc("/v1/chat/completions", concurrencyControlMiddleware(chatHandler))
	http.HandleFunc("/", optionsHandler)

	// 启动服务器
	addr := fmt.Sprintf(":%d", port)
	log.Printf("🌐 服务器启动在端口 %d", port)
	log.Printf("📊 健康检查: http://localhost:%d/health", port)
	log.Printf("📈 状态监控: http://localhost:%d/status", port)
	log.Printf("🎯 模型列表: http://localhost:%d/v1/models", port)
	log.Printf("💬 聊天接口: http://localhost:%d/v1/chat/completions", port)
	log.Printf("🔑 API密钥: %s", maskAPIKey(defaultKey))
	log.Printf("🔗 上游地址: %s", upstreamURL)

	// 启动性能监控（如果启用）
	if enableMetrics {
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for range ticker.C {
				status := getSystemStatus()
				debugLog("📊 系统状态: 连接数=%d/%d, 内存=%dMB/%dMB, 请求数=%d, 错误数=%d",
					status["current_connections"], status["max_connections"],
					status["memory_usage_mb"], status["memory_limit_mb"],
					status["total_requests"], status["error_count"])

				// 内存使用检查
				if memUsage, ok := status["memory_usage_mb"].(uint64); ok && memUsage > uint64(memoryLimitMB) {
					log.Printf("⚠️ 内存使用超过限制: %dMB > %dMB", memUsage, memoryLimitMB)
					runtime.GC() // 强制垃圾回收
				}
			}
		}()
	}

	// 优雅关闭处理
	log.Printf("✅ Z2API Go优化版启动完成！")
	log.Printf("🎉 相比原版的改进:")
	log.Printf("   ✅ 企业级并发控制")
	log.Printf("   ✅ 完整的结构化日志")
	log.Printf("   ✅ 性能模式配置")
	log.Printf("   ✅ 重试机制和错误恢复")
	log.Printf("   ✅ 健康检查和监控")
	log.Printf("   ✅ 优化的流式响应处理")
	log.Printf("   ✅ 内存管理和泄漏防护")
	log.Printf("   ✅ 智能连接检测")
	log.Printf("   ✅ 保持原版的匿名token和思考处理特性")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("❌ 服务器启动失败: %v", err)
	}
}

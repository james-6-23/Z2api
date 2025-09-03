import { serve } from "https://deno.land/std@0.224.0/http/server.ts";

// 类型定义
interface ChatMessage {
  role: string;
  content: string;
}

interface ChatRequest {
  model: string;
  messages: ChatMessage[];
  stream?: boolean;
  temperature?: number;
  max_tokens?: number;
}

interface Delta {
  content?: string | null;
  reasoning_content?: string | null;
}

interface Choice {
  delta: Delta;
}

interface StreamResponse {
  choices: Choice[];
}

// 日志系统类型定义
type LogLevel = "INFO" | "WARN" | "ERROR";

interface RequestLog {
  request_id: string;
  timestamp: string;
  level: LogLevel;
  type: "request";
  client_ip: string;
  api_key: string;
  model: string;
  messages?: ChatMessage[];
  parameters?: Record<string, any>;
  user_agent?: string;
}

interface ResponseLog {
  request_id: string;
  timestamp: string;
  level: LogLevel;
  type: "response";
  status_code: number;
  response_time_ms: number;
  endpoint: string;
  retry_count: number;
  content?: any;
  reasoning_content?: string;
  error?: string;
}

interface StreamLog {
  request_id: string;
  timestamp: string;
  level: LogLevel;
  type: "stream";
  content?: any;
  delta?: any;
}

// 配置常量
const DEEPINFRA_URL = "https://api.deepinfra.com/v1/openai/chat/completions";
const PORT = parseInt(Deno.env.get("PORT") || "8000");

// 性能模式配置
const PERFORMANCE_MODE = Deno.env.get("PERFORMANCE_MODE") || "balanced";

// 根据性能模式设置参数
const getPerformanceConfig = () => {
  const mode = PERFORMANCE_MODE.toLowerCase();
  
  switch (mode) {
    case "fast":
      return {
        maxRetries: parseInt(Deno.env.get("MAX_RETRIES") || "1"),
        retryDelay: parseInt(Deno.env.get("RETRY_DELAY") || "200"),
        requestTimeout: parseInt(Deno.env.get("REQUEST_TIMEOUT") || "10000"),
        randomDelayMin: parseInt(Deno.env.get("RANDOM_DELAY_MIN") || "0"),
        randomDelayMax: parseInt(Deno.env.get("RANDOM_DELAY_MAX") || "100")
      };
    case "secure":
      return {
        maxRetries: parseInt(Deno.env.get("MAX_RETRIES") || "5"),
        retryDelay: parseInt(Deno.env.get("RETRY_DELAY") || "2000"),
        requestTimeout: parseInt(Deno.env.get("REQUEST_TIMEOUT") || "60000"),
        randomDelayMin: parseInt(Deno.env.get("RANDOM_DELAY_MIN") || "500"),
        randomDelayMax: parseInt(Deno.env.get("RANDOM_DELAY_MAX") || "1500")
      };
    default: // balanced
      return {
        maxRetries: parseInt(Deno.env.get("MAX_RETRIES") || "3"),
        retryDelay: parseInt(Deno.env.get("RETRY_DELAY") || "1000"),
        requestTimeout: parseInt(Deno.env.get("REQUEST_TIMEOUT") || "30000"),
        randomDelayMin: parseInt(Deno.env.get("RANDOM_DELAY_MIN") || "100"),
        randomDelayMax: parseInt(Deno.env.get("RANDOM_DELAY_MAX") || "500")
      };
  }
};

const config = getPerformanceConfig();
const MAX_RETRIES = config.maxRetries;
const RETRY_DELAY = config.retryDelay;
const REQUEST_TIMEOUT = config.requestTimeout;

// 支持多个镜像端点进行负载均衡
const getApiEndpoints = (): string[] => {
  const mirrors = Deno.env.get("DEEPINFRA_MIRRORS");
  if (mirrors) {
    return mirrors.split(",").map(url => url.trim());
  }
  return [DEEPINFRA_URL];
};

const API_ENDPOINTS = getApiEndpoints();

// 随机延迟函数，避免请求太频繁
const randomDelay = () => {
  const min = config.randomDelayMin;
  const max = config.randomDelayMax;
  const delay = Math.random() * (max - min) + min;
  return new Promise(resolve => setTimeout(resolve, delay));
};

// 带重试和多端点的请求函数
const fetchWithRetry = async (options: RequestInit, retries = MAX_RETRIES): Promise<Response> => {
  let lastError: Error | null = null;
  
  for (let endpointIndex = 0; endpointIndex < API_ENDPOINTS.length; endpointIndex++) {
    const endpoint = API_ENDPOINTS[endpointIndex];
    
    for (let i = 0; i < retries; i++) {
      try {
        // 添加随机延迟
        if (i > 0 || endpointIndex > 0) {
          await new Promise(resolve => setTimeout(resolve, RETRY_DELAY * Math.pow(2, i)));
        }
        
        await randomDelay();
        
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT);
        
        console.log(`尝试请求端点: ${endpoint} (第${endpointIndex + 1}个端点, 第${i + 1}次尝试)`);
        
        const response = await fetch(endpoint, {
          ...options,
          signal: controller.signal
        });
        
        clearTimeout(timeoutId);
        
        if (response.ok) {
          console.log(`请求成功: ${endpoint}`);
          return response;
        }
        
        // 如果是限流或封禁错误，等待更长时间
        if (response.status === 429 || response.status === 403) {
          const waitTime = Math.min(RETRY_DELAY * Math.pow(2, i), 10000);
          console.warn(`端点 ${endpoint} 被限流或封禁 (${response.status})，等待 ${waitTime}ms 后重试...`);
          await new Promise(resolve => setTimeout(resolve, waitTime));
          continue;
        }
        
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        
      } catch (error) {
        lastError = error instanceof Error ? error : new Error('未知错误');
        console.warn(`端点 ${endpoint} 请求尝试 ${i + 1}/${retries} 失败:`, lastError.message);
        
        if (i === retries - 1) {
          console.warn(`端点 ${endpoint} 所有重试都失败，尝试下一个端点`);
          break;
        }
      }
    }
  }
  
  throw lastError || new Error('所有端点和重试都失败');
};

// API Key 配置
const getValidApiKeys = (): string[] => {
  const keys = Deno.env.get("VALID_API_KEYS") || "linux.do";
  return keys.split(",").map(key => key.trim());
};

const VALID_API_KEYS = getValidApiKeys();

// 日志配置
const ENABLE_DETAILED_LOGGING = Deno.env.get("ENABLE_DETAILED_LOGGING") !== "false";
const LOG_USER_MESSAGES = Deno.env.get("LOG_USER_MESSAGES") === "true";
const LOG_RESPONSE_CONTENT = Deno.env.get("LOG_RESPONSE_CONTENT") === "true";

// 日志系统函数
function generateRequestId(): string {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return `req_${Array.from(bytes, b => b.toString(16).padStart(2, '0')).join('')}`;
}

function maskApiKey(apiKey: string): string {
  if (apiKey.length <= 8) {
    return "*".repeat(apiKey.length);
  }
  return apiKey.slice(0, 4) + "*".repeat(apiKey.length - 8) + apiKey.slice(-4);
}

function getClientIp(request: Request): string {
  // 检查 X-Forwarded-For 头
  const xff = request.headers.get("X-Forwarded-For");
  if (xff) {
    const ips = xff.split(",");
    if (ips.length > 0) {
      return ips[0].trim();
    }
  }

  // 检查 X-Real-IP 头
  const xri = request.headers.get("X-Real-IP");
  if (xri) {
    return xri;
  }

  // 默认返回未知
  return "unknown";
}

function logStructured(data: any): void {
  if (!ENABLE_DETAILED_LOGGING) {
    return;
  }

  try {
    console.log(JSON.stringify(data));
  } catch (error) {
    console.error("日志序列化失败:", error);
  }
}

function logRequest(
  requestId: string,
  clientIp: string,
  apiKey: string,
  model: string,
  messageCount: number,
  parameters?: Record<string, any>,
  userAgent?: string
): void {
  if (!ENABLE_DETAILED_LOGGING) {
    return;
  }

  const requestLog: RequestLog = {
    request_id: requestId,
    timestamp: new Date().toISOString(),
    level: "INFO",
    type: "request",
    client_ip: clientIp,
    api_key: maskApiKey(apiKey),
    model: model,
    user_agent: userAgent,
  };

  // 只记录消息数量，不记录具体内容
  requestLog.parameters = {
    message_count: messageCount,
    parameters: parameters,
  };

  logStructured(requestLog);
}

function logResponse(
  requestId: string,
  statusCode: number,
  responseTime: number,
  endpoint: string,
  retryCount: number,
  error?: string
): void {
  if (!ENABLE_DETAILED_LOGGING) {
    return;
  }

  let level: LogLevel = "INFO";
  if (statusCode >= 400) {
    level = "ERROR";
  } else if (statusCode >= 300) {
    level = "WARN";
  }

  const responseLog: ResponseLog = {
    request_id: requestId,
    timestamp: new Date().toISOString(),
    level: level,
    type: "response",
    status_code: statusCode,
    response_time_ms: responseTime,
    endpoint: endpoint,
    retry_count: retryCount,
    error: error,
  };

  // 不记录响应内容，只记录技术指标

  logStructured(responseLog);
}

// logStream 函数已移除，不再记录流式内容

// 支持的模型列表
const SUPPORTED_MODELS = [
  { id: "openai/gpt-oss-120b", object: "model" },
  { id: "moonshotai/Kimi-K2-Instruct", object: "model" },
  { id: "zai-org/GLM-4.5", object: "model" },
  { id: "zai-org/GLM-4.5-Air", object: "model" },
  { id: "Qwen/Qwen3-Coder-480B-A35B-Instruct-Turbo", object: "model" },
  { id: "deepseek-ai/DeepSeek-R1-0528-Turbo", object: "model" },
  { id: "deepseek-ai/DeepSeek-V3-0324-Turbo", object: "model" },
  { id: "deepseek-ai/DeepSeek-V3.1", object: "model" },
  { id: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-Turbo", object: "model" }
];

console.log(`🚀 DeepInfra API Proxy started on port ${PORT}`);
console.log(`⚡ Performance mode: ${PERFORMANCE_MODE}`);
console.log(`🔧 Config: retries=${MAX_RETRIES}, delay=${RETRY_DELAY}ms, timeout=${REQUEST_TIMEOUT}ms`);
console.log(`⏱️  Random delay: ${config.randomDelayMin}-${config.randomDelayMax}ms`);

// 性能统计
let requestCount = 0;
let totalResponseTime = 0;
let errorCount = 0;

// 主处理函数
async function handler(req: Request): Promise<Response> {
  const url = new URL(req.url);

  // 健康检查接口
  if (req.method === "GET" && url.pathname === "/health") {
    const stats = {
      status: "ok",
      timestamp: new Date().toISOString(),
      performance_mode: PERFORMANCE_MODE,
      config: {
        max_retries: MAX_RETRIES,
        retry_delay: RETRY_DELAY,
        request_timeout: REQUEST_TIMEOUT,
        random_delay: `${config.randomDelayMin}-${config.randomDelayMax}ms`
      },
      stats: {
        total_requests: requestCount,
        average_response_time: requestCount > 0 ? Math.round(totalResponseTime / requestCount) : 0,
        error_rate: requestCount > 0 ? Math.round((errorCount / requestCount) * 100) : 0
      }
    };
    
    return new Response(JSON.stringify(stats, null, 2), {
      status: 200,
      headers: { "Content-Type": "application/json" }
    });
  }

  // 模型列表接口
  if (req.method === "GET" && url.pathname === "/v1/models") {
    return new Response(JSON.stringify({
      object: "list",
      data: SUPPORTED_MODELS
    }), {
      status: 200,
      headers: { "Content-Type": "application/json" }
    });
  }

  // 聊天完成接口
  if (req.method === "POST" && url.pathname === "/v1/chat/completions") {
    const startTime = Date.now();
    requestCount++;

    // 生成请求 ID 和获取客户端信息
    const requestId = generateRequestId();
    const clientIp = getClientIp(req);
    const userAgent = req.headers.get("User-Agent") || "";

    const body = await req.text();
    const headers = new Headers(req.headers);

    // API Key 验证
    const auth = headers.get("Authorization");
    const key = auth?.replace("Bearer ", "").trim();
    if (!key || !VALID_API_KEYS.includes(key)) {
      const responseTime = Date.now() - startTime;
      logResponse(requestId, 401, responseTime, "", 0, "Unauthorized");
      return new Response(JSON.stringify({ error: "Unauthorized" }), {
        status: 401,
        headers: { "Content-Type": "application/json" }
      });
    }

    // 解析请求体
    let parsed: ChatRequest;
    try {
      parsed = JSON.parse(body) as ChatRequest;
    } catch (error) {
      const responseTime = Date.now() - startTime;
      logResponse(requestId, 400, responseTime, "", 0, "Invalid JSON format");
      return new Response(JSON.stringify({ error: "Invalid JSON format" }), {
        status: 400,
        headers: { "Content-Type": "application/json" }
      });
    }

    // 记录请求日志
    const parameters = {
      stream: parsed.stream,
      temperature: parsed.temperature,
      max_tokens: parsed.max_tokens,
    };
    logRequest(requestId, clientIp, key, parsed.model, parsed.messages.length, parameters, userAgent);

    const isStream = parsed.stream === true;

    // 构造更真实的请求头，避免被识别为机器人
    const userAgents = [
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
      "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0"
    ];
    
    const randomUserAgent = userAgents[Math.floor(Math.random() * userAgents.length)];
    
    const forwardHeaders: HeadersInit = {
      "Content-Type": "application/json",
      "User-Agent": randomUserAgent,
      "Accept": "text/event-stream, application/json, text/plain, */*",
      "Accept-Language": "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
      "Accept-Encoding": "gzip, deflate, br",
      "Origin": "https://deepinfra.com",
      "Referer": "https://deepinfra.com/",
      "Sec-CH-UA": '"Not_A Brand";v="8", "Chromium";v="120", "Google Chrome";v="120"',
      "Sec-CH-UA-Mobile": "?0",
      "Sec-CH-UA-Platform": '"Windows"',
      "Sec-Fetch-Dest": "empty",
      "Sec-Fetch-Mode": "cors",
      "Sec-Fetch-Site": "same-origin",
      "X-Requested-With": "XMLHttpRequest",
      "Cache-Control": "no-cache",
      "Pragma": "no-cache"
    };

    // 请求 DeepInfra API（使用多端点重试机制）
    let response: Response;
    try {
      response = await fetchWithRetry({
        method: "POST",
        headers: forwardHeaders,
        body
      });
    } catch (error) {
      errorCount++;
      const responseTime = Date.now() - startTime;
      totalResponseTime += responseTime;

      const errorMsg = error instanceof Error ? error.message : "未知错误";
      logResponse(requestId, 502, responseTime, "all_endpoints", MAX_RETRIES, errorMsg);
      console.error('DeepInfra API 所有端点请求失败:', error);
      return new Response(JSON.stringify({
        error: "External API request failed",
        details: errorMsg,
        retry_after: 60, // 建议客户端 60 秒后重试
        available_endpoints: API_ENDPOINTS.length,
        performance_mode: PERFORMANCE_MODE
      }), {
        status: 502,
        headers: { "Content-Type": "application/json" }
      });
    }

    // 非流式响应
    if (!isStream) {
      const result = await response.text();
      const responseTime = Date.now() - startTime;
      totalResponseTime += responseTime;

      logResponse(requestId, response.status, responseTime, "deepinfra_api", 0, undefined);
      console.log(`✅ 请求完成: ${responseTime}ms`);

      return new Response(result, {
        status: response.status,
        headers: { "Content-Type": "application/json" }
      });
    }

    // 流式响应处理
    const stream = new ReadableStream({
      async start(controller) {
        try {
          const reader = response.body?.getReader();
          if (!reader) {
            controller.close();
            return;
          }
          
          const decoder = new TextDecoder();
          let isInThinkBlock = false;
          let bufferedThinkContent = "";
          let streamClosed = false;

          while (true) {
            try {
              const { done, value } = await reader.read();
              if (done || streamClosed) break;
              
              const chunk = decoder.decode(value);
              const lines = chunk.split("\n");
              
              for (const line of lines) {
                if (streamClosed) break;
                
                if (line.startsWith("data: ")) {
                  const jsonText = line.slice(6).trim();
                  
                  if (jsonText === "[DONE]") {
                    // 发送缓存的思考内容
                    if (isInThinkBlock && bufferedThinkContent) {
                      try {
                        const output = `data: ${JSON.stringify({ choices: [{ delta: { content: `<think>${bufferedThinkContent}</think>` } }] })}\n\n`;
                        controller.enqueue(new TextEncoder().encode(output));
                      } catch (e) {
                        console.warn('发送思考内容失败:', e);
                      }
                    }
                    
                    try {
                      controller.enqueue(new TextEncoder().encode("data: [DONE]\n\n"));
                    } catch (e) {
                      console.warn('发送结束标记失败:', e);
                    }
                    streamClosed = true;
                    break;
                  }
                  
                  if (jsonText) {
                    try {
                      const parsed = JSON.parse(jsonText) as StreamResponse;
                      const delta = parsed.choices?.[0]?.delta;
                      
                      if (delta) {
                        let contentToSend: string | null = null;
                        
                        // 处理思考内容
                        if (delta.reasoning_content !== undefined && delta.reasoning_content !== null) {
                          if (delta.reasoning_content) {
                            bufferedThinkContent += delta.reasoning_content;
                          }
                          isInThinkBlock = true;
                        }
                        // 处理正常内容
                        else if (delta.content !== undefined && delta.content !== null) {
                          if (isInThinkBlock) {
                            // 发送思考内容
                            if (bufferedThinkContent) {
                              try {
                                const thinkOutput = `data: ${JSON.stringify({ choices: [{ delta: { content: `<think>${bufferedThinkContent}</think>` } }] })}\n\n`;
                                controller.enqueue(new TextEncoder().encode(thinkOutput));
                              } catch (e) {
                                console.warn('发送思考内容失败:', e);
                              }
                              bufferedThinkContent = "";
                            }
                            isInThinkBlock = false;
                          }
                          contentToSend = delta.content;
                        }
                        
                        // 发送正常内容
                        if (contentToSend !== null) {
                          try {
                            const output = `data: ${JSON.stringify({ choices: [{ delta: { content: contentToSend } }] })}\n\n`;
                            controller.enqueue(new TextEncoder().encode(output));
                          } catch (e) {
                            console.warn('发送内容失败:', e);
                            streamClosed = true;
                            break;
                          }
                        }
                      }
                    } catch (parseError) {
                      // 忽略 JSON 解析错误
                    }
                  }
                }
              }
            } catch (readError) {
              console.warn('读取数据失败:', readError);
              streamClosed = true;
              break;
            }
          }
        } catch (error) {
          console.error('流处理错误:', error);
        } finally {
          try {
            controller.close();
          } catch (closeError) {
            // 忽略关闭错误
          }
        }
      }
    });

    return new Response(stream, {
      status: 200,
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        "Connection": "keep-alive",
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization"
      }
    });
  }

  return new Response(JSON.stringify({ error: "Not Found" }), { 
    status: 404,
    headers: { "Content-Type": "application/json" }
  });
}

// 启动服务器
console.log(`🚀 DeepInfra API Proxy started on port ${PORT}`);
console.log(`⚡ Performance mode: ${PERFORMANCE_MODE}`);
console.log(`🔧 Config: retries=${MAX_RETRIES}, delay=${RETRY_DELAY}ms, timeout=${REQUEST_TIMEOUT}ms`);
console.log(`⏱️  Random delay: ${config.randomDelayMin}-${config.randomDelayMax}ms`);
console.log(`📝 Detailed logging: ${ENABLE_DETAILED_LOGGING}, User messages: ${LOG_USER_MESSAGES}, Response content: ${LOG_RESPONSE_CONTENT}`);

serve(handler, { port: PORT });
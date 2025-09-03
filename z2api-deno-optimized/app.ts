import { serve } from "https://deno.land/std@0.224.0/http/server.ts";

// 类型定义
interface ChatMessage {
  role: string;
  content: string;
  reasoning_content?: string;
}

interface OpenAIRequest {
  model: string;
  messages: ChatMessage[];
  stream?: boolean;
  temperature?: number;
  max_tokens?: number;
}

interface UpstreamRequest {
  stream: boolean;
  model: string;
  messages: ChatMessage[];
  params: Record<string, any>;
  features: Record<string, any>;
  background_tasks?: Record<string, boolean>;
  chat_id?: string;
  id?: string;
  mcp_servers?: string[];
  model_item?: {
    id: string;
    name: string;
    owned_by: string;
  };
  tool_servers?: string[];
  variables?: Record<string, string>;
}

interface Delta {
  role?: string;
  content?: string;
  reasoning_content?: string;
}

interface Choice {
  index: number;
  delta?: Delta;
  message?: ChatMessage;
  finish_reason?: string;
}

interface OpenAIResponse {
  id: string;
  object: string;
  created: number;
  model: string;
  choices: Choice[];
  usage?: {
    prompt_tokens: number;
    completion_tokens: number;
    total_tokens: number;
  };
}

interface UpstreamData {
  type: string;
  data: {
    delta_content: string;
    edit_content: string;
    phase: string;
    done: boolean;
    usage?: any;
    error?: {
      detail: string;
      code: number;
    };
    data?: {
      error?: {
        detail: string;
        code: number;
      };
    };
  };
  error?: {
    detail: string;
    code: number;
  };
}

interface Model {
  id: string;
  object: string;
  created: number;
  owned_by: string;
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

// 配置常量
const UPSTREAM_URL = "https://chat.z.ai/api/chat/completions";
const PORT = parseInt(Deno.env.get("PORT") || "8080");
const DEFAULT_KEY = Deno.env.get("DEFAULT_KEY") || "123456";
const UPSTREAM_TOKEN = Deno.env.get("UPSTREAM_TOKEN") || "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6Ijc3NWI4MjMyLTFjMDgtNDZjOC1iM2ZjLTc4NGZkOTYzOTFkMCIsImVtYWlsIjoiR3Vlc3QtMTc1NjQxNzIwODY2NkBndWVzdC5jb20ifQ.ANLFGzTOIhaocgsVRMtzhcHOfhvxWrf3RwiEV0b4mmeNMu72fIbp9j0D42aWlrupZN5AARqGPeIDUFU5po0gFQ";

// 模型配置
const DEFAULT_MODEL_NAME = "GLM-4.5";
const THINKING_MODEL_NAME = "GLM-4.5-Thinking";
const SEARCH_MODEL_NAME = "GLM-4.5-Search";

// 伪装前端头部
const X_FE_VERSION = "prod-fe-1.0.70";
const BROWSER_UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36 Edg/139.0.0.0";
const SEC_CH_UA = '"Not;A=Brand";v="99", "Microsoft Edge";v="139", "Chromium";v="139"';
const SEC_CH_UA_MOB = "?0";
const SEC_CH_UA_PLAT = '"Windows"';
const ORIGIN_BASE = "https://chat.z.ai";

// 性能配置
const PERFORMANCE_MODE = Deno.env.get("PERFORMANCE_MODE") || "balanced";
const ANON_TOKEN_ENABLED = Deno.env.get("ANON_TOKEN_ENABLED") !== "false";
const THINK_TAGS_MODE = Deno.env.get("THINK_TAGS_MODE") || "think";
const DEBUG_MODE = Deno.env.get("DEBUG_MODE") === "true";

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

// 日志配置
const ENABLE_DETAILED_LOGGING = Deno.env.get("ENABLE_DETAILED_LOGGING") !== "false";
const LOG_USER_MESSAGES = Deno.env.get("LOG_USER_MESSAGES") === "true";
const LOG_RESPONSE_CONTENT = Deno.env.get("LOG_RESPONSE_CONTENT") === "true";

// 支持的模型列表
const SUPPORTED_MODELS: Model[] = [
  { id: DEFAULT_MODEL_NAME, object: "model", created: Date.now(), owned_by: "z.ai" },
  { id: THINKING_MODEL_NAME, object: "model", created: Date.now(), owned_by: "z.ai" },
  { id: SEARCH_MODEL_NAME, object: "model", created: Date.now(), owned_by: "z.ai" }
];

// 性能统计
let requestCount = 0;
let totalResponseTime = 0;
let errorCount = 0;

// 随机延迟函数
const randomDelay = () => {
  const min = config.randomDelayMin;
  const max = config.randomDelayMax;
  const delay = Math.random() * (max - min) + min;
  return new Promise(resolve => setTimeout(resolve, delay));
};

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
  const xff = request.headers.get("X-Forwarded-For");
  if (xff) {
    const ips = xff.split(",");
    if (ips.length > 0) {
      return ips[0].trim();
    }
  }

  const xri = request.headers.get("X-Real-IP");
  if (xri) {
    return xri;
  }

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

function debugLog(format: string, ...args: any[]): void {
  if (DEBUG_MODE) {
    console.log(`[DEBUG] ${format}`, ...args);
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

  logStructured(responseLog);
}

// 获取匿名token（每次对话使用不同token，避免共享记忆）
async function getAnonymousToken(): Promise<string> {
  const controller = new AbortController();
  const timeoutId = setTimeout(() => controller.abort(), 10000);

  try {
    const response = await fetch(ORIGIN_BASE + "/api/v1/auths/", {
      method: "GET",
      headers: {
        "User-Agent": BROWSER_UA,
        "Accept": "*/*",
        "Accept-Language": "zh-CN,zh;q=0.9",
        "X-FE-Version": X_FE_VERSION,
        "sec-ch-ua": SEC_CH_UA,
        "sec-ch-ua-mobile": SEC_CH_UA_MOB,
        "sec-ch-ua-platform": SEC_CH_UA_PLAT,
        "Origin": ORIGIN_BASE,
        "Referer": ORIGIN_BASE + "/",
      },
      signal: controller.signal
    });

    clearTimeout(timeoutId);

    if (!response.ok) {
      throw new Error(`anon token status=${response.status}`);
    }

    const body = await response.json();
    if (!body.token) {
      throw new Error("anon token empty");
    }

    return body.token;
  } catch (error) {
    clearTimeout(timeoutId);
    throw error;
  }
}

// 思考内容转换函数
function transformThinking(s: string): string {
  // 去 <summary>…</summary>
  s = s.replace(/(?s)<summary>.*?<\/summary>/g, "");
  // 清理残留自定义标签，如 </thinking>、<Full> 等
  s = s.replace(/<\/thinking>/g, "");
  s = s.replace(/<Full>/g, "");
  s = s.replace(/<\/Full>/g, "");
  s = s.trim();

  switch (THINK_TAGS_MODE) {
    case "think":
      s = s.replace(/<details[^>]*>/g, "<think>");
      s = s.replace(/<\/details>/g, "</think>");
      break;
    case "strip":
      s = s.replace(/<details[^>]*>/g, "");
      s = s.replace(/<\/details>/g, "");
      break;
  }

  // 处理每行前缀 "> "（包括起始位置）
  s = s.replace(/^> /, "");
  s = s.replace(/\n> /g, "\n");
  return s.trim();
}

// 带重试的请求函数
async function fetchWithRetry(
  upstreamReq: UpstreamRequest,
  chatID: string,
  authToken: string,
  retries = MAX_RETRIES
): Promise<Response> {
  let lastError: Error | null = null;

  for (let i = 0; i < retries; i++) {
    try {
      // 添加延迟
      if (i > 0) {
        await new Promise(resolve => setTimeout(resolve, RETRY_DELAY * Math.pow(2, i)));
      }

      await randomDelay();

      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), REQUEST_TIMEOUT);

      debugLog(`尝试请求上游 (第${i + 1}次尝试)`);

      const response = await fetch(UPSTREAM_URL, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Accept": "application/json, text/event-stream",
          "User-Agent": BROWSER_UA,
          "Authorization": "Bearer " + authToken,
          "Accept-Language": "zh-CN",
          "sec-ch-ua": SEC_CH_UA,
          "sec-ch-ua-mobile": SEC_CH_UA_MOB,
          "sec-ch-ua-platform": SEC_CH_UA_PLAT,
          "X-FE-Version": X_FE_VERSION,
          "Origin": ORIGIN_BASE,
          "Referer": ORIGIN_BASE + "/c/" + chatID,
        },
        body: JSON.stringify(upstreamReq),
        signal: controller.signal
      });

      clearTimeout(timeoutId);

      if (response.ok) {
        debugLog("请求成功");
        return response;
      }

      // 如果是限流或封禁错误，等待更长时间
      if (response.status === 429 || response.status === 403) {
        const waitTime = Math.min(RETRY_DELAY * Math.pow(2, i), 10000);
        debugLog(`被限流或封禁 (${response.status})，等待 ${waitTime}ms 后重试...`);
        await new Promise(resolve => setTimeout(resolve, waitTime));
        continue;
      }

      throw new Error(`HTTP ${response.status}: ${response.statusText}`);

    } catch (error) {
      lastError = error instanceof Error ? error : new Error('未知错误');
      debugLog(`请求尝试 ${i + 1}/${retries} 失败:`, lastError.message);

      if (i === retries - 1) {
        debugLog("所有重试都失败");
        break;
      }
    }
  }

  throw lastError || new Error('所有重试都失败');
}

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
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization"
      }
    });
  }

  // 模型列表接口
  if (req.method === "GET" && url.pathname === "/v1/models") {
    return new Response(JSON.stringify({
      object: "list",
      data: SUPPORTED_MODELS
    }), {
      status: 200,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization"
      }
    });
  }

  // OPTIONS 处理
  if (req.method === "OPTIONS") {
    return new Response(null, {
      status: 200,
      headers: {
        "Access-Control-Allow-Origin": "*",
        "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
        "Access-Control-Allow-Headers": "Content-Type, Authorization",
        "Access-Control-Allow-Credentials": "true"
      }
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

    // API Key 验证
    const auth = req.headers.get("Authorization");
    const key = auth?.replace("Bearer ", "").trim();
    if (!key || key !== DEFAULT_KEY) {
      const responseTime = Date.now() - startTime;
      logResponse(requestId, 401, responseTime, "", 0, "Unauthorized");
      return new Response(JSON.stringify({ error: "Unauthorized" }), {
        status: 401,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*"
        }
      });
    }

    debugLog("API key验证通过");

    // 解析请求体
    let parsed: OpenAIRequest;
    try {
      parsed = JSON.parse(body) as OpenAIRequest;
    } catch (error) {
      const responseTime = Date.now() - startTime;
      logResponse(requestId, 400, responseTime, "", 0, "Invalid JSON format");
      return new Response(JSON.stringify({ error: "Invalid JSON format" }), {
        status: 400,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*"
        }
      });
    }

    debugLog(`请求解析成功 - 模型: ${parsed.model}, 流式: ${parsed.stream}, 消息数: ${parsed.messages.length}`);

    // 生成会话相关ID
    const chatID = `${Date.now()}-${Math.floor(Date.now() / 1000)}`;
    const msgID = `${Date.now()}`;

    let isThinking = false;
    let isSearch = false;
    let searchMcp = "";

    if (parsed.model === THINKING_MODEL_NAME) {
      isThinking = true;
    } else if (parsed.model === SEARCH_MODEL_NAME) {
      isThinking = true;
      isSearch = true;
      searchMcp = "deep-web-search";
    }

    // 记录请求日志
    const parameters = {
      stream: parsed.stream,
      temperature: parsed.temperature,
      max_tokens: parsed.max_tokens,
    };
    logRequest(requestId, clientIp, key, parsed.model, parsed.messages.length, parameters, userAgent);

    // 构造上游请求
    const upstreamReq: UpstreamRequest = {
      stream: true, // 总是使用流式从上游获取
      chat_id: chatID,
      id: msgID,
      model: "0727-360B-API", // 上游实际模型ID
      messages: parsed.messages,
      params: {},
      features: {
        enable_thinking: isThinking,
        web_search: isSearch,
        auto_web_search: isSearch,
      },
      background_tasks: {
        title_generation: false,
        tags_generation: false,
      },
      mcp_servers: [searchMcp],
      model_item: {
        id: "0727-360B-API",
        name: "GLM-4.5",
        owned_by: "openai"
      },
      tool_servers: [],
      variables: {
        "{{USER_NAME}}": "User",
        "{{USER_LOCATION}}": "Unknown",
        "{{CURRENT_DATETIME}}": new Date().toISOString().replace('T', ' ').slice(0, 19),
      },
    };

    // 选择本次对话使用的token
    let authToken = UPSTREAM_TOKEN;
    if (ANON_TOKEN_ENABLED) {
      try {
        const t = await getAnonymousToken();
        authToken = t;
        debugLog(`匿名token获取成功: ${t.length > 10 ? t.slice(0, 10) + "..." : t}`);
      } catch (error) {
        debugLog(`匿名token获取失败，回退固定token: ${error}`);
      }
    }

    const isStream = parsed.stream === true;

    // 请求上游API
    let response: Response;
    try {
      response = await fetchWithRetry(upstreamReq, chatID, authToken);
    } catch (error) {
      errorCount++;
      const responseTime = Date.now() - startTime;
      totalResponseTime += responseTime;

      const errorMsg = error instanceof Error ? error.message : "未知错误";
      logResponse(requestId, 502, responseTime, "upstream", MAX_RETRIES, errorMsg);
      console.error('上游API请求失败:', error);
      return new Response(JSON.stringify({
        error: "External API request failed",
        details: errorMsg,
        retry_after: 60,
        performance_mode: PERFORMANCE_MODE
      }), {
        status: 502,
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*"
        }
      });
    }

    // 处理响应
    if (!isStream) {
      return handleNonStreamResponse(response, requestId, startTime);
    } else {
      return handleStreamResponse(response, requestId, startTime);
    }
  }

  return new Response(JSON.stringify({ error: "Not Found" }), {
    status: 404,
    headers: {
      "Content-Type": "application/json",
      "Access-Control-Allow-Origin": "*"
    }
  });
}

// 处理非流式响应
async function handleNonStreamResponse(response: Response, requestId: string, startTime: number): Promise<Response> {
  try {
    // 收集完整响应
    let fullContent = "";
    const reader = response.body?.getReader();
    if (!reader) {
      throw new Error("无法获取响应流");
    }

    const decoder = new TextDecoder();

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      const chunk = decoder.decode(value);
      const lines = chunk.split("\n");

      for (const line of lines) {
        if (!line.startsWith("data: ")) continue;

        const dataStr = line.slice(6).trim();
        if (dataStr === "" || dataStr === "[DONE]") continue;

        try {
          const upstreamData = JSON.parse(dataStr) as UpstreamData;

          if (upstreamData.data.delta_content) {
            let out = upstreamData.data.delta_content;
            if (upstreamData.data.phase === "thinking") {
              out = transformThinking(out);
            }
            if (out) {
              fullContent += out;
            }
          }

          if (upstreamData.data.done || upstreamData.data.phase === "done") {
            break;
          }
        } catch (parseError) {
          // 忽略JSON解析错误
        }
      }
    }

    const responseTime = Date.now() - startTime;
    totalResponseTime += responseTime;

    logResponse(requestId, 200, responseTime, "upstream", 0, undefined);
    debugLog(`非流式响应完成: ${responseTime}ms`);

    // 构造完整响应
    const openaiResponse: OpenAIResponse = {
      id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
      object: "chat.completion",
      created: Math.floor(Date.now() / 1000),
      model: DEFAULT_MODEL_NAME,
      choices: [{
        index: 0,
        message: {
          role: "assistant",
          content: fullContent,
        },
        finish_reason: "stop",
      }],
      usage: {
        prompt_tokens: 0,
        completion_tokens: 0,
        total_tokens: 0,
      },
    };

    return new Response(JSON.stringify(openaiResponse), {
      status: 200,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*"
      }
    });
  } catch (error) {
    errorCount++;
    const responseTime = Date.now() - startTime;
    totalResponseTime += responseTime;

    const errorMsg = error instanceof Error ? error.message : "处理响应失败";
    logResponse(requestId, 500, responseTime, "upstream", 0, errorMsg);

    return new Response(JSON.stringify({ error: "Internal server error" }), {
      status: 500,
      headers: {
        "Content-Type": "application/json",
        "Access-Control-Allow-Origin": "*"
      }
    });
  }
}

// 处理流式响应
function handleStreamResponse(response: Response, requestId: string, startTime: number): Response {
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
        let sentInitialAnswer = false;

        // 发送第一个chunk（role）
        const firstChunk: OpenAIResponse = {
          id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
          object: "chat.completion.chunk",
          created: Math.floor(Date.now() / 1000),
          model: DEFAULT_MODEL_NAME,
          choices: [{
            index: 0,
            delta: { role: "assistant" },
          }],
        };

        const firstData = `data: ${JSON.stringify(firstChunk)}\n\n`;
        controller.enqueue(new TextEncoder().encode(firstData));

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
                      const thinkChunk: OpenAIResponse = {
                        id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                        object: "chat.completion.chunk",
                        created: Math.floor(Date.now() / 1000),
                        model: DEFAULT_MODEL_NAME,
                        choices: [{
                          index: 0,
                          delta: { content: `<think>${bufferedThinkContent}</think>` },
                        }],
                      };
                      const output = `data: ${JSON.stringify(thinkChunk)}\n\n`;
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
                    const upstreamData = JSON.parse(jsonText) as UpstreamData;

                    // 错误检测
                    if (upstreamData.error || upstreamData.data.error ||
                        (upstreamData.data.data && upstreamData.data.data.error)) {
                      debugLog("上游错误，结束流");
                      const endChunk: OpenAIResponse = {
                        id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                        object: "chat.completion.chunk",
                        created: Math.floor(Date.now() / 1000),
                        model: DEFAULT_MODEL_NAME,
                        choices: [{ index: 0, delta: {}, finish_reason: "stop" }],
                      };
                      const endData = `data: ${JSON.stringify(endChunk)}\n\n`;
                      controller.enqueue(new TextEncoder().encode(endData));
                      controller.enqueue(new TextEncoder().encode("data: [DONE]\n\n"));
                      streamClosed = true;
                      break;
                    }

                    // 处理EditContent在最初的answer信息（只发送一次）
                    if (!sentInitialAnswer && upstreamData.data.edit_content &&
                        upstreamData.data.phase === "answer") {
                      const out = upstreamData.data.edit_content;
                      if (out) {
                        const parts = out.split("</details>");
                        if (parts.length > 1) {
                          const content = parts[1];
                          if (content) {
                            debugLog("发送初始答案内容");
                            const chunk: OpenAIResponse = {
                              id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                              object: "chat.completion.chunk",
                              created: Math.floor(Date.now() / 1000),
                              model: DEFAULT_MODEL_NAME,
                              choices: [{ index: 0, delta: { content: content } }],
                            };
                            const chunkData = `data: ${JSON.stringify(chunk)}\n\n`;
                            controller.enqueue(new TextEncoder().encode(chunkData));
                            sentInitialAnswer = true;
                          }
                        }
                      }
                    }

                    if (upstreamData.data.delta_content) {
                      let out = upstreamData.data.delta_content;
                      if (upstreamData.data.phase === "thinking") {
                        out = transformThinking(out);
                        // 思考内容使用 reasoning_content 字段
                        if (out) {
                          debugLog("发送思考内容");
                          const chunk: OpenAIResponse = {
                            id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                            object: "chat.completion.chunk",
                            created: Math.floor(Date.now() / 1000),
                            model: DEFAULT_MODEL_NAME,
                            choices: [{
                              index: 0,
                              delta: { reasoning_content: out },
                            }],
                          };
                          const chunkData = `data: ${JSON.stringify(chunk)}\n\n`;
                          controller.enqueue(new TextEncoder().encode(chunkData));
                        }
                      } else {
                        // 普通内容使用 content 字段
                        if (out) {
                          debugLog("发送普通内容");
                          const chunk: OpenAIResponse = {
                            id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                            object: "chat.completion.chunk",
                            created: Math.floor(Date.now() / 1000),
                            model: DEFAULT_MODEL_NAME,
                            choices: [{
                              index: 0,
                              delta: { content: out },
                            }],
                          };
                          const chunkData = `data: ${JSON.stringify(chunk)}\n\n`;
                          controller.enqueue(new TextEncoder().encode(chunkData));
                        }
                      }
                    }

                    // 检查是否结束
                    if (upstreamData.data.done || upstreamData.data.phase === "done") {
                      debugLog("检测到流结束信号");
                      // 发送结束chunk
                      const endChunk: OpenAIResponse = {
                        id: `chatcmpl-${Math.floor(Date.now() / 1000)}`,
                        object: "chat.completion.chunk",
                        created: Math.floor(Date.now() / 1000),
                        model: DEFAULT_MODEL_NAME,
                        choices: [{
                          index: 0,
                          delta: {},
                          finish_reason: "stop",
                        }],
                      };
                      const endData = `data: ${JSON.stringify(endChunk)}\n\n`;
                      controller.enqueue(new TextEncoder().encode(endData));

                      // 发送[DONE]
                      controller.enqueue(new TextEncoder().encode("data: [DONE]\n\n"));
                      debugLog("流式响应完成");
                      streamClosed = true;
                      break;
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

// 启动服务器
console.log(`🚀 Z2API Deno优化版启动在端口 ${PORT}`);
console.log(`⚡ 性能模式: ${PERFORMANCE_MODE}`);
console.log(`🔧 配置: retries=${MAX_RETRIES}, delay=${RETRY_DELAY}ms, timeout=${REQUEST_TIMEOUT}ms`);
console.log(`⏱️  随机延迟: ${config.randomDelayMin}-${config.randomDelayMax}ms`);
console.log(`📝 详细日志: ${ENABLE_DETAILED_LOGGING}, 用户消息: ${LOG_USER_MESSAGES}, 响应内容: ${LOG_RESPONSE_CONTENT}`);
console.log(`🔐 匿名token: ${ANON_TOKEN_ENABLED}, 思考标签模式: ${THINK_TAGS_MODE}`);
console.log(`🎯 支持模型: ${SUPPORTED_MODELS.map(m => m.id).join(", ")}`);

serve(handler, { port: PORT });

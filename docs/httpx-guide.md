# httpx

通用、易用的 Go HTTP Client 库，基于 Functional Options 模式 + 中间件洋葱模型。

## 快速开始

```go
import "github.com/llbox/x-one/pkg/httpx"

client := httpx.New(
    httpx.WithBaseURL("https://api.example.com"),
    httpx.WithTimeout(10 * time.Second),
    httpx.WithRetry(3, time.Second),
)

// GET
resp, err := client.Get(ctx, "/users", httpx.WithQuery("page", "1"))

// POST (struct/map 自动序列化为 JSON)
resp, err := client.Post(ctx, "/users", User{Name: "Alice"})

// 解析响应
var user User
resp.JSON(&user)
```

## Client 选项

| 选项 | 说明 |
|------|------|
| `WithBaseURL(url)` | 基础 URL |
| `WithTimeout(d)` | 全局请求超时 |
| `WithDefaultHeader(k, v)` | 全局默认 header |
| `WithHTTPClient(hc)` | 复用自定义 `*http.Client` |
| `WithUserAgent(ua)` | 设置 User-Agent |
| `WithMiddleware(mw...)` | 注册中间件 |
| `WithRetry(max, backoff)` | 重试配置（只重试 5xx 和网络错误，指数退避） |
| `WithDebug(w)` | 输出请求/响应日志到 `io.Writer` |

## 请求选项

| 选项 | 说明 |
|------|------|
| `WithQuery(k, v)` | 添加 query 参数 |
| `WithQueryParams(map)` | 批量添加 query 参数 |
| `WithHeader(k, v)` | 本次请求 header |
| `WithRequestTimeout(d)` | 覆盖本次请求超时 |
| `WithContext(ctx)` | 覆盖 context |

## Body 类型处理

`Post` / `Put` / `Patch` 的 `body any` 参数按类型自动识别：

| 类型 | Content-Type |
|------|-------------|
| `struct` / `map` | `application/json` |
| `string` | `text/plain` |
| `[]byte` | `application/octet-stream` |
| `io.Reader` | `application/octet-stream` |
| `url.Values` | `application/x-www-form-urlencoded` |
| `*httpx.FormData` | `multipart/form-data` |

## Form Data

### URL-encoded 表单

```go
client.Post(ctx, "/login", url.Values{
    "username": {"alice"},
    "password": {"secret"},
})
```

### Multipart 表单（含文件上传）

```go
form := httpx.NewFormData().
    Set("name", "Alice").
    SetFile("avatar", "/path/to/photo.png")

resp, err := client.Post(ctx, "/upload", form)
```

从内存上传文件：

```go
form := httpx.NewFormData().
    SetFileReader("avatar", "photo.png", bytes.NewReader(imageBytes))

resp, err := client.Post(ctx, "/upload", form)
```

## 自定义中间件

### 请求签名

```go
func SigningMiddleware(secret string) httpx.Middleware {
    return func(next httpx.Handler) httpx.Handler {
        return func(req *httpx.Request) (*httpx.Response, error) {
            bodyBytes, _ := json.Marshal(req.Body)
            mac := hmac.New(sha256.New, []byte(secret))
            mac.Write(bodyBytes)
            sig := hex.EncodeToString(mac.Sum(nil))

            if req.Header == nil {
                req.Header = http.Header{}
            }
            req.Header.Set("X-Signature", sig)
            return next(req)
        }
    }
}

client := httpx.New(
    httpx.WithBaseURL("https://api.example.com"),
    httpx.WithMiddleware(SigningMiddleware("my-secret")),
)
```

### 链路追踪

```go
func TracingMiddleware() httpx.Middleware {
    return func(next httpx.Handler) httpx.Handler {
        return func(req *httpx.Request) (*httpx.Response, error) {
            if req.Header == nil {
                req.Header = http.Header{}
            }
            req.Header.Set("X-Trace-ID", uuid.New().String())
            return next(req)
        }
    }
}
```

## 错误处理

```go
resp, err := client.Get(ctx, "/users")
if err != nil {
    if httpx.IsTimeout(err) {
        // 处理超时
    }
    if httpx.IsHTTPError(err) {
        // 处理 HTTP 错误
    }
    if httpx.IsNetworkError(err) {
        // 处理网络错误
    }
}
if resp.IsError() {
    fmt.Println("server returned", resp.Status())
}
```

## 内置中间件

| 中间件 | 说明 |
|--------|------|
| `RetryMiddleware(max, backoff)` | 指数退避重试 |
| `LoggingMiddleware(w)` | 请求/响应日志 |
| `AuthMiddleware(scheme, value)` | 自动注入 Authorization 头 |

## 组织建议

多个 API 接口建议按资源拆 Service：

```go
type API struct {
    Users  *UserService
    Orders *OrderService
}

func NewAPI(baseURL string) *API {
    c := httpx.New(httpx.WithBaseURL(baseURL))
    return &API{
        Users:  &UserService{c},
        Orders: &OrderService{c},
    }
}

type UserService struct {
    c *httpx.Client
}

func (s *UserService) List(ctx context.Context) ([]User, error) {
    var users []User
    _, err := s.c.Get(ctx, "/users")
    return users, err
}
```

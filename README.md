# x-one

[![Go Reference](https://pkg.go.dev/badge/github.com/llbox/x-one.svg)](https://pkg.go.dev/github.com/llbox/x-one)

Go 通用工具库。

## Packages

### [httpx](pkg/httpx/) — HTTP Client

通用、易用的 HTTP Client，支持 Functional Options 配置、中间件洋葱模型、自动序列化、重试、文件上传。

```go
import "github.com/llbox/x-one/pkg/httpx"

client := httpx.New(
    httpx.WithBaseURL("https://api.example.com"),
    httpx.WithTimeout(10 * time.Second),
)

resp, _ := client.Get(ctx, "/users", httpx.WithQuery("page", "1"))
resp, _ := client.Post(ctx, "/users", User{Name: "Alice"})
```

更多用法见 [docs/httpx-guide.md](docs/httpx-guide.md)。

## Install

```bash
go get github.com/llbox/x-one@master
```

## License

MIT

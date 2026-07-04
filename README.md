# eventbus-starter

基于 [`github.com/kordar/eventbus`](../../packages/eventbus) 的 Starter 模块，以配置驱动方式在应用容器中创建和管理多个 `EventBus` 实例。

## 子模块

| 目录 | 说明 |
|------|------|
| [fx](./fx) | Uber Fx 集成，实现 `GoCfgModule` 接口，按配置创建 Named EventBus 实例并注入容器 |

## 快速接入 (Fx)

```go
import (
    gocfgmodulefx "github.com/kordar/gocfg-load-module/fx/v2"
    eventbusstarter "github.com/kordar/eventbus-starter/fx/v2"
)

gocfgmodulefx.Register(eventbusstarter.StarterModule("eventbus"))
```

配置后通过命名注入或全局 API 获取：

```go
// Fx 命名注入
type MyService struct {
    fx.In
    Bus *eventbus.EventBus `name:"eventbus.sys"`
}

// 全局 API
bus := eventbus.Get("sys")
```

详细配置参考与使用示例见 [fx/README.md](./fx/README.md)。

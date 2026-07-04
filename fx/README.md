# eventbus-starter/fx

基于 `go.uber.org/fx` 的事件总线 starter，实现 `gocfg-load-module/fx` 的 `GoCfgModule` 接口，按配置创建多个命名 `EventBus` 实例并注入到 Fx 容器。

## 快速接入

### 1. 注册 starter

```go
import (
    gocfgmodulefx "github.com/kordar/gocfg-load-module/fx/v2"
    eventbusstarter "github.com/kordar/eventbus-starter/fx/v2"
)

gocfgmodulefx.Register(eventbusstarter.StarterModule("eventbus"))
```

### 2. 注入 EventBus

```go
type MyService struct {
    fx.In
    SysBus *eventbus.EventBus `name:"eventbus.sys"`
}
```

命名规则：`name:"eventbus.<id>"`

### 3. 全局 API

```go
bus := eventbus.Get("sys")
```

---

## 配置方式

### 单实例

```ini
[eventbus]
id = "default"
async_task = on
async_task_work_size = 5
async_task_queue_buff_len = 30
```

### 多实例

```ini
[eventbus.sys]
async_task = on
async_task_work_size = 10
async_task_queue_buff_len = 50

[eventbus.file]
async_task = off
```

---

## 中间件配置

支持 `Metrics`、`Tracing`、`Recovery`、`Logging` 四种中间件，通过全局默认 + 实例覆盖机制控制。

### 全局默认 + 实例覆盖

```ini
[eventbus]
# 全局默认 — 所有实例生效
metrics_enabled = on
tracing_enabled = on
recovery_enabled = on

[eventbus.sys]
async_task = on
# 所有中间件继承全局默认 = on

[eventbus.file]
async_task = off
logging = off  # 关闭 logging，其余继承全局
```

### 仅单个实例开启

```ini
[eventbus.sys]
async_task = on
metrics = on
tracing = on
recovery = on
```

### 全关（默认）

不配置任何中间件键，所有中间件默认关闭。

### 中间件执行顺序（洋葱模型）

```
Recovery（最外层 · panic 恢复）
  └── Metrics（指标记录）
      └── Tracing（Span 创建）
          └── Logging（日志输出 · 最内层）
              └── Dispatcher
```

---

## 配置项说明

| 配置项 | 类型 | 默认值 | 说明 |
|--------|------|--------|------|
| `id` | string | — | 实例标识（单实例模式必填） |
| `async_task` | bool | `false` | 启用 gotask 工作池作为 Dispatcher |
| `async_task_work_size` | int | `3` | 工作池 worker 数量 |
| `async_task_queue_buff_len` | int | `20` | 任务队列缓冲长度 |
| `metrics_enabled` | bool | `false` | 全局默认：启用 MetricsMiddleware |
| `tracing_enabled` | bool | `false` | 全局默认：启用 TracingMiddleware |
| `recovery_enabled` | bool | `false` | 全局默认：启用 RecoveryMiddleware |
| `logging_enabled` | bool | `false` | 全局默认：启用 LoggingMiddleware |
| `metrics` | bool | 继承全局 | 实例级覆写 |
| `tracing` | bool | 继承全局 | 实例级覆写 |
| `recovery` | bool | 继承全局 | 实例级覆写 |
| `logging` | bool | 继承全局 | 实例级覆写 |

> `async_task` 开启后，EventBus 使用 `eventbus-dispatcher-gotask` 的 `TaskQueueDispatcher` 作为默认分发器，事件通过 gotask 工作池执行。未开启时默认使用 `SyncDispatcher`。

---

## 注入方式

```go
// 方式一：Fx 命名注入
fx.Invoke(fx.Annotate(
    func(bus *eventbus.EventBus) {
        bus.On("order.created", func(e eventbus.Event) {
            // ...
        })
    },
    fx.ParamTags(`name:"eventbus.sys"`),
))

// 方式二：全局 API
bus := eventbus.Get("sys")
bus.On("order.created", func(e eventbus.Event) {
    // ...
})
```

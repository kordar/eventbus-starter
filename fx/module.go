package eventbus_starter

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/kordar/eventbus"
	dispatchergotask "github.com/kordar/eventbus-dispatcher-gotask"
	gocfgmodulefx "github.com/kordar/gocfg-load-module/fx/v2"
	"github.com/kordar/gotask"
	"github.com/spf13/cast"
	"go.uber.org/fx"
)

// InstanceConfig 单个 EventBus 实例配置
type InstanceConfig struct {
	ID               string
	AsyncTask        bool
	AsyncTaskWorkers int
	AsyncTaskQueue   int
	Metrics          bool
	Tracing          bool
	Recovery         bool
	Logging          bool
}

// ModuleConfig 事件总线模块配置
type ModuleConfig struct {
	Instances       []InstanceConfig
	MetricsEnabled  bool // 全局默认：指标
	TracingEnabled  bool // 全局默认：链路追踪
	RecoveryEnabled bool // 全局默认：panic 恢复
	LoggingEnabled  bool // 全局默认：调试日志
}

type cfgModule struct {
	name  string
	index int
}

var _ gocfgmodulefx.GoCfgModule = cfgModule{}
var _ gocfgmodulefx.GoCfgIndex = cfgModule{}

type Option func(*cfgModule)

// WithIndex 设置优先级
func WithIndex(index int) Option {
	return func(s *cfgModule) {
		s.index = index
	}
}

// StarterModule 返回可注册到 gocfg-load-module/fx 的事件总线模块适配器。
// name 对应配置段名称，例如 "eventbus" → [eventbus] 或 [eventbus.xxx]。
func StarterModule(name string, opts ...Option) gocfgmodulefx.GoCfgModule {
	c := &cfgModule{name: name}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (m cfgModule) Name() string {
	return m.name
}

func (m cfgModule) Index() int {
	return m.index
}

func (m cfgModule) Load(data any) []fx.Option {
	slog.Info("Module Load Complete", "module", "eventbus-starter(fx)")
	return []fx.Option{
		Module(buildModuleConfig(data)),
	}
}

// Module 返回一个 fx.Option，按配置初始化并注册所有 EventBus 实例。
// 每个实例以 `name:"eventbus.<id>"` 的命名标签注册到 fx 容器。
func Module(config ModuleConfig) fx.Option {
	providers := make([]any, 0, len(config.Instances))
	for _, ic := range config.Instances {
		cfg := ic
		providers = append(providers,
			fx.Annotate(
				func() (*eventbus.EventBus, error) { return provideEventBus(cfg) },
				fx.ResultTags(fmt.Sprintf(`name:"eventbus.%s"`, cfg.ID)),
			),
		)
	}

	return fx.Module("eventbus-starter",
		fx.Supply(config),
		fx.Provide(providers...),
	)
}

// ---------- 配置解析 ----------

func buildModuleConfig(data any) ModuleConfig {
	cfg := ModuleConfig{
		Instances: make([]InstanceConfig, 0),
	}

	section := cast.ToStringMap(data)
	if len(section) == 0 {
		return cfg
	}

	// 全局默认
	cfg.MetricsEnabled = parseBool(section["metrics_enabled"])
	cfg.TracingEnabled = parseBool(section["tracing_enabled"])
	cfg.RecoveryEnabled = parseBool(section["recovery_enabled"])
	cfg.LoggingEnabled = parseBool(section["logging_enabled"])

	// 单实例模式：[eventbus] id = "xxx"
	if section["id"] != nil {
		id := cast.ToString(section["id"])
		if id != "" {
			cfg.Instances = append(cfg.Instances, InstanceConfig{
				ID:               id,
				AsyncTask:        parseBool(section["async_task"]),
				AsyncTaskWorkers: cast.ToInt(section["async_task_work_size"]),
				AsyncTaskQueue:   cast.ToInt(section["async_task_queue_buff_len"]),
				Metrics:          pickMiddlewareFlag(cfg.MetricsEnabled, section, "metrics"),
				Tracing:          pickMiddlewareFlag(cfg.TracingEnabled, section, "tracing"),
				Recovery:         pickMiddlewareFlag(cfg.RecoveryEnabled, section, "recovery"),
				Logging:          pickMiddlewareFlag(cfg.LoggingEnabled, section, "logging"),
			})
		}
		return cfg
	}

	// 多实例模式：[eventbus.xxx]
	for id, raw := range section {
		v := cast.ToStringMap(raw)
		cfg.Instances = append(cfg.Instances, InstanceConfig{
			ID:               id,
			AsyncTask:        parseBool(v["async_task"]),
			AsyncTaskWorkers: cast.ToInt(v["async_task_work_size"]),
			AsyncTaskQueue:   cast.ToInt(v["async_task_queue_buff_len"]),
			Metrics:          pickMiddlewareFlag(cfg.MetricsEnabled, v, "metrics"),
			Tracing:          pickMiddlewareFlag(cfg.TracingEnabled, v, "tracing"),
			Recovery:         pickMiddlewareFlag(cfg.RecoveryEnabled, v, "recovery"),
			Logging:          pickMiddlewareFlag(cfg.LoggingEnabled, v, "logging"),
		})
	}

	return cfg
}

// ---------- 实例初始化 ----------

func provideEventBus(cfg InstanceConfig) (*eventbus.EventBus, error) {
	if cfg.ID == "" {
		return nil, fmt.Errorf("eventbus-starter: instance id cannot be empty")
	}

	cfg = normalizeInstanceConfig(cfg)

	var opts []eventbus.Option
	var handle *gotask.TaskHandle
	if cfg.AsyncTask {
		handle = gotask.NewTaskHandleWithName(cfg.ID, cfg.AsyncTaskWorkers, cfg.AsyncTaskQueue)
		handle.StartWorkerPool()
		handle.AddTask(dispatchergotask.EventTask{Name: cfg.ID})
		opts = append(opts, eventbus.WithDispatcher(dispatchergotask.TaskQueueDispatcher{Handle: handle}))
	}

	// 中间件按固定顺序添加 — Logging（最内）→ Tracing → Metrics → Recovery（最外）
	var mw []eventbus.Middleware
	if cfg.Logging {
		mw = append(mw, eventbus.LoggingMiddleware())
	}
	if cfg.Tracing {
		mw = append(mw, eventbus.TracingMiddleware())
	}
	if cfg.Metrics {
		mw = append(mw, eventbus.MetricsMiddleware())
	}
	if cfg.Recovery {
		mw = append(mw, eventbus.RecoveryMiddleware())
	}
	if len(mw) > 0 {
		opts = append(opts, eventbus.WithMiddleware(mw...))
	}

	bus := eventbus.NewEventBus(opts...)

	eventbus.Provide(cfg.ID, bus)

	slog.Info("eventbus instance initialized", "id", cfg.ID,
		"async_task", cfg.AsyncTask,
		"metrics", cfg.Metrics,
		"tracing", cfg.Tracing,
		"recovery", cfg.Recovery,
		"logging", cfg.Logging,
	)
	return bus, nil
}

func parseBool(v any) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "y", "on", "enabled":
			return true
		default:
			return false
		}
	default:
		return cast.ToBool(v)
	}
}

// pickMiddlewareFlag 从实例配置中读取中间件开关，
// 若未显式设置则回退到全局默认值。
func pickMiddlewareFlag(defaultVal bool, section map[string]any, key string) bool {
	if raw, ok := section[key]; ok {
		return parseBool(raw)
	}
	return defaultVal
}

func normalizeInstanceConfig(cfg InstanceConfig) InstanceConfig {
	if cfg.AsyncTaskWorkers <= 0 {
		cfg.AsyncTaskWorkers = 3
	}
	if cfg.AsyncTaskQueue <= 0 {
		cfg.AsyncTaskQueue = 20
	}
	return cfg
}

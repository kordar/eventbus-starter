package eventbus_starter

import (
	"testing"
)

func TestBuildModuleConfigEmpty(t *testing.T) {
	cfg := buildModuleConfig(nil)
	if len(cfg.Instances) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(cfg.Instances))
	}

	cfg = buildModuleConfig(map[string]any{})
	if len(cfg.Instances) != 0 {
		t.Fatalf("expected 0 instances, got %d", len(cfg.Instances))
	}
}

func TestBuildModuleConfigSingle(t *testing.T) {
	cfg := buildModuleConfig(map[string]any{
		"id":                        "default",
		"async_task":                "on",
		"async_task_work_size":      "5",
		"async_task_queue_buff_len": "30",
	})

	if len(cfg.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(cfg.Instances))
	}

	inst := cfg.Instances[0]
	if inst.ID != "default" {
		t.Fatalf("expected id 'default', got %q", inst.ID)
	}
	if !inst.AsyncTask {
		t.Fatal("expected async_task = true")
	}
	if inst.AsyncTaskWorkers != 5 {
		t.Fatalf("expected 5 workers, got %d", inst.AsyncTaskWorkers)
	}
	if inst.AsyncTaskQueue != 30 {
		t.Fatalf("expected 30 queue, got %d", inst.AsyncTaskQueue)
	}
}

func TestBuildModuleConfigSingleAsyncOff(t *testing.T) {
	cfg := buildModuleConfig(map[string]any{
		"id": "sync_bus",
	})

	if len(cfg.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(cfg.Instances))
	}

	if cfg.Instances[0].AsyncTask {
		t.Fatal("expected async_task = false by default")
	}
}

func TestBuildModuleConfigMulti(t *testing.T) {
	cfg := buildModuleConfig(map[string]any{
		"sys": map[string]any{
			"async_task":                "on",
			"async_task_work_size":      "10",
			"async_task_queue_buff_len": "50",
		},
		"file": map[string]any{
			"async_task": "off",
		},
	})

	if len(cfg.Instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(cfg.Instances))
	}

	byID := make(map[string]InstanceConfig, len(cfg.Instances))
	for _, inst := range cfg.Instances {
		byID[inst.ID] = inst
	}

	sys, ok := byID["sys"]
	if !ok {
		t.Fatal("expected instance 'sys' not found")
	}
	if !sys.AsyncTask {
		t.Fatal("expected sys async_task = true")
	}
	if sys.AsyncTaskWorkers != 10 {
		t.Fatalf("expected 10 workers, got %d", sys.AsyncTaskWorkers)
	}

	file, ok := byID["file"]
	if !ok {
		t.Fatal("expected instance 'file' not found")
	}
	if file.AsyncTask {
		t.Fatal("expected file async_task = false")
	}
}

func TestNormalizeInstanceConfigDefaults(t *testing.T) {
	cfg := normalizeInstanceConfig(InstanceConfig{ID: "test"})
	if cfg.AsyncTaskWorkers != 3 {
		t.Fatalf("expected default 3 workers, got %d", cfg.AsyncTaskWorkers)
	}
	if cfg.AsyncTaskQueue != 20 {
		t.Fatalf("expected default 20 queue, got %d", cfg.AsyncTaskQueue)
	}
}

func TestNormalizeInstanceConfigPreservesValues(t *testing.T) {
	cfg := normalizeInstanceConfig(InstanceConfig{
		ID:               "test",
		AsyncTaskWorkers: 7,
		AsyncTaskQueue:   50,
	})
	if cfg.AsyncTaskWorkers != 7 {
		t.Fatalf("expected 7 workers, got %d", cfg.AsyncTaskWorkers)
	}
	if cfg.AsyncTaskQueue != 50 {
		t.Fatalf("expected 50 queue, got %d", cfg.AsyncTaskQueue)
	}
}

package plugins

import (
	"context"
	"fmt"
	"sync"

	"github.com/alpen/alpen-cli/internal/lifecycle"
)

// Plugin 定义插件需要实现的接口
type Plugin interface {
	Name() string
	Handle(ctx context.Context, event lifecycle.Event, payload *lifecycle.Context) error
}

// Registry 维护插件列表并负责派发事件
type Registry struct {
	mu      sync.RWMutex
	plugins []Plugin
}

// NewRegistry 创建插件注册表
func NewRegistry() *Registry {
	return &Registry{
		plugins: make([]Plugin, 0),
	}
}

// Register 向注册表中添加插件
func (r *Registry) Register(plugin Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.plugins {
		if existing.Name() == plugin.Name() {
			return fmt.Errorf("插件 %s 已注册", plugin.Name())
		}
	}
	r.plugins = append(r.plugins, plugin)
	return nil
}

// Emit 将事件广播给所有插件
func (r *Registry) Emit(ctx context.Context, event lifecycle.Event, payload *lifecycle.Context) error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, plugin := range r.plugins {
		if err := plugin.Handle(ctx, event, payload); err != nil {
			return fmt.Errorf("插件 %s 处理事件失败: %w", plugin.Name(), err)
		}
	}
	return nil
}

// Snapshot 返回当前已注册插件列表，便于对外展示
func (r *Registry) Snapshot() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copied := make([]Plugin, len(r.plugins))
	copy(copied, r.plugins)
	return copied
}

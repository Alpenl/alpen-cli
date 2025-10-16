package lifecycle

import (
	"context"
	"sync"
	"time"
)

// Event 定义生命周期事件枚举
type Event string

const (
	EventRegistryLoaded Event = "registry_loaded"
	EventBeforeExecute  Event = "before_execute"
	EventAfterExecute   Event = "after_execute"
	EventError          Event = "error"
)

// Context 提供事件处理所需的上下文信息
type Context struct {
	CommandPath []string
	ScriptName  string
	GroupName   string
	Command     string
	Args        []string
	Env         map[string]string
	StartAt     time.Time
	EndAt       time.Time
	Err         error
}

// Handler 定义事件处理函数签名
type Handler func(ctx context.Context, event Event, payload *Context) error

// Manager 维护事件与处理函数映射
type Manager struct {
	mu       sync.RWMutex
	handlers map[Event][]Handler
}

// NewManager 创建生命周期管理器
func NewManager() *Manager {
	return &Manager{
		handlers: make(map[Event][]Handler),
	}
}

// Register 为指定事件注册处理函数
func (m *Manager) Register(event Event, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[event] = append(m.handlers[event], handler)
}

// Emit 依次触发事件处理函数
func (m *Manager) Emit(ctx context.Context, event Event, payload *Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	handlers := m.handlers[event]
	for _, handler := range handlers {
		if err := handler(ctx, event, payload); err != nil {
			return err
		}
	}
	return nil
}

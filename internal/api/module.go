package api

import (
	"context"
	"reflect"
	"sync"

	"github.com/rs/zerolog"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/database"
	"github.com/nimbleflux/fluxbase/internal/pubsub"
	"github.com/nimbleflux/fluxbase/internal/ratelimit"
)

type Module interface {
	Name() string
	Init(ctx context.Context, registry *ServiceRegistry) error
}

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

type ServiceRegistry struct {
	mu          sync.RWMutex
	services    map[reflect.Type]interface{}
	Config      *config.Config
	DB          *database.Connection
	PubSub      pubsub.PubSub
	RateLimiter ratelimit.Store
}

func NewServiceRegistry(cfg *config.Config, db *database.Connection) *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[reflect.Type]interface{}),
		Config:   cfg,
		DB:       db,
	}
}

func (r *ServiceRegistry) Register(svc interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[reflect.TypeOf(svc)] = svc
}

func (r *ServiceRegistry) Get(target reflect.Type) interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.services[target]
}

func GetService[T any](r *ServiceRegistry) T {
	var zero T
	svc := r.Get(reflect.TypeOf(zero))
	if svc == nil {
		return zero
	}
	return svc.(T)
}

func InitModules(ctx context.Context, registry *ServiceRegistry, mods []Module) error {
	for _, m := range mods {
		if err := m.Init(ctx, registry); err != nil {
			return err
		}
	}
	return nil
}

func ShutdownModules(ctx context.Context, mods []Module) {
	for i := len(mods) - 1; i >= 0; i-- {
		if s, ok := mods[i].(Shutdowner); ok {
			if err := s.Shutdown(ctx); err != nil {
				zerolog.Ctx(ctx).Error().Err(err).Str("module", mods[i].Name()).Msg("Module shutdown failed")
			}
		}
	}
}

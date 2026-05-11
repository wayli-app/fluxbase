package api

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/storage"
)

type StorageModule struct {
	Handlers *StorageHandlers
	Manager  *storage.Manager
	Service  *storage.Service
}

func (m *StorageModule) Name() string { return "storage" }

func (m *StorageModule) Init(ctx context.Context, registry *ServiceRegistry) error {
	cfg := registry.Config
	db := registry.DB

	storageManager, err := storage.NewManager(&cfg.Storage, cfg.GetPublicBaseURL(), cfg.Auth.JWTSecret)
	if err != nil {
		return err
	}
	m.Manager = storageManager
	m.Service = storageManager.GetBaseService()

	if err := storageManager.EnsureDefaultBuckets(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to ensure default buckets")
	}

	if err := EnsureDefaultBucketRecords(ctx, db.Pool(), m.Service.DefaultBuckets()); err != nil {
		log.Warn().Err(err).Msg("Failed to ensure default bucket DB records")
	}

	m.Handlers = &StorageHandlers{
		Handler: NewStorageHandler(storageManager, db, cfg, &cfg.Storage.Transforms),
	}

	registry.Register(m.Service)
	return nil
}

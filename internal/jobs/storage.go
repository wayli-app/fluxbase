package jobs

import (
	"github.com/nimbleflux/fluxbase/internal/database"
)

type Storage struct {
	database.TenantAware
}

func NewStorage(conn *database.Connection) *Storage {
	return &Storage{TenantAware: database.TenantAware{DB: conn}}
}

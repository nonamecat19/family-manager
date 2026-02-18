package graph

import (
	"github.com/nnc/notes-manager-backend/internal/auth"
	"github.com/nnc/notes-manager-backend/internal/database/sqlc"
	"github.com/nnc/notes-manager-backend/internal/storage"
)

type Resolver struct {
	Queries *sqlc.Queries
	Auth    *auth.Service
	JWT     *auth.JWTManager
	Storage *storage.MinIOStorage
}

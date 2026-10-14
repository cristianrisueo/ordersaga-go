// Aplica con goose las migraciones SQL que recibe, sin saber de qué servicio son

package database

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Migrate aplica con goose las migraciones pendientes de la carpeta dir de fsys.
func Migrate(ctx context.Context, pool *pgxpool.Pool, fsys fs.FS, dir string) error {
	// Abre sobre el pool el *sql.DB que necesita goose (cerrarlo no cierra el pool)
	db := stdlib.OpenDBFromPool(pool)

	defer func() { _ = db.Close() }()

	// Deja las migraciones de dir en la raíz del sistema de ficheros que ve goose
	migrationsFS, err := fs.Sub(fsys, dir)
	if err != nil {
		return fmt.Errorf("open migrations dir: %w", err)
	}

	// Crea el advisory lock de PostgreSQL para que dos procesos no puedan migrar a la vez
	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("create migrations lock: %w", err)
	}

	// Crea el proveedor de goose para PostgreSQL con el lock
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("create migrations provider: %w", err)
	}

	// Aplica las migraciones pendientes
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	// Registra cada migración aplicada, o que no había ninguna pendiente
	if len(results) == 0 {
		slog.Info("no pending migrations")
	}
	for _, result := range results {
		slog.Info("migration applied", "version", result.Source.Version, "file", result.Source.Path)
	}

	// Devuelve nil con la base de datos al día
	return nil
}

// Abre la conexión con la base de datos.

package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect crea el pool de conexiones y comprueba que la base de datos responde.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	// Crea el pool (las conexiones se abren bajo demanda)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	// Hace un ping y comprueba que responda
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Devuelve el pool listo para usar
	return pool, nil
}

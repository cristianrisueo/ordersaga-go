// Migraciones SQL de goose de cada servicio, embebidas en el binario

package migrations

import "embed"

// Orders contiene las migraciones de la base de datos de orders.
//
//go:embed orders/*.sql
var Orders embed.FS

// Stock contiene las migraciones de la base de datos de stock.
//
//go:embed stock/*.sql
var Stock embed.FS

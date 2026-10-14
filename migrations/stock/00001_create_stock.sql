-- +goose Up

-- Productos del catálogo con las unidades que quedan disponibles
CREATE TABLE products (
    id        TEXT PRIMARY KEY,
    available INT NOT NULL
);

-- Reservas de unidades, una por pedido: reserved, failed o released
CREATE TABLE reservations (
    order_id   UUID PRIMARY KEY,
    product_id TEXT NOT NULL,
    quantity   INT NOT NULL,
    status     TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Eventos pendientes de publicar en Kafka, guardados en la misma transacción que el cambio que los origina
CREATE TABLE outbox (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    event_type TEXT NOT NULL,
    payload    JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down

-- Eventos pendientes de publicar
DROP TABLE outbox;

-- Reservas
DROP TABLE reservations;

-- Productos
DROP TABLE products;

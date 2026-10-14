-- +goose Up

-- Pedidos: cada uno nace en pending y la saga lo pasa a confirmed o cancelled
CREATE TABLE orders (
    id         UUID PRIMARY KEY,
    product_id TEXT NOT NULL,
    quantity   INT NOT NULL,
    amount     INT NOT NULL,
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

-- Pedidos
DROP TABLE orders;

-- +goose Up

-- Pagos, uno por pedido: completed si el importe no supera el máximo y failed si lo supera
CREATE TABLE payments (
    order_id   UUID PRIMARY KEY,
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

-- Pagos
DROP TABLE payments;

-- +goose Up

-- Catálogo inicial: unidades disponibles de cada producto
INSERT INTO products (id, available) VALUES
    ('teclado', 5),
    ('monitor', 1);

-- +goose Down

-- Quita el catálogo inicial
DELETE FROM products WHERE id IN ('teclado', 'monitor');

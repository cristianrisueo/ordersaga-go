# OrderSaga Go

Saga por coreografía entre tres microservicios en Go (orders, stock y payments) que se comunican solo por
eventos en Kafka. Reescritura de una versión Python con el mismo comportamiento. Proyecto de aprendizaje: lo más
simple que funcione bien, idiomático y con buenas prácticas de Go.

## Reglas de trabajo

- Nunca hagas commit, push ni ninguna operación de git que modifique el repositorio.
- Haz solo lo que pide el bloque en curso. Si la especificación tiene un hueco o algo no encaja, para y pregunta.
- Antes de dar un bloque por terminado: `make check` limpio y la prueba del bloque pasando.
- Al terminar, informa de los ficheros creados o modificados y de la salida de las comprobaciones.

## Estructura

- `cmd/<servicio>/main.go`: un binario por servicio con tres modos (`api`, `producer`, `consumer`), elegidos
  por argumento. Es el único sitio donde se conectan las piezas. La configuración se lee aquí con `os.Getenv`.
- `internal/database`: conexión a PostgreSQL. No sabe nada de eventos.
- `internal/saga`: cómo viajan los eventos: el tipo `Event`, el productor de la outbox y el consumidor.
- `internal/<dominio>`: un paquete por servicio con siempre los mismos ficheros:
  - `schema.sql`: sus tablas, incluida su `outbox`; la API lo aplica al arrancar.
  - `types.go`: tipos del dominio, estados y errores.
  - `repository.go`: solo SQL, incluido `insertOutbox`.
  - `service.go`: reglas de negocio y límite de cada transacción.
  - `http_routes.go`: las rutas y el middleware de cada una.
  - `http_handler.go`: la puerta HTTP: valida, llama al servicio y responde JSON.
  - `events_handler.go`: la puerta de Kafka: qué evento va a qué función del servicio.
- Los dominios importan `saga` y `database`; `saga` importa `database`; `database` no importa nada interno.

## Dependencias

Solo `pgx/v5`, `franz-go`, `google/uuid` y, en los tests, `testcontainers-go`. Ninguna otra sin preguntar.

## Convenciones de Go

- `context.Context` es el primer parámetro de toda función de servicio y repositorio, y se pasa a cada llamada
  a la base de datos y a Kafka.
- Errores envueltos con contexto (`fmt.Errorf("insert order: %w", err)`), comparados con `errors.Is`; los
  errores de dominio son variables del paquete (`ErrNotFound`).
- Sin repetir el paquete en los nombres: `orders.Service`, `orders.Handler`, nunca `orders.OrderService`.
- Tipos concretos. Ninguna interfaz ni abstracción que no se use hoy.
- Logs con `log/slog`; cada `main.go` configura un `JSONHandler` como logger por defecto.
- Ficheros en minúsculas con guion bajo si hace falta, nunca camelCase.

## Transacciones y SQL

- El servicio abre la transacción con `pgx.BeginFunc` y decide qué va dentro. Los métodos de escritura del
  repositorio reciben `pgx.Tx`; las lecturas usan el pool del repositorio.
- El cambio de negocio y su evento en la outbox se guardan siempre en la misma transacción.
- SQL a mano con placeholders (`$1`), nunca concatenando. Columnas explícitas, nunca `SELECT *`.
- Idempotencia por clave de negocio: `INSERT ... ON CONFLICT DO NOTHING RETURNING` y `UPDATE` condicionales
  sobre el estado. Nunca comprobar si algo existe antes de insertarlo.
- Contrato de la tabla `outbox` (igual en los tres servicios, la lee `saga.RunProducer`):
  `id UUID PRIMARY KEY DEFAULT uuidv7()`, `event_type TEXT NOT NULL`, `payload JSONB NOT NULL`,
  `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`.

## Comentarios

- En español.
- El código va en bloques separados por una línea en blanco, con un comentario de una línea encima de cada
  bloque que resume lo que hace. Sin comentarios dentro de las ramas de un `if`/`else`.
- Comentarios de documentación cortos, de una línea, en todo identificador exportado.

## Tests

- Solo `testing` y `httptest`. Sin mocks, fakes ni librerías de aserciones.
- PostgreSQL real con testcontainers-go, aplicando el `schema.sql` del dominio.
- Nombres `Test<Unidad>_<Método>_<Caso>`; tablas de casos con `t.Run` cuando haya varios.

## Compatibilidad con los scripts de prueba

Los scripts de `scripts/` prueban el sistema como caja negra. No cambies sin preguntar: los puertos (8001
orders, 8002 stock, 8003 payments), las rutas, los nombres de los servicios de compose (`kafka`, `kafka-init`,
`db-orders`, `db-stock`, `db-payments`...), el usuario y la base de datos de cada PostgreSQL (igual al nombre del
servicio), la tabla `outbox`, los topics (`orders`, `stock`, `payments`) ni los grupos de consumidores (igual al
nombre del servicio).

## Comandos

- `make up` / `make down`: levanta todo y espera a que esté sano / lo para y borra los datos.
- `make logs`: sigue los logs.
- `make check`: formatea y pasa golangci-lint.
- `make test`: tests con el detector de carreras.

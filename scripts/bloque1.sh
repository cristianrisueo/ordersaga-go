#!/usr/bin/env bash
# Comprobaciones del bloque 1: levanta todo desde cero, prueba la API de orders y el outbox, y lo apaga.
set -uo pipefail

# Se ejecuta siempre desde la raíz del proyecto
cd "$(dirname "$0")/.."

# URL de la API, contadores de resultados y el pedido que se crea en las pruebas
API=http://localhost:8001
PASSED=0
FAILED=0
ORDER_ID=""

# Al salir, haya ido bien o mal, borra contenedores y datos
trap 'docker compose down -v > /dev/null 2>&1' EXIT

# Imprime OK o FALLO según si la condición (el resto de argumentos) se cumple, y lleva la cuenta
check() {
  local name=$1
  shift
  if "$@"; then
    echo "OK    $name"
    PASSED=$((PASSED + 1))
  else
    echo "FALLO $name"
    FAILED=$((FAILED + 1))
  fi
}

# Arranca desde cero; si no levanta, no tiene sentido seguir
docker compose down -v > /dev/null 2>&1
if ! make up; then
  echo "FALLO make up: el sistema no ha arrancado"
  exit 1
fi
echo

# 1. /health responde 200
status=$(curl -s -o /dev/null -w '%{http_code}' "$API/health")
check "GET /health responde 200 (recibido $status)" test "$status" = 200

# 2. Crear un pedido válido responde 201 en pending
response=$(curl -s -w '\n%{http_code}' -X POST "$API/orders" \
  -H 'Content-Type: application/json' \
  -d '{"product_id": "teclado", "quantity": 2, "amount": 50}')
status=$(tail -n 1 <<< "$response")
body=$(sed '$d' <<< "$response")
ORDER_ID=$(jq -r '.id // empty' <<< "$body")
order_status=$(jq -r '.status // empty' <<< "$body")
check "POST /orders responde 201 con status pending (recibido $status, $order_status)" \
  test "$status" = 201 -a "$order_status" = pending

# 3. Leer ese pedido devuelve pending
order_status=$(curl -s "$API/orders/$ORDER_ID" | jq -r '.status // empty')
check "GET /orders/{id} devuelve pending (recibido $order_status)" test "$order_status" = pending

# 4. Un pedido con quantity 0 se rechaza con 422
status=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$API/orders" \
  -H 'Content-Type: application/json' \
  -d '{"product_id": "teclado", "quantity": 0, "amount": 50}')
check "POST /orders con quantity 0 responde 422 (recibido $status)" test "$status" = 422

# 5. Un pedido que no existe devuelve 404
status=$(curl -s -o /dev/null -w '%{http_code}' "$API/orders/$(uuidgen | tr '[:upper:]' '[:lower:]')")
check "GET /orders/{uuid inventado} responde 404 (recibido $status)" test "$status" = 404

# 6. El evento order.created del pedido está en el topic orders, con el order_id como clave
messages=$(docker compose exec -T kafka /opt/kafka/bin/kafka-console-consumer.sh \
  --bootstrap-server kafka:9092 --topic orders --from-beginning --timeout-ms 10000 \
  --property print.key=true --property key.separator='|' 2> /dev/null)
event_type=$(grep "^$ORDER_ID|" <<< "$messages" | head -n 1 | cut -d '|' -f 2- | jq -r '.event_type // empty')
check "order.created en el topic orders con clave order_id (recibido '${event_type}')" \
  test -n "$ORDER_ID" -a "$event_type" = order.created

# 7. La outbox queda vacía tras unos segundos
for _ in $(seq 10); do
  pending=$(docker compose exec -T db-orders psql -U orders -d orders -tAc "SELECT count(*) FROM outbox")
  [ "$pending" = 0 ] && break
  sleep 1
done
check "La outbox queda vacía (filas: $pending)" test "$pending" = 0

# Resumen: sale con error si alguna comprobación ha fallado
echo
echo "Resumen: $PASSED OK, $FAILED FALLO"
[ "$FAILED" -eq 0 ]

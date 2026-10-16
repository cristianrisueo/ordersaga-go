#!/usr/bin/env bash
# Comprobaciones del bloque 2: levanta todo desde cero, prueba el camino feliz de la saga y lo apaga.
set -uo pipefail

# Se ejecuta siempre desde la raíz del proyecto
cd "$(dirname "$0")/.."

# URLs de las APIs, contadores de resultados y los pedidos que se crean en las pruebas
ORDERS=http://localhost:8001
STOCK=http://localhost:8002
PAYMENTS=http://localhost:8003
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

# Devuelve las unidades disponibles de un producto según la API de stock
available() {
  curl -s "$STOCK/stock/$1" | jq -r '.available // empty'
}

# Crea un pedido y devuelve su id
create_order() {
  curl -s -X POST "$ORDERS/orders" \
    -H 'Content-Type: application/json' \
    -d "{\"product_id\": \"$1\", \"quantity\": $2, \"amount\": 50}" | jq -r '.id // empty'
}

# Devuelve el estado de un pedido según la API de orders
order_status() {
  curl -s "$ORDERS/orders/$1" | jq -r '.status // empty'
}

# Lee todos los mensajes de un topic como «clave|valor»
topic_messages() {
  docker compose exec -T kafka /opt/kafka/bin/kafka-console-consumer.sh \
    --bootstrap-server kafka:9092 --topic "$1" --from-beginning --timeout-ms 10000 \
    --property print.key=true --property key.separator='|' 2> /dev/null
}

# Devuelve el event_type del primer mensaje de un topic (primer argumento) con un order_id como clave (segundo)
event_type() {
  topic_messages "$1" | grep "^$2|" | head -n 1 | cut -d '|' -f 2- | jq -r '.event_type // empty'
}

# Arranca desde cero; si no levanta, no tiene sentido seguir
docker compose down -v > /dev/null 2>&1
if ! make up; then
  echo "FALLO make up: el sistema no ha arrancado"
  exit 1
fi
echo

# 1. /health de stock y payments responden 200
stock_health=$(curl -s -o /dev/null -w '%{http_code}' "$STOCK/health")
payments_health=$(curl -s -o /dev/null -w '%{http_code}' "$PAYMENTS/health")
check "GET /health de stock y payments responden 200 (recibido $stock_health y $payments_health)" \
  test "$stock_health" = 200 -a "$payments_health" = 200

# 2. El catálogo inicial está cargado
teclado=$(available teclado)
monitor=$(available monitor)
check "GET /stock/teclado devuelve 5 y /stock/monitor devuelve 1 (recibido $teclado y $monitor)" \
  test "$teclado" = 5 -a "$monitor" = 1

# 3. Un producto y un pago que no existen devuelven 404
stock_status=$(curl -s -o /dev/null -w '%{http_code}' "$STOCK/stock/raton")
payment_status=$(curl -s -o /dev/null -w '%{http_code}' "$PAYMENTS/payments/$(uuidgen | tr '[:upper:]' '[:lower:]')")
check "GET /stock/raton y /payments/{uuid inventado} responden 404 (recibido $stock_status y $payment_status)" \
  test "$stock_status" = 404 -a "$payment_status" = 404

# 4. Un pedido de 2 teclados recorre la saga hasta confirmed
ORDER_ID=$(create_order teclado 2)
for _ in $(seq 20); do
  status=$(order_status "$ORDER_ID")
  [ "$status" = confirmed ] && break
  sleep 1
done
check "Tras pedir 2 teclados, GET /orders/{id} acaba en confirmed (recibido $status)" test "$status" = confirmed

# 5. Stock ha descontado las 2 unidades
teclado=$(available teclado)
check "GET /stock/teclado devuelve 3 (recibido $teclado)" test "$teclado" = 3

# 6. Payments ha cobrado el importe del pedido
payment=$(curl -s "$PAYMENTS/payments/$ORDER_ID")
payment_status=$(jq -r '.status // empty' <<< "$payment")
amount=$(jq -r '.amount // empty' <<< "$payment")
check "GET /payments/{id} devuelve completed con amount 50 (recibido $payment_status, $amount)" \
  test "$payment_status" = completed -a "$amount" = 50

# 7. Los eventos stock.reserved y payment.completed del pedido están publicados, con el order_id como clave
stock_event=$(event_type stock "$ORDER_ID")
payments_event=$(event_type payments "$ORDER_ID")
check "stock.reserved y payment.completed publicados con clave order_id (recibido '${stock_event}' y '${payments_event}')" \
  test -n "$ORDER_ID" -a "$stock_event" = stock.reserved -a "$payments_event" = payment.completed

# 8. Las outbox de orders, stock y payments quedan vacías
for _ in $(seq 20); do
  orders_pending=$(docker compose exec -T db-orders psql -U orders -d orders -tAc "SELECT count(*) FROM outbox")
  stock_pending=$(docker compose exec -T db-stock psql -U stock -d stock -tAc "SELECT count(*) FROM outbox")
  payments_pending=$(docker compose exec -T db-payments psql -U payments -d payments -tAc "SELECT count(*) FROM outbox")
  [ "$orders_pending" = 0 ] && [ "$stock_pending" = 0 ] && [ "$payments_pending" = 0 ] && break
  sleep 1
done
check "Las outbox de orders, stock y payments quedan vacías (filas: $orders_pending, $stock_pending y $payments_pending)" \
  test "$orders_pending" = 0 -a "$stock_pending" = 0 -a "$payments_pending" = 0

# Resumen: sale con error si alguna comprobación ha fallado
echo
echo "Resumen: $PASSED OK, $FAILED FALLO"
[ "$FAILED" -eq 0 ]

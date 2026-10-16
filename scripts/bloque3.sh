#!/usr/bin/env bash
# Comprobaciones del bloque 3: levanta todo desde cero, prueba los caminos tristes (sin stock y pago rechazado) y lo apaga.
set -uo pipefail

# Se ejecuta siempre desde la raíz del proyecto
cd "$(dirname "$0")/.."

# URLs de las APIs, contadores de resultados y los pedidos que se crean en las pruebas
ORDERS=http://localhost:8001
STOCK=http://localhost:8002
PAYMENTS=http://localhost:8003
PASSED=0
FAILED=0
NO_STOCK_ID=""
REJECTED_ID=""
HAPPY_ID=""

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

# Crea un pedido (producto, cantidad e importe) y devuelve su id
create_order() {
  curl -s -X POST "$ORDERS/orders" \
    -H 'Content-Type: application/json' \
    -d "{\"product_id\": \"$1\", \"quantity\": $2, \"amount\": $3}" | jq -r '.id // empty'
}

# Devuelve el estado de un pedido según la API de orders
order_status() {
  curl -s "$ORDERS/orders/$1" | jq -r '.status // empty'
}

# Espera hasta 20 segundos a que un pedido (primer argumento) llegue a un estado (segundo) y devuelve el último leído
wait_order_status() {
  local status=""
  for _ in $(seq 20); do
    status=$(order_status "$1")
    [ "$status" = "$2" ] && break
    sleep 1
  done
  echo "$status"
}

# Lee todos los mensajes de un topic como «clave|valor»
topic_messages() {
  docker compose exec -T kafka /opt/kafka/bin/kafka-console-consumer.sh \
    --bootstrap-server kafka:9092 --topic "$1" --from-beginning --timeout-ms 10000 \
    --property print.key=true --property key.separator='|' 2> /dev/null
}

# Devuelve, separados por comas y en orden, los event_type de un topic (primer argumento) con un order_id como clave (segundo)
event_types() {
  topic_messages "$1" | grep "^$2|" | cut -d '|' -f 2- | jq -r '.event_type // empty' | paste -sd , -
}

# Arranca desde cero; si no levanta, no tiene sentido seguir
docker compose down -v > /dev/null 2>&1
if ! make up; then
  echo "FALLO make up: el sistema no ha arrancado"
  exit 1
fi
echo

# 1. /health de orders, stock y payments responden 200
orders_health=$(curl -s -o /dev/null -w '%{http_code}' "$ORDERS/health")
stock_health=$(curl -s -o /dev/null -w '%{http_code}' "$STOCK/health")
payments_health=$(curl -s -o /dev/null -w '%{http_code}' "$PAYMENTS/health")
check "GET /health de orders, stock y payments responden 200 (recibido $orders_health, $stock_health y $payments_health)" \
  test "$orders_health" = 200 -a "$stock_health" = 200 -a "$payments_health" = 200

# 2. Sin stock: un pedido de 5 monitores acaba cancelado
NO_STOCK_ID=$(create_order monitor 5 50)
status=$(wait_order_status "$NO_STOCK_ID" cancelled)
check "Sin stock: tras pedir 5 monitores, el pedido acaba en cancelled (recibido $status)" test "$status" = cancelled

# 3. Sin stock: no se descuenta, no se cobra y se publica stock.failed
monitor=$(available monitor)
payment_status=$(curl -s -o /dev/null -w '%{http_code}' "$PAYMENTS/payments/$NO_STOCK_ID")
stock_events=$(event_types stock "$NO_STOCK_ID")
check "Sin stock: monitor sigue en 1, /payments/{id} responde 404 y stock.failed publicado (recibido $monitor, $payment_status, '${stock_events}')" \
  test -n "$NO_STOCK_ID" -a "$monitor" = 1 -a "$payment_status" = 404 -a "$stock_events" = stock.failed

# 4. Pago rechazado: un pedido de 2 teclados por 5000 euros acaba cancelado
REJECTED_ID=$(create_order teclado 2 5000)
status=$(wait_order_status "$REJECTED_ID" cancelled)
check "Pago rechazado: tras pedir 2 teclados por 5000, el pedido acaba en cancelled (recibido $status)" test "$status" = cancelled

# 5. Pago rechazado: el pago queda en failed con el importe del pedido
payment=$(curl -s "$PAYMENTS/payments/$REJECTED_ID")
payment_status=$(jq -r '.status // empty' <<< "$payment")
amount=$(jq -r '.amount // empty' <<< "$payment")
check "Pago rechazado: GET /payments/{id} devuelve failed con amount 5000 (recibido $payment_status, $amount)" \
  test "$payment_status" = failed -a "$amount" = 5000

# 6. Pago rechazado: las unidades reservadas se han liberado
teclado=$(available teclado)
check "Pago rechazado: GET /stock/teclado vuelve a 5 (recibido $teclado)" test "$teclado" = 5

# 7. Pago rechazado: stock.reserved y stock.released en stock, y payment.failed en payments, con el order_id como clave
stock_events=$(event_types stock "$REJECTED_ID")
payments_events=$(event_types payments "$REJECTED_ID")
check "Pago rechazado: stock.reserved, stock.released y payment.failed publicados con clave order_id (recibido '${stock_events}' y '${payments_events}')" \
  test -n "$REJECTED_ID" -a "$stock_events" = stock.reserved,stock.released -a "$payments_events" = payment.failed

# 8. El camino feliz sigue funcionando: un pedido de 1 teclado acaba confirmado y descuenta su unidad
HAPPY_ID=$(create_order teclado 1 50)
status=$(wait_order_status "$HAPPY_ID" confirmed)
teclado=$(available teclado)
check "Camino feliz: tras pedir 1 teclado, el pedido acaba en confirmed y teclado en 4 (recibido $status, $teclado)" \
  test "$status" = confirmed -a "$teclado" = 4

# 9. Las outbox de orders, stock y payments quedan vacías
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

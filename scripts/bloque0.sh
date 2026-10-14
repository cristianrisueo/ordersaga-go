#!/usr/bin/env bash
# Prueba del bloque 0: Kafka arranca con los topics de la saga y sin autocreación de topics.

set -uo pipefail

# Se ejecuta desde la raíz del proyecto y deja el entorno limpio al salir
cd "$(dirname "$0")/.."
trap 'docker compose down -v > /dev/null 2>&1' EXIT

# Imprime OK o FALLO según el resultado del comando y lleva la cuenta
ok=0
fail=0
check() {
  local name=$1
  shift
  if "$@" > /dev/null 2>&1; then
    echo "OK    $name"
    ok=$((ok + 1))
  else
    echo "FALLO $name"
    fail=$((fail + 1))
  fi
}

# Comprueba que un topic existe y tiene 3 particiones (grep sin -q: si corta la tubería, exec devuelve 255)
topic_has_3_partitions() {
  docker compose exec -T kafka /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 \
    --describe --topic "$1" | grep "PartitionCount: 3"
}

# Comprueba que el broker tiene desactivada la autocreación de topics (grep sin -q por lo mismo)
auto_create_disabled() {
  docker compose exec -T kafka /opt/kafka/bin/kafka-configs.sh --bootstrap-server kafka:9092 \
    --describe --entity-type brokers --entity-name 1 --all | grep "auto.create.topics.enable=false"
}

# Arranca desde cero y espera a que kafka-init cree los topics (--wait no espera a que termine)
docker compose down -v > /dev/null 2>&1
if ! make up > /dev/null 2>&1 || ! docker compose wait kafka-init > /dev/null 2>&1; then
  echo "FALLO make up"
  exit 1
fi

# Comprobaciones
for topic in orders stock payments; do
  check "topic $topic con 3 particiones" topic_has_3_partitions "$topic"
done
check "autocreación de topics desactivada" auto_create_disabled

# Resumen
echo
echo "Resumen: $ok OK, $fail FALLO"
[ "$fail" -eq 0 ]

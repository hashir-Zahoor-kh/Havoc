#!/usr/bin/env sh
# Creates the two Kafka topics Havoc uses. Idempotent — re-running is
# safe; existing topics are left alone. Auto-create is on in dev so
# this script is a belt for the suspenders rather than strictly needed,
# but it makes the partition count explicit for the demo.
#
# Run from the host: `make bootstrap` or directly:
#   docker compose -f deploy/docker-compose.yml exec kafka /bin/sh -c \
#     "/havoc/create-topics.sh"
set -eu

BROKER="${BROKER:-localhost:9092}"
PARTITIONS="${PARTITIONS:-3}"
RF="${RF:-1}"

create() {
  topic="$1"
  if kafka-topics --bootstrap-server "$BROKER" --list | grep -qx "$topic"; then
    echo "topic $topic already exists"
  else
    kafka-topics --bootstrap-server "$BROKER" \
      --create --topic "$topic" \
      --partitions "$PARTITIONS" --replication-factor "$RF"
    echo "created topic $topic"
  fi
}

create havoc.commands
create havoc.results

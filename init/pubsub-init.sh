#!/bin/bash

#set -euo pipefail

project="${PROJECT:-wire-emulator}"
pubsub_host="${PUBSUB_EMULATOR_URL:-http://pubsub:9200}"
bindings_file="${PUBSUB_BINDINGS_FILE:-/init/pubsub-bindings}"

create_topic() {
  local topic="$1"
  curl --fail --silent --show-error --location --request PUT \
    "${pubsub_host}/v1/projects/${project}/topics/${topic}"
}

create_subscription() {
  local topic="$1"
  local subscription="$2"

  echo "host: ${pubsub_host}; project: ${project}; topic: ${topic}; sub: ${subscription}"

  curl --fail --silent --show-error --location --request PUT \
    "${pubsub_host}/v1/projects/${project}/subscriptions/${subscription}" \
    --header 'Content-Type: application/json' \
    --data '{
      "topic": "projects/'"${project}"'/topics/'"${topic}"'",
      "retryPolicy": {
            "minimumBackoff": "1s",
            "maximumBackoff": "10s"
        }
    }'
}

if [[ ! -f "$bindings_file" ]]; then
  echo "Bindings file not found: ${bindings_file}" >&2
  exit 1
fi

echo "project: ${project}"
echo "pubsub_host: ${pubsub_host}"
echo "bindings_file: ${bindings_file}"

echo "Setting up Pub/Sub"

declare -A created_topics=()

while IFS= read -r raw_line || [[ -n "$raw_line" ]]; do
  line="${raw_line%%#*}"
  line="${line#${line%%[![:space:]]*}}"
  line="${line%${line##*[![:space:]]}}"

  if [[ -z "$line" ]]; then
    continue
  fi

  if [[ "$line" != *:* ]]; then
    echo "Invalid binding '${line}'. Expected format topic:subscription" >&2
    exit 1
  fi

  topic="${line%%:*}"
  subscription="${line#*:}"

  if [[ -z "$topic" || -z "$subscription" ]]; then
    echo "Invalid binding '${line}'. Topic and subscription are required." >&2
    exit 1
  fi

  if [[ -z "${created_topics[$topic]:-}" ]]; then
    create_topic "$topic"
    created_topics[$topic]=1
  fi

  create_subscription "$topic" "$subscription"
done < "$bindings_file"

#!/usr/bin/env python3
"""
Google Cloud Pub/Sub message listener utility.

Usage:
    python pubsub_listener.py --project PROJECT_ID --subscription SUBSCRIPTION_ID
                              [--host HOST] [--port PORT]

The --host and --port flags configure the Pub/Sub emulator endpoint.
Omit them (or leave blank) to connect to the real Google Cloud Pub/Sub service.
"""

import argparse
import json
import logging
import os
import signal
import sys
from datetime import datetime, timezone

from google.cloud import pubsub_v1
from google.api_core.exceptions import GoogleAPICallError, NotFound


# ---------------------------------------------------------------------------
# Logging setup
# ---------------------------------------------------------------------------

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%dT%H:%M:%S%z",
)
logger = logging.getLogger(__name__)


# ---------------------------------------------------------------------------
# Message callback
# ---------------------------------------------------------------------------

def _decode_payload(data: bytes) -> str:
    """Try to return a pretty-printed JSON string; fall back to raw text."""
    try:
        parsed = json.loads(data.decode("utf-8"))
        return json.dumps(parsed, indent=2, ensure_ascii=False)
    except (json.JSONDecodeError, UnicodeDecodeError):
        try:
            return data.decode("utf-8")
        except UnicodeDecodeError:
            return repr(data)


def make_callback(ack: bool = True):
    """Return a Pub/Sub subscriber callback that logs message details."""

    def callback(message: pubsub_v1.types.PubsubMessage) -> None:
        received_at = datetime.now(tz=timezone.utc).isoformat()
        payload = _decode_payload(message.data)

        # Build a readable attribute block (may be empty)
        attrs = dict(message.attributes) if message.attributes else {}
        attrs_str = json.dumps(attrs, indent=2) if attrs else "(none)"

        # Pub/Sub publish time (proto Timestamp → datetime)
        try:
            publish_dt = message.publish_time.isoformat()
        except AttributeError:
            publish_dt = "unavailable"

        logger.info(
            "\n"
            "╔══════════════════════════════════════════════════════════════╗\n"
            "  📨  Pub/Sub Message Received\n"
            "╚══════════════════════════════════════════════════════════════╝\n"
            "  Message ID   : %s\n"
            "  Received at  : %s\n"
            "  Published at : %s\n"
            "  Delivery cnt : %s\n"
            "  Ordering key : %s\n"
            "  Attributes   :\n%s\n"
            "  Payload      :\n%s\n"
            "────────────────────────────────────────────────────────────────",
            message.message_id,
            received_at,
            publish_dt,
            message.delivery_attempt if message.delivery_attempt else "N/A",
            message.ordering_key or "(none)",
            "\n".join(f"    {k}: {v}" for k, v in attrs.items()) if attrs else "    (none)",
            "\n".join(f"    {line}" for line in payload.splitlines()),
        )

        if ack:
            message.ack()
            logger.debug("Message %s acknowledged.", message.message_id)
        else:
            message.nack()
            logger.debug("Message %s nacked (ack=False).", message.message_id)

    return callback


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Listen to a Google Cloud Pub/Sub subscription and log messages.",
        formatter_class=argparse.ArgumentDefaultsHelpFormatter,
    )
    parser.add_argument(
        "--project", "-p",
        required=True,
        help="Google Cloud project ID.",
    )
    parser.add_argument(
        "--subscription", "-s",
        required=True,
        help="Pub/Sub subscription ID (not the full path).",
    )
    parser.add_argument(
        "--host",
        default="",
        help="Emulator host (e.g. localhost). Leave empty for real GCP.",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=8085,
        help="Emulator port. Only used when --host is set.",
    )
    parser.add_argument(
        "--no-ack",
        action="store_true",
        help="Nack every message instead of acking it (useful for debugging).",
    )
    parser.add_argument(
        "--max-messages",
        type=int,
        default=5,
        help="Maximum number of messages to pull concurrently.",
    )
    return parser.parse_args()


def build_subscriber(host: str, port: int) -> pubsub_v1.SubscriberClient:
    """Create a SubscriberClient, optionally pointed at a local emulator."""
    if host:
        emulator_endpoint = f"{host}:{port}"
        os.environ["PUBSUB_EMULATOR_HOST"] = emulator_endpoint
        logger.info("Using Pub/Sub emulator at %s", emulator_endpoint)
    else:
        # Remove emulator env var if it was set previously
        os.environ.pop("PUBSUB_EMULATOR_HOST", None)
        logger.info("Connecting to real Google Cloud Pub/Sub.")

    return pubsub_v1.SubscriberClient()


def main() -> None:
    args = parse_args()

    subscriber = build_subscriber(args.host, args.port)
    subscription_path = subscriber.subscription_path(args.project, args.subscription)

    flow_control = pubsub_v1.types.FlowControl(max_messages=args.max_messages, max_bytes=1*1024*1024)
    callback = make_callback(ack=not args.no_ack)

    logger.info(
        "Starting listener on subscription: %s  (ack=%s)",
        subscription_path,
        not args.no_ack,
    )

    streaming_pull_future = subscriber.subscribe(
        subscription_path,
        callback=callback,
        flow_control=flow_control,
    )

    # Graceful shutdown on SIGINT / SIGTERM
    def _shutdown(signum, frame):  # noqa: ANN001
        logger.info("Shutdown signal received, cancelling subscriber…")
        streaming_pull_future.cancel()
        streaming_pull_future.result(timeout=10)
        logger.info("Subscriber shut down cleanly.")
        sys.exit(0)

    signal.signal(signal.SIGINT, _shutdown)
    signal.signal(signal.SIGTERM, _shutdown)

    logger.info("Waiting for messages. Press Ctrl-C to stop.")
    try:
        streaming_pull_future.result()          # blocks until cancelled or error
    except NotFound:
        logger.error(
            "Subscription '%s' not found in project '%s'. "
            "Check the subscription name and project ID.",
            args.subscription,
            args.project,
        )
        sys.exit(1)
    except GoogleAPICallError as exc:
        logger.error("API error: %s", exc)
        sys.exit(1)
    except Exception as exc:  # noqa: BLE001
        logger.error("Unexpected error: %s", exc, exc_info=True)
        sys.exit(1)


if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""
Simple Google Pub/Sub subscriber that displays messages.
Works with both real Pub/Sub and local emulator.
"""

import argparse
import json
import sys
from google.cloud import pubsub_v1
import os


def callback(message):
    """Process received message."""
    print(f"CALLBACK TRIGGERED", flush=True)
    print(f"\n{'='*60}")
    print(f"Message ID: {message.message_id}")
    print(f"Publish time: {message.publish_time}")

    if message.attributes:
        print(f"\nAttributes:")
        for key, value in message.attributes.items():
            print(f"  {key}: {value}")

    print(f"\nPayload:")
    try:
        # Try to pretty-print as JSON
        payload = json.loads(message.data.decode('utf-8'))
        print(json.dumps(payload, indent=2))
    except (json.JSONDecodeError, UnicodeDecodeError):
        # Fall back to raw bytes
        print(message.data)

    print(f"{'='*60}\n")

    # Acknowledge the message
    message.ack()
    print(f"✓ Acknowledged message {message.message_id}")


def main():
    parser = argparse.ArgumentParser(
        description='Subscribe to Google Pub/Sub topic and display messages'
    )
    parser.add_argument('--project', required=True, help='GCP project ID')
    parser.add_argument('--subscription', required=True, help='Subscription name')
    parser.add_argument('--host', default=None, help='Emulator host (e.g., localhost)')
    parser.add_argument('--port', default='9200', help='Emulator port (default: 9200)')

    args = parser.parse_args()

    # Set emulator host if specified
    if args.host:
        emulator_host = f"{args.host}:{args.port}"
        os.environ['PUBSUB_EMULATOR_HOST'] = emulator_host
        print(f"Using emulator: {emulator_host}")

    subscription_path = f"projects/{args.project}/subscriptions/{args.subscription}"
    print(f"Subscribing to: {subscription_path}")
    print("Waiting for messages... (Ctrl+C to exit)\n")

    subscriber = pubsub_v1.SubscriberClient()

    streaming_pull_future = subscriber.subscribe(subscription_path, callback=callback)

    try:
        # Wait for messages indefinitely
        streaming_pull_future.result()
    except KeyboardInterrupt:
        print("\n\nShutting down...")
        streaming_pull_future.cancel()
        streaming_pull_future.result()  # Wait for cancellation to complete
        print("Subscriber stopped.")


if __name__ == '__main__':
    main()
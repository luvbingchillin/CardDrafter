import json
import logging
import os
import threading
import time
import redis

logger = logging.getLogger(__name__)

STREAM_NAME = "stream:pack_openings"
GROUP_NAME = "analytics_group"
CONSUMER_NAME = "worker_1"


def init_redis() -> redis.Redis:
    """Initializes and returns a Redis client configured via environment variables."""
    host = os.getenv("REDIS_HOST", "localhost")
    port = int(os.getenv("REDIS_PORT", 6379))
    return redis.Redis(host=host, port=port, decode_responses=True)


def setup_consumer_group(r: redis.Redis) -> None:
    """
    Ensures the consumer group exists on the target stream.
    
    If the group already exists (BUSYGROUP), the exception is safely ignored.
    mkstream=True ensures the stream is initialized if no events have been pushed yet.
    """
    try:
        r.xgroup_create(name=STREAM_NAME, groupname=GROUP_NAME, id="0", mkstream=True)
        logger.info(f"Consumer group '{GROUP_NAME}' created.")
    except redis.exceptions.ResponseError as e:
        if "BUSYGROUP" in str(e):
            logger.info(f"Consumer group '{GROUP_NAME}' already initialized.")
        else:
            raise e


def process_batch(events: list) -> None:
    """
    Parses and deserializes a batch of raw Redis Stream messages.

    Args:
        events: List of (message_id, field_dict) tuples returned by XREADGROUP.
    """
    records = []
    for message_id, data in events:
        card_ids = json.loads(data.get("card_ids", "[]"))
        records.append({
            "message_id": message_id,
            "user_id": data.get("user_id"),
            "set_code": data.get("set_code"),
            "card_count": int(data.get("card_count", 0)),
            "opened_at": data.get("opened_at"),
            "card_ids": card_ids,
        })

    logger.info(f"Processing batch of {len(records)} pack open events")
    # TODO: Ingest records into Pandas DataFrame or persist aggregates to MongoDB.


def run_stream_consumer(stop_event: threading.Event) -> None:
    """
    Continuous background worker loop that consumes events from Redis Streams.

    1. Retries connection and group creation until Redis is ready.
    2. Long-polls for new undelivered messages (">") up to 100 at a time.
    3. Invokes batch processing and acknowledges (XACK) processed message IDs.
    """
    r = init_redis()

    # Wait for Redis container to become reachable
    while not stop_event.is_set():
        try:
            setup_consumer_group(r)
            break
        except redis.exceptions.ConnectionError:
            logger.warning("Redis unavailable. Retrying consumer group setup in 2s...")
            time.sleep(2)

    # Main event consumption loop
    while not stop_event.is_set():
        try:
            response = r.xreadgroup(
                groupname=GROUP_NAME,
                consumername=CONSUMER_NAME,
                streams={STREAM_NAME: ">"},
                count=100,
                block=2000,
            )

            if not response:
                continue

            for stream_name, messages in response:
                if messages:
                    process_batch(messages)
                    msg_ids = [msg_id for msg_id, _ in messages]
                    r.xack(STREAM_NAME, GROUP_NAME, *msg_ids)

        except Exception as e:
            logger.error(f"Error in Redis stream consumer loop: {e}")
            time.sleep(1)


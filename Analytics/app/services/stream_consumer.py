import json
import logging
import os
import threading
import time
import redis
from datetime import datetime, timezone
from pymongo import MongoClient
import pandas as pd

_mongo_client: MongoClient | None = None
_indexes_ensured: bool = False


def get_mongo_db():
    """Returns a singleton PyMongo database handle, reusing the client connection pool."""
    global _mongo_client, _indexes_ensured
    if _mongo_client is None:
        mongo_uri = os.getenv("MONGODB_URI", "mongodb://localhost:27017")
        _mongo_client = MongoClient(mongo_uri, serverSelectionTimeoutMS=5000)
    db = _mongo_client["cardsim_db"]
    if not _indexes_ensured:
        try:
            db["roll_logs"].create_index("message_id", unique=True)
            _indexes_ensured = True
        except Exception as e:
            logger.warning(f"Could not ensure unique index on roll_logs.message_id: {e}")
    return db


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
    Idempotently parses a batch of Redis Stream events, filters out any already-recorded
    message_ids in 'roll_logs', aggregates pull counts using Pandas, and updates
    lifetime statistics in MongoDB.
    """
    if not events:
        return
    db = get_mongo_db()

    # 1. Deduplicate against roll_logs so PEL retries never double-count $inc analytics
    batch_msg_ids = [message_id for message_id, _ in events]
    existing_ids = set(
        db["roll_logs"].distinct("message_id", {"message_id": {"$in": batch_msg_ids}})
    )

    records = []
    audit_logs = []

    # 2. Parse and deserialize only unrecorded Redis stream messages
    for message_id, data in events:
        if message_id in existing_ids:
            continue

        card_ids = json.loads(data.get("card_ids", "[]"))
        rarities = json.loads(data.get("rarities", "[]"))
        set_code = data.get("set_code", "UNKNOWN")
        user_id = data.get("user_id", "ANONYMOUS")
        opened_at = data.get("opened_at", datetime.now(timezone.utc).isoformat())

        audit_logs.append({
            "message_id": message_id,
            "user_id": user_id,
            "set_code": set_code,
            "card_count": int(data.get("card_count", 0)),
            "opened_at": opened_at,
            "card_ids": card_ids,
            "rarities": rarities,
        })

        for rarity in rarities:
            records.append({
                "set_code": set_code,
                "rarity": rarity.lower(),
            })

    if not audit_logs:
        logger.info("Batch already processed (idempotent skip).")
        return

    # 3. Insert raw historical rolls into 'roll_logs'
    db["roll_logs"].insert_many(audit_logs, ordered=False)

    # 4. Aggregate rarity counts per set using Pandas
    if records:
        df = pd.DataFrame(records)
        rarity_counts = (
            df.groupby(["set_code", "rarity"])
            .size()
            .unstack(fill_value=0)
        )
        packs_df = pd.DataFrame(audit_logs)
        packs_per_set = packs_df["set_code"].value_counts().to_dict()

        # 5. Atomically increment ($inc) lifetime counters in MongoDB
        for set_code, pack_count in packs_per_set.items():
            mythics = int(rarity_counts.loc[set_code].get("mythic", 0)) if set_code in rarity_counts.index else 0
            rares = int(rarity_counts.loc[set_code].get("rare", 0)) if set_code in rarity_counts.index else 0
            uncommons = int(rarity_counts.loc[set_code].get("uncommon", 0)) if set_code in rarity_counts.index else 0
            commons = int(rarity_counts.loc[set_code].get("common", 0)) if set_code in rarity_counts.index else 0
            db["set_analytics"].update_one(
                {"set_code": set_code},
                {
                    "$inc": {
                        "total_packs_opened": pack_count,
                        "mythic_count": mythics,
                        "rare_count": rares,
                        "uncommon_count": uncommons,
                        "common_count": commons,
                    },
                    "$set": {
                        "last_updated": datetime.now(timezone.utc),
                    },
                },
                upsert=True,
            )
    logger.info(f"Successfully processed batch of {len(audit_logs)} pack open events.")


def drain_pending_messages(r: redis.Redis, stop_event: threading.Event) -> None:
    """
    Phase 1 Recovery: Drains any unacknowledged messages sitting in the consumer group's
    Pending Entries List (PEL) from a prior crash using stream ID '0'.
    """
    while not stop_event.is_set():
        response = r.xreadgroup(
            groupname=GROUP_NAME,
            consumername=CONSUMER_NAME,
            streams={STREAM_NAME: "0"},
            count=100,
        )
        if not response:
            break

        has_pending = False
        for stream_name, messages in response:
            if messages:
                has_pending = True
                logger.info(f"Recovering {len(messages)} unacknowledged PEL messages...")
                process_batch(messages)
                msg_ids = [msg_id for msg_id, _ in messages]
                r.xack(STREAM_NAME, GROUP_NAME, *msg_ids)

        if not has_pending:
            break


def run_stream_consumer(stop_event: threading.Event) -> None:
    """
    Continuous background worker loop that consumes events from Redis Streams.

    1. Retries connection and group creation until Redis is ready.
    2. Drains any pending unacknowledged messages ('0') left over from previous crashes.
    3. Long-polls for new undelivered messages ('>') up to 100 at a time.
    4. Invokes idempotent batch processing and acknowledges (XACK) processed message IDs.
    """
    r = init_redis()

    # Wait for Redis container to become reachable
    while not stop_event.is_set():
        try:
            setup_consumer_group(r)
            drain_pending_messages(r, stop_event)
            break
        except redis.exceptions.ConnectionError:
            logger.warning("Redis unavailable. Retrying consumer group setup in 2s...")
            time.sleep(2)
        except Exception as e:
            logger.error(f"Error during initial PEL recovery: {e}")
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

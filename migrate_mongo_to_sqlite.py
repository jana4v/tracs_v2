import argparse
import os
import sys
from datetime import date, datetime, time
from decimal import Decimal
from pathlib import Path
from typing import Any

# Ensure the project root is on sys.path so `src.*` imports resolve when running as a script.
sys.path.insert(0, str(Path(__file__).resolve().parent))

from src.config import settings
from src.database.sqlite_json_store import SQLiteJsonCollection


def _serialize_for_json(value: Any) -> Any:
    if isinstance(value, dict):
        return {str(k): _serialize_for_json(v) for k, v in value.items()}
    if isinstance(value, list):
        return [_serialize_for_json(v) for v in value]
    if isinstance(value, tuple):
        return [_serialize_for_json(v) for v in value]
    if isinstance(value, (datetime, date, time)):
        return value.isoformat()
    if isinstance(value, Decimal):
        return float(value)
    if isinstance(value, bytes):
        return value.hex()

    class_name = value.__class__.__name__
    if class_name in {"ObjectId", "Int64", "Decimal128", "Binary", "UUID"}:
        return str(value)

    return value


def _resolve_sqlite_path(sqlite_path: str) -> Path:
    candidate = Path(sqlite_path)
    if candidate.is_absolute():
        return candidate
    return Path(__file__).resolve().parent / candidate


def _parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Copy collections from MongoDB tracs_v2 into SQLite JSON collections."
    )
    parser.add_argument(
        "--mongo-uri",
        default=os.getenv("MONGO_URI", "mongodb://localhost:27017"),
        help="MongoDB URI. Uses MONGO_URI env var when present.",
    )
    parser.add_argument(
        "--mongo-db",
        default=os.getenv("MONGO_DB", "tracs_v2"),
        help="MongoDB database name.",
    )
    parser.add_argument(
        "--sqlite-path",
        default=settings.SQLITE_DB_PATH,
        help="Path to SQLite database file.",
    )
    parser.add_argument(
        "--collections",
        nargs="+",
        default=None,
        help="Specific collection names to copy. Default: all collections in the MongoDB database.",
    )
    return parser.parse_args()


def main() -> int:
    args = _parse_args()

    try:
        from pymongo import MongoClient
    except Exception as exc:  # pragma: no cover
        print("pymongo is required. Install it with: pip install pymongo")
        print(f"Import error: {exc}")
        return 1

    sqlite_db_path = _resolve_sqlite_path(args.sqlite_path)
    sqlite_db_path.parent.mkdir(parents=True, exist_ok=True)

    client = MongoClient(args.mongo_uri, serverSelectionTimeoutMS=5000)
    try:
        client.admin.command("ping")
    except Exception as exc:
        print(f"Unable to connect to MongoDB: {exc}")
        return 1

    mongo_db = client[args.mongo_db]
    collection_names = args.collections or mongo_db.list_collection_names()

    if not collection_names:
        print(f"No collections found in MongoDB database '{args.mongo_db}'.")
        return 0

    print(f"MongoDB database: {args.mongo_db}")
    print(f"SQLite database: {sqlite_db_path}")
    print(f"Collections to migrate: {', '.join(collection_names)}")

    total_docs = 0
    for collection_name in collection_names:
        source_collection = mongo_db[collection_name]
        target_collection = SQLiteJsonCollection(str(sqlite_db_path), collection_name)

        migrated_count = 0
        for raw_doc in source_collection.find({}):
            doc = _serialize_for_json(raw_doc)
            if "_id" not in doc:
                continue

            doc["_id"] = str(doc["_id"])
            target_collection.update_one(
                {"_id": doc["_id"]},
                {"$set": doc},
                upsert=True,
            )
            migrated_count += 1

        total_docs += migrated_count
        print(f"- {collection_name}: {migrated_count} documents migrated")

    print(f"Migration complete. Total migrated documents: {total_docs}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

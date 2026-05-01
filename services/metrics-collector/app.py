import os
import logging
import time
import uuid
from flask import Flask, jsonify, request

app = Flask(__name__)

logging.basicConfig(
    level=os.environ.get("LOG_LEVEL", "INFO"),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("metrics-collector")

AGGREGATOR_URL = os.environ.get("AGGREGATOR_URL", "http://aggregator:8081")

metrics_store: list[dict] = []


@app.route("/health")
def health():
    return jsonify({"status": "ok", "service": "metrics-collector", "timestamp": time.time()})


@app.route("/metrics", methods=["POST"])
def collect_metric():
    data = request.get_json()
    if not data:
        logger.warning("Received empty payload")
        return jsonify({"error": "Request body must be JSON"}), 400

    required = ["name", "value"]
    missing = [f for f in required if f not in data]
    if missing:
        logger.warning("Missing fields: %s", missing)
        return jsonify({"error": f"Missing required fields: {missing}"}), 400

    if not isinstance(data["value"], (int, float)):
        return jsonify({"error": "Field 'value' must be a number"}), 400

    metric = {
        "id": str(uuid.uuid4()),
        "name": data["name"],
        "value": data["value"],
        "tags": data.get("tags", {}),
        "timestamp": time.time(),
    }
    metrics_store.append(metric)
    logger.info("Collected metric: %s = %s", metric["name"], metric["value"])
    return jsonify(metric), 201


@app.route("/metrics", methods=["GET"])
def list_metrics():
    name = request.args.get("name")
    if name:
        filtered = [m for m in metrics_store if m["name"] == name]
        return jsonify(filtered)
    return jsonify(metrics_store)


@app.route("/metrics/flush", methods=["POST"])
def flush_metrics():
    count = len(metrics_store)
    metrics_store.clear()
    logger.info("Flushed %d metrics", count)
    return jsonify({"flushed": count})


if __name__ == "__main__":
    port = int(os.environ.get("PORT", 8080))
    logger.info("Starting metrics-collector on port %d", port)
    app.run(host="0.0.0.0", port=port)

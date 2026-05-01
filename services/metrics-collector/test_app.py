import pytest
from app import app, metrics_store


@pytest.fixture
def client():
    app.config["TESTING"] = True
    with app.test_client() as c:
        metrics_store.clear()
        yield c


def test_health(client):
    resp = client.get("/health")
    assert resp.status_code == 200
    data = resp.get_json()
    assert data["status"] == "ok"
    assert data["service"] == "metrics-collector"
    assert "timestamp" in data


def test_collect_metric(client):
    resp = client.post("/metrics", json={"name": "cpu_usage", "value": 72.5})
    assert resp.status_code == 201
    data = resp.get_json()
    assert data["name"] == "cpu_usage"
    assert data["value"] == 72.5
    assert "id" in data
    assert "timestamp" in data


def test_collect_metric_with_tags(client):
    payload = {"name": "memory", "value": 1024, "tags": {"host": "server-1"}}
    resp = client.post("/metrics", json=payload)
    assert resp.status_code == 201
    assert resp.get_json()["tags"] == {"host": "server-1"}


def test_collect_metric_missing_fields(client):
    resp = client.post("/metrics", json={"name": "cpu"})
    assert resp.status_code == 400
    assert "Missing required fields" in resp.get_json()["error"]


def test_collect_metric_invalid_value(client):
    resp = client.post("/metrics", json={"name": "cpu", "value": "high"})
    assert resp.status_code == 400
    assert "number" in resp.get_json()["error"]


def test_collect_metric_empty_body(client):
    resp = client.post("/metrics", content_type="application/json")
    assert resp.status_code == 400


def test_list_metrics(client):
    client.post("/metrics", json={"name": "a", "value": 1})
    client.post("/metrics", json={"name": "b", "value": 2})
    resp = client.get("/metrics")
    assert resp.status_code == 200
    assert len(resp.get_json()) == 2


def test_list_metrics_filter_by_name(client):
    client.post("/metrics", json={"name": "a", "value": 1})
    client.post("/metrics", json={"name": "b", "value": 2})
    resp = client.get("/metrics?name=a")
    data = resp.get_json()
    assert len(data) == 1
    assert data[0]["name"] == "a"


def test_flush_metrics(client):
    client.post("/metrics", json={"name": "x", "value": 10})
    resp = client.post("/metrics/flush")
    assert resp.status_code == 200
    assert resp.get_json()["flushed"] == 1
    assert len(metrics_store) == 0

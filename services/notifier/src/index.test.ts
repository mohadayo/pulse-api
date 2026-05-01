import request from "supertest";
import { createApp } from "./index";

const app = createApp();

beforeEach(() => {
  createApp();
});

describe("GET /health", () => {
  it("returns ok status", async () => {
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body.status).toBe("ok");
    expect(res.body.service).toBe("notifier");
    expect(res.body.timestamp).toBeDefined();
  });
});

describe("POST /rules", () => {
  it("creates an alert rule", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu_usage",
      threshold: 90,
      operator: "gt",
      channel: "email",
    });
    expect(res.status).toBe(201);
    expect(res.body.metricName).toBe("cpu_usage");
    expect(res.body.threshold).toBe(90);
    expect(res.body.operator).toBe("gt");
    expect(res.body.id).toBeDefined();
  });

  it("rejects missing fields", async () => {
    const res = await request(app).post("/rules").send({ metricName: "cpu" });
    expect(res.status).toBe(400);
    expect(res.body.error).toContain("Missing required fields");
  });

  it("rejects invalid operator", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu",
      threshold: 50,
      operator: "invalid",
      channel: "slack",
    });
    expect(res.status).toBe(400);
    expect(res.body.error).toContain("Invalid operator");
  });

  it("rejects non-numeric threshold", async () => {
    const res = await request(app).post("/rules").send({
      metricName: "cpu",
      threshold: "high",
      operator: "gt",
      channel: "slack",
    });
    expect(res.status).toBe(400);
    expect(res.body.error).toContain("number");
  });
});

describe("GET /rules", () => {
  it("returns all rules", async () => {
    await request(app).post("/rules").send({
      metricName: "mem",
      threshold: 80,
      operator: "gte",
      channel: "slack",
    });
    const res = await request(app).get("/rules");
    expect(res.status).toBe(200);
    expect(Array.isArray(res.body)).toBe(true);
    expect(res.body.length).toBeGreaterThanOrEqual(1);
  });
});

describe("POST /evaluate", () => {
  it("triggers alert when condition is met", async () => {
    await request(app).post("/rules").send({
      metricName: "cpu",
      threshold: 80,
      operator: "gt",
      channel: "webhook",
    });
    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
      value: 95,
    });
    expect(res.status).toBe(200);
    expect(res.body.triggered.length).toBe(1);
    expect(res.body.triggered[0].message).toContain("cpu");
  });

  it("does not trigger when condition is not met", async () => {
    await request(app).post("/rules").send({
      metricName: "cpu",
      threshold: 80,
      operator: "gt",
      channel: "webhook",
    });
    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
      value: 50,
    });
    expect(res.status).toBe(200);
    expect(res.body.triggered.length).toBe(0);
  });

  it("rejects missing fields", async () => {
    const res = await request(app).post("/evaluate").send({ metricName: "cpu" });
    expect(res.status).toBe(400);
  });

  it("rejects non-numeric value", async () => {
    const res = await request(app).post("/evaluate").send({
      metricName: "cpu",
      value: "high",
    });
    expect(res.status).toBe(400);
    expect(res.body.error).toContain("number");
  });
});

describe("GET /notifications", () => {
  it("returns notification history", async () => {
    await request(app).post("/rules").send({
      metricName: "disk",
      threshold: 90,
      operator: "gte",
      channel: "pager",
    });
    await request(app).post("/evaluate").send({
      metricName: "disk",
      value: 95,
    });
    const res = await request(app).get("/notifications");
    expect(res.status).toBe(200);
    expect(res.body.length).toBeGreaterThanOrEqual(1);
  });
});

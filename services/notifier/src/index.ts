import express, { Request, Response } from "express";

const app = express();
app.use(express.json());

const PORT = parseInt(process.env.PORT || "8082", 10);
const LOG_LEVEL = process.env.LOG_LEVEL || "info";

interface AlertRule {
  id: string;
  metricName: string;
  threshold: number;
  operator: "gt" | "lt" | "gte" | "lte" | "eq";
  channel: string;
}

interface Notification {
  id: string;
  ruleId: string;
  metricName: string;
  value: number;
  message: string;
  channel: string;
  timestamp: number;
}

const alertRules: AlertRule[] = [];
const notifications: Notification[] = [];
let ruleCounter = 0;
let notifCounter = 0;

function log(level: string, message: string): void {
  if (level === "debug" && LOG_LEVEL !== "debug") return;
  const ts = new Date().toISOString();
  console.log(`${ts} [${level.toUpperCase()}] notifier: ${message}`);
}

function evaluateRule(rule: AlertRule, value: number): boolean {
  switch (rule.operator) {
    case "gt":
      return value > rule.threshold;
    case "lt":
      return value < rule.threshold;
    case "gte":
      return value >= rule.threshold;
    case "lte":
      return value <= rule.threshold;
    case "eq":
      return value === rule.threshold;
    default:
      return false;
  }
}

app.get("/health", (_req: Request, res: Response) => {
  res.json({
    status: "ok",
    service: "notifier",
    timestamp: Date.now(),
  });
});

app.post("/rules", (req: Request, res: Response) => {
  const { metricName, threshold, operator, channel } = req.body;

  if (!metricName || threshold === undefined || !operator || !channel) {
    log("warn", "Missing required fields in rule creation");
    res.status(400).json({ error: "Missing required fields: metricName, threshold, operator, channel" });
    return;
  }

  const validOps = ["gt", "lt", "gte", "lte", "eq"];
  if (!validOps.includes(operator)) {
    res.status(400).json({ error: `Invalid operator. Must be one of: ${validOps.join(", ")}` });
    return;
  }

  if (typeof threshold !== "number") {
    res.status(400).json({ error: "threshold must be a number" });
    return;
  }

  ruleCounter++;
  const rule: AlertRule = {
    id: `rule-${ruleCounter}`,
    metricName,
    threshold,
    operator,
    channel,
  };
  alertRules.push(rule);
  log("info", `Created alert rule: ${rule.id} (${metricName} ${operator} ${threshold})`);
  res.status(201).json(rule);
});

app.get("/rules", (_req: Request, res: Response) => {
  res.json(alertRules);
});

app.post("/evaluate", (req: Request, res: Response) => {
  const { metricName, value } = req.body;

  if (!metricName || value === undefined) {
    res.status(400).json({ error: "Missing required fields: metricName, value" });
    return;
  }

  if (typeof value !== "number") {
    res.status(400).json({ error: "value must be a number" });
    return;
  }

  const matchingRules = alertRules.filter((r) => r.metricName === metricName);
  const triggered: Notification[] = [];

  for (const rule of matchingRules) {
    if (evaluateRule(rule, value)) {
      notifCounter++;
      const notif: Notification = {
        id: `notif-${notifCounter}`,
        ruleId: rule.id,
        metricName,
        value,
        message: `Alert: ${metricName} = ${value} (${rule.operator} ${rule.threshold})`,
        channel: rule.channel,
        timestamp: Date.now(),
      };
      notifications.push(notif);
      triggered.push(notif);
      log("info", `Triggered: ${notif.message} -> ${rule.channel}`);
    }
  }

  log("debug", `Evaluated ${matchingRules.length} rules for ${metricName}, ${triggered.length} triggered`);
  res.json({ evaluated: matchingRules.length, triggered });
});

app.get("/notifications", (_req: Request, res: Response) => {
  res.json(notifications);
});

export function createApp() {
  ruleCounter = 0;
  notifCounter = 0;
  alertRules.length = 0;
  notifications.length = 0;
  return app;
}

if (require.main === module) {
  app.listen(PORT, () => {
    log("info", `Starting notifier on port ${PORT}`);
  });
}

export default app;

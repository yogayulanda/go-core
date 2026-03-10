import http from "k6/http";
import { check, sleep } from "k6";

const scenario = __ENV.SCENARIO || "steady";
const failRate = Number(__ENV.FAIL_RATE || "0.01");
const p95 = Number(__ENV.P95_MS || "500");
const p99 = Number(__ENV.P99_MS || "1000");

function scenarioOptions() {
  switch (scenario) {
    case "spike":
      return {
        stages: [
          { duration: "30s", target: 20 },
          { duration: "30s", target: 200 },
          { duration: "1m", target: 200 },
          { duration: "30s", target: 20 },
        ],
      };
    case "soak":
      return {
        stages: [
          { duration: "2m", target: 50 },
          { duration: "30m", target: 50 },
          { duration: "2m", target: 0 },
        ],
      };
    case "steady":
    default:
      return {
        stages: [
          { duration: "1m", target: 50 },
          { duration: "5m", target: 50 },
          { duration: "1m", target: 0 },
        ],
      };
  }
}

export const options = {
  ...scenarioOptions(),
  thresholds: {
    http_req_failed: [`rate<${failRate}`],
    http_req_duration: [`p(95)<${p95}`, `p(99)<${p99}`],
  },
};

const baseURL = __ENV.BASE_URL;
const targetPath = __ENV.TARGET_PATH || "/health";

if (!baseURL) {
  throw new Error("BASE_URL is required");
}

export default function () {
  const res = http.get(`${baseURL}${targetPath}`, {
    timeout: "30s",
    tags: { scenario },
  });

  check(res, {
    "status is 200": (r) => r.status === 200,
  });

  sleep(1);
}

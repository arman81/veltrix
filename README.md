# Veltrix

<img width="2752" height="1536" alt="Gemini_Generated_Image_2jwqtm2jwqtm2jwq" src="https://github.com/user-attachments/assets/a745815f-73c6-4b4c-bb56-1ac5f96733a5" />



# Scheduler Archiecture 

<img width="2022" height="1322" alt="veltrix-master-architecture" src="https://github.com/user-attachments/assets/5c811226-1640-434e-bd5a-d2bacf89bfe2" />





# Node Architecture

<img width="2772" height="1522" alt="veltrix-node-architecture" src="https://github.com/user-attachments/assets/d494dee1-b32b-40a8-963c-926ca6ffd8f2" />

# Local Dev Quickstart

Brings up the simulator (fake GPU metrics), Prometheus (scraper/TSDB), and Grafana (provisioned dashboards + alerts).

## Prerequisites

- Docker Desktop (or Docker Engine + Compose v2) running
- Ports free on host: `3000` (Grafana), `9090` (Prometheus), `9100` (simulator)
- Optional: `jq`, `curl` for the smoke tests below

## Bring the stack up

```bash
# from repo root
docker compose pull            # first run only — fetches grafana 11.3.0, prometheus, etc.
docker compose up -d           # start simulator + prometheus + grafana in background
docker compose ps              # confirm all three services healthy
```

`postgres` and `redis` are gated behind the `full` profile and are NOT started by default. To include them:

```bash
docker compose --profile full up -d
```

## Access the services

| Service    | URL                       | Credentials       |
|------------|---------------------------|-------------------|
| Grafana    | http://localhost:3000     | `admin` / `veltrix` |
| Prometheus | http://localhost:9090     | none              |
| Simulator  | http://localhost:9100/metrics | none          |

The Grafana home dashboard is **Veltrix — GPU Cluster Overview**. The `Veltrix` folder also contains: Training Jobs, Cost & Efficiency, Scheduler Internals, SLOs & Alerts.

## Smoke test

```bash
# simulator emits metrics
curl -s http://localhost:9100/metrics | grep -c '^# TYPE veltrix_'
# expect: ~38

# prometheus has scraped them
curl -s 'http://localhost:9090/api/v1/query?query=veltrix_cluster_gpus_total' | jq '.data.result | length'
# expect: 1

# grafana healthy + datasource provisioned
curl -s http://localhost:3000/api/health | jq -r .database
curl -s -u admin:veltrix http://localhost:3000/api/datasources/uid/prometheus-veltrix | jq -r '.name + " " + .type'
# expect: "ok" then "Prometheus prometheus"

# all 5 dashboards reachable
for uid in veltrix-gpu-overview veltrix-training-jobs veltrix-cost-efficiency veltrix-scheduler-internals veltrix-slo-alerts; do
  curl -s -o /dev/null -w "$uid %{http_code}\n" -u admin:veltrix "http://localhost:3000/api/dashboards/uid/$uid"
done
# expect: each line ends with 200

# 7 alert rules provisioned
curl -s -u admin:veltrix http://localhost:3000/api/v1/provisioning/alert-rules | jq 'length'
# expect: 7
```

## Tail logs

```bash
docker compose logs -f grafana       # provisioning + alert evaluation
docker compose logs -f prometheus    # scrape errors
docker compose logs -f simulator     # request log
```

## Iterating on dashboards / alerts / metrics

| Change                                     | How it picks up                              |
|--------------------------------------------|----------------------------------------------|
| Edit `config/grafana/dashboards/*.json`    | Auto-reloaded by Grafana (no restart needed) |
| Edit `config/grafana/provisioning/**`      | `docker compose restart grafana`             |
| Edit `cmd/simulator/main.go`               | `docker compose up -d --build simulator`     |
| Edit `config/prometheus.yaml`              | `docker compose restart prometheus`          |
| Change Grafana plugins / image / volumes   | `docker compose up -d --force-recreate grafana` |

## Tear down

```bash
docker compose down              # stop containers, keep named volumes
docker compose down -v           # also wipe grafana_data + prometheus_data + postgres_data
```

## Build the simulator binary directly (no Docker)

```bash
go build -o bin/veltrix-simulator ./cmd/simulator
./bin/veltrix-simulator   # listens on :9100
```

In this mode Prometheus and Grafana still need to be running (e.g. `docker compose up -d prometheus grafana`) and `config/prometheus.yaml` already targets `simulator:9100`. To scrape a host-side simulator, edit `prometheus.yaml` to target `host.docker.internal:9100` and restart Prometheus.

## Troubleshooting

| Symptom                                              | Fix                                                                 |
|------------------------------------------------------|---------------------------------------------------------------------|
| Grafana stuck `(unhealthy)` >90s                     | `docker compose logs grafana --tail=200` — first error wins         |
| Dashboard panels show "No data"                      | Check `http://localhost:9090/targets` — simulator must be `UP`      |
| `port already allocated`                             | Another process on 3000/9090/9100; `lsof -i :3000` then kill it     |
| Alert rules show `Error` state                       | `docker compose logs grafana \| grep -i 'rule evaluator'`           |
| Plugin install fails on first start                  | First-run network egress required; retry `docker compose up -d`     |

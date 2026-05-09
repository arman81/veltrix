# Veltrix — Product Build Plan

## 0. North star

**Veltrix is the neutral cost/perf optimization layer for heterogeneous AI workloads.** Inference-first cross-silicon routing today; training-extending as CUDA lock-in genuinely opens.

The defensible moat is **data + switching costs**, not algorithms:
1. Workload-silicon performance telemetry accumulated across customers (network effect).
2. Validated integration paths per silicon × framework × runtime (engineering-hour wall).
3. Job state + audit + billing integration (boring switching cost).

## 1. What is already built

| Component | Path | Status |
|---|---|---|
| API binary skeleton | `cmd/api/main.go` | scaffold |
| Agent binary skeleton | `cmd/agent/main.go` | scaffold |
| Simulator (24 metric families) | `cmd/simulator/main.go` | working — emits GPU + training + scheduler + cost + SLO + interconnect metrics |
| Control-plane package layout | `internal/controlplane/{scheduler,placement,policy,feedback,api,domain}` | empty dirs |
| Data-plane package layout | `internal/dataplane/{cache,queue,repository,telemetry}` | empty dirs |
| Proto definitions | `proto/metrics.proto` | minimal (metrics only) |
| Grafana integration | `config/grafana/{dashboards,provisioning}/` | **complete** — 5 dashboards, 7 alert rules, contact points, plugin install, provisioned datasource |
| Docker compose stack | `docker-compose.yml` | grafana 11.3.0 + prometheus + simulator boot cleanly |

## 2. What we are explicitly NOT building

| Anti-goal | Why |
|---|---|
| CUDA → non-NVIDIA automated porting | Compiler problem (Modular/OctoML territory). $100M+ R&D bet we won't win. |
| Training-time live silicon migration | >95% of training stacks are CUDA-locked; portability claim cannot be honored. |
| Custom GPU fractionalizing scheme | Vendors are commoditizing this themselves (MIG, NeuronLink partitions, MI300 SR-IOV). Feature, not moat. |
| Foundation model training | Frontier-lab game; needs $5–10B capital. |
| Owning a GPU fleet (yet) | Capex + supply relationships gate this. Revisit after anchor tenant signed. |
| New silicon | NVIDIA/AMD/Trainium territory. |

## 3. Phases — what we build, in order

Each phase is independently shippable. Do NOT start phase N+1 until N's success criteria are met.

### Phase A — Scheduler core (months 0–3)

**Goal:** A real, runnable scheduler that accepts jobs via API and places them on simulated GPUs.

| Ticket | Files | Acceptance |
|---|---|---|
| A1. Define core domain types | `internal/controlplane/domain/{job,gpu,placement}.go` | Job, GPU, NodeState, PlacementDecision structs with stable JSON tags |
| A2. Job submission gRPC + REST | `proto/scheduler.proto`, `internal/controlplane/api/{submit,status,cancel}.go` | `veltrixctl submit` returns job_id; `veltrixctl status` reports state transitions |
| A3. In-memory queue with priority | `internal/dataplane/queue/memory.go` | Priority + FIFO within priority; tested under concurrency |
| A4. Scheduler loop | `internal/controlplane/scheduler/loop.go` | Pulls from queue, calls placement, persists decision, transitions job state |
| A5. Placement scorer v0 (greedy bin-pack, topology-aware) | `internal/controlplane/placement/{scorer,topology}.go` | NVLink-affinity for TP shards; PP stages on adjacent IB nodes; deterministic given input |
| A6. Repository layer (Postgres) | `internal/dataplane/repository/postgres/{jobs,placements,events}.go` | sqlc or pgx; migrations under `migrations/` |
| A7. CLI `veltrixctl` | `cmd/veltrixctl/main.go` | submit / status / cancel / list commands |
| A8. Wire scheduler into existing simulator topology | `cmd/api/main.go` | `docker compose up` runs API, scheduler picks simulated GPUs |
| A9. Unit + integration tests | `internal/.../*_test.go` + `test/integration/scheduler_test.go` | `make test` passes; ≥70% coverage on scheduler/placement |
| A10. Grafana dashboard for scheduler internals | already done — verify alerts fire on real load | scheduler-internals dashboard panels populated by real data, not just simulator |

**Success criteria:** 1k synthetic jobs submitted via CLI complete with correct placement decisions; scheduler p99 decision latency <500ms; no crashes under 10× load burst.

### Phase B — Cost optimization (months 3–6)

**Goal:** Veltrix can place jobs across multiple cost tiers (on-demand / reserved / spot) with budget controls.

| Ticket | Files | Acceptance |
|---|---|---|
| B1. Cost model abstraction | `internal/controlplane/policy/cost/{model,prices}.go` | Pluggable price feeds per cloud × instance × tier × region |
| B2. Spot / preempt awareness in placement | `internal/controlplane/placement/cost_scorer.go` | Multi-objective score: $/hr × success-prob × queue-time |
| B3. Per-tenant budget caps + quotas | `internal/controlplane/policy/{tenant,budget}.go` | Soft + hard caps; hard cap rejects new jobs; soft cap warns |
| B4. Workload fingerprinting | `internal/controlplane/feedback/fingerprint.go` | Compute deterministic hash from {framework, parallelism, batch, model arch, dataset shape}; persist with every job |
| B5. AWS / GCP / Azure / Lambda / CoreWeave price feeds | `internal/dataplane/prices/{aws,gcp,azure,lambda,coreweave}.go` | Refresh every 15 min; fallback cache; cost dashboard now reads real prices |
| B6. Reservation tracker | `internal/controlplane/policy/reservation.go` | Track committed capacity; prefer it before spot/on-demand |
| B7. Cost dashboards now use real numbers | already exists — wire to real price feed | savings vs baseline panel reflects actual placement choices |
| B8. Drain rebalancer | `internal/controlplane/scheduler/rebalancer/drain.go` | GPU health signal → evacuate restartable jobs to healthy capacity |

**Success criteria:** Synthetic 30-day workload mix shows ≥15% cost savings vs static placement on the same cluster topology.

### Phase C — Cross-silicon inference routing (months 6–12)

**Goal:** A job submitted as "serve this model with these SLOs" routes to the cheapest silicon meeting the SLO across NVIDIA + AMD + Trainium + Groq.

| Ticket | Files | Acceptance |
|---|---|---|
| C1. Silicon abstraction interface | `internal/dataplane/silicon/silicon.go` | `Silicon` trait: capabilities, runtime adapters, perf model hook |
| C2. NVIDIA adapter (vLLM, TensorRT-LLM, SGLang) | `internal/dataplane/silicon/nvidia/` | Health check, runtime selection, perf calibration on first run |
| C3. AMD adapter (vLLM-ROCm, SGLang-ROCm) | `internal/dataplane/silicon/amd/` | Same shape; only models with validated parity |
| C4. AWS Trainium adapter (Neuron SDK) | `internal/dataplane/silicon/trainium/` | Inferentia2/Trn1 supported; inference-only |
| C5. Groq adapter (GroqCloud API) | `internal/dataplane/silicon/groq/` | API-only adapter; no node management |
| C6. Cross-silicon perf prediction model | `internal/controlplane/feedback/predictor.go` | Workload fingerprint → predicted (latency, throughput, $/req) per silicon; calibrated on real telemetry |
| C7. Quality-equivalence gate | `internal/controlplane/policy/equivalence.go` | Refuse to route to a silicon if validated output divergence >ε for the model |
| C8. Inference router service | `cmd/inference-router/main.go` | OpenAI-compatible endpoint; routes per-request based on (cost, latency-SLO, equivalence) |
| C9. Per-customer model registry | `internal/controlplane/domain/model.go` + repo layer | Customer registers model + accepted silicons; quality-validation harness |

**Success criteria:** Veltrix-routed inference is ≥20% cheaper than naive single-silicon serving on the same SLO, demonstrated on 5 reference models (Llama-3-70B, Mixtral-8x7B, Qwen-72B, Phi-3, an embedding model).

### Phase D — Data moat (runs concurrent with B + C, months 3–18)

**Goal:** Anonymized cross-customer telemetry that no newcomer can replicate.

| Ticket | Files | Acceptance |
|---|---|---|
| D1. Telemetry opt-in framework | `internal/dataplane/telemetry/optin.go` | Per-tenant flag; legal terms; data minimization (no model weights, no input data — only fingerprints + measurements) |
| D2. Anonymized fingerprint export | `internal/dataplane/telemetry/export.go` | Daily roll-up to long-term store; PII-stripped |
| D3. Long-term store | `internal/dataplane/telemetry/warehouse/` | Postgres → ClickHouse for query; partitioned by month |
| D4. Public benchmark API (opt-in customers see industry comparisons) | `cmd/benchmark-api/main.go` | "How does my workload compare to similar workloads on similar silicon?" |
| D5. Predictor retraining pipeline | `internal/controlplane/feedback/retrain.go` | Weekly model refresh from anonymized warehouse |
| D6. "Which silicon for your job" recommendation API (sellable standalone) | `cmd/silicon-advisor/main.go` | Free tier for marketing; paid tier for enterprise |

**Success criteria:** ≥1M scored workload-runs in warehouse by month 18; predictor median absolute % error on $/perf prediction <12% vs measured.

### Phase E — Lock-in surfaces (months 9–18)

**Goal:** Once installed, Veltrix is expensive to remove.

| Ticket | Files | Acceptance |
|---|---|---|
| E1. Job state + checkpoint registry | `internal/dataplane/repository/checkpoints/` | Periodic checkpoints + provenance; restart from any checkpoint on any silicon (where compatible) |
| E2. Audit log (immutable, signed) | `internal/dataplane/repository/audit/` | Every placement, preemption, cost decision recorded with reason codes; tamper-evident |
| E3. Tenant billing integration | `cmd/billing/main.go` + Stripe / NetSuite hooks | Per-job, per-customer, per-silicon, per-cost-tier line items; chargeback exports |
| E4. Multi-region failover | `internal/controlplane/scheduler/failover.go` | Region-down detection + automatic re-placement of restartable jobs |
| E5. Compliance reporting | `internal/dataplane/repository/compliance/` | SOC2 / HIPAA / EU-AI-Act report generators |

**Success criteria:** Switching cost (estimated engineering hours to remove Veltrix and recreate equivalent state elsewhere) ≥3 person-months for a mid-tier customer. Validate by interviewing 3 customers post-deployment.

## 4. Cross-cutting infrastructure

These are not phase-gated — they happen continuously.

| Item | Where | When |
|---|---|---|
| CI (GitHub Actions): build, lint, vet, test, race | `.github/workflows/` | Phase A |
| Helm chart for Veltrix on K8s | `deploy/helm/veltrix/` | Phase A end |
| Terraform module for AWS / GCP / Azure deployment | `deploy/terraform/` | Phase B |
| OpenTelemetry tracing across components | already wired in `config/otel-collector.yaml`; expand to control plane | Phase A |
| Security: mTLS between components, RBAC, secret management | `internal/controlplane/auth/` | Phase A end |
| Documentation: developer + operator + tenant guides | `docs/` | Each phase |
| Performance regression suite | `test/perf/` | Phase B |
| Chaos testing (kill nodes, drop links) | `test/chaos/` | Phase C |

## 5. Headcount expectation by phase

Approximate FTE allocation; assumes start with 4-person founding team.

| Phase | Months | Total FTE needed | Composition |
|---|---|---|---|
| A | 0–3 | 4 | 2 backend Go, 1 SRE/infra, 1 founder/PM |
| B | 3–6 | 6 | + 1 backend, + 1 data eng |
| C | 6–12 | 10 | + 2 ML systems, + 1 silicon-vendor partnerships, + 1 backend |
| D | 3–18 | parallel +1 | 1 ML eng on predictor + warehouse |
| E | 9–18 | parallel +2 | 1 backend, 1 compliance/security |

By month 18: ~14 engineers + 2 founders + 2 GTM. Burn ~$5M/yr.

## 6. Capital alignment

| Stage | Funding bucket | Use |
|---|---|---|
| Pre-seed → Seed ($3–6M) | Phase A + start of B | Get scheduler running on real GPU cluster (rented), 2 design partners |
| Seed → Series A ($15–25M) | Finish B, run C, start D | Cross-silicon inference live; 5 paying customers |
| Series A → B ($40–80M) | Finish C, D, E | Repeatable GTM; $5–10M ARR; expand to enterprise |

## 7. Honest revenue ceiling

Per the conversation that produced this plan:

| Path | Plan execution required | 5-year ARR ceiling |
|---|---|---|
| A. Software only (current plan) | Phases A–E above | $50–150M |
| B. Aggregator / broker (asset-light) | + acquire idle capacity contracts | $200–500M |
| C. Anchor-tenant CoreWeave clone | requires anchor + $300M+ raise + supply-side hire | $1–4B (4–5 year ramp) |

**This plan delivers Path A.** Pivoting to B or C requires non-engineering work (fundraising, tenant hunting, NVIDIA relationship building) that cannot be planned in the codebase. Re-open the discussion if/when those become viable.

## 8. Decision gates — when to stop and re-plan

Before starting each phase, validate:

| Gate | Question | If "no" |
|---|---|---|
| Before Phase B | Did Phase A produce ≥1 paying or LOI design partner? | Pause. Talk to customers. The scheduler isn't valuable in isolation. |
| Before Phase C | Are ≥3 customers asking for cross-silicon routing specifically? | Skip C. Double down on D and E. Inference routing ahead of demand burns runway. |
| Before Phase E | Is renewal rate <90%? | Stop scaling. Fix product before lock-in is worth building. |
| End of month 18 | ARR <$2M? | Path A is not validated. Decide: pivot to aggregator (B), shut down, or sell scheduler IP. |

## 9. First two weeks — tickets to start tomorrow

1. Land Phase A1 (domain types) and A2 skeleton (proto + handler stubs).
2. Wire the scheduler loop A4 against the in-memory queue A3 — even if placement is `random()` for now.
3. End-to-end test: `veltrixctl submit` → scheduler picks → status returns "scheduled". Single test that exercises the full path.
4. Set up CI before phase A merges accumulate.

This unblocks every subsequent ticket. Begin here.

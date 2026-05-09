// Command simulator generates fake GPU telemetry for development.
//
// Exposes a /metrics endpoint in Prometheus text format. Prometheus scrapes
// this endpoint, and Grafana visualizes the data.
//
// Simulates a 3-node cluster with 4 GPUs each (12 GPUs total).
// Metrics fluctuate realistically over time to make dashboards look alive.
//
// This is a development-only tool. In production, real NVML telemetry
// flows through the OTel pipeline.
package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Simulated cluster topology
// ---------------------------------------------------------------------------

type simulatedGPU struct {
	nodeID      string
	gpuID       string
	deviceIndex int
	model       string
	vramTotal   int64 // bytes
	strategy    string
	hasJob      bool
	jobType     string
}

var cluster = []simulatedGPU{
	// Node 1: 4x A100-80GB — heavy training node
	{nodeID: "node-1", gpuID: "gpu-1a", deviceIndex: 0, model: "a100_80gb", vramTotal: 85899345920, strategy: "full_gpu", hasJob: true, jobType: "training"},
	{nodeID: "node-1", gpuID: "gpu-1b", deviceIndex: 1, model: "a100_80gb", vramTotal: 85899345920, strategy: "full_gpu", hasJob: true, jobType: "training"},
	{nodeID: "node-1", gpuID: "gpu-1c", deviceIndex: 2, model: "a100_80gb", vramTotal: 85899345920, strategy: "mps", hasJob: true, jobType: "inference"},
	{nodeID: "node-1", gpuID: "gpu-1d", deviceIndex: 3, model: "a100_80gb", vramTotal: 85899345920, strategy: "idle", hasJob: false, jobType: ""},

	// Node 2: 4x A100-40GB — mixed workloads
	{nodeID: "node-2", gpuID: "gpu-2a", deviceIndex: 0, model: "a100_40gb", vramTotal: 42949672960, strategy: "full_gpu", hasJob: true, jobType: "fine_tuning"},
	{nodeID: "node-2", gpuID: "gpu-2b", deviceIndex: 1, model: "a100_40gb", vramTotal: 42949672960, strategy: "mps", hasJob: true, jobType: "inference"},
	{nodeID: "node-2", gpuID: "gpu-2c", deviceIndex: 2, model: "a100_40gb", vramTotal: 42949672960, strategy: "mps", hasJob: true, jobType: "inference"},
	{nodeID: "node-2", gpuID: "gpu-2d", deviceIndex: 3, model: "a100_40gb", vramTotal: 42949672960, strategy: "idle", hasJob: false, jobType: ""},

	// Node 3: 4x H100-80GB — inference serving
	{nodeID: "node-3", gpuID: "gpu-3a", deviceIndex: 0, model: "h100_80gb", vramTotal: 85899345920, strategy: "mps", hasJob: true, jobType: "inference"},
	{nodeID: "node-3", gpuID: "gpu-3b", deviceIndex: 1, model: "h100_80gb", vramTotal: 85899345920, strategy: "mps", hasJob: true, jobType: "inference"},
	{nodeID: "node-3", gpuID: "gpu-3c", deviceIndex: 2, model: "h100_80gb", vramTotal: 85899345920, strategy: "full_gpu", hasJob: true, jobType: "training"},
	{nodeID: "node-3", gpuID: "gpu-3d", deviceIndex: 3, model: "h100_80gb", vramTotal: 85899345920, strategy: "idle", hasJob: false, jobType: ""},
}

// ---------------------------------------------------------------------------
// Metric generation — realistic patterns per workload type
// ---------------------------------------------------------------------------

type gpuState struct {
	utilization float64
	memoryUsed  float64 // fraction 0–1
	powerDraw   float64
	temperature float64
	smOccupancy float64
}

var (
	mu     sync.Mutex
	states = make(map[string]*gpuState)
	start  = time.Now()
)

func init() {
	for _, gpu := range cluster {
		states[gpu.gpuID] = &gpuState{}
	}
}

// updateMetrics evolves GPU states with realistic patterns.
func updateMetrics() {
	mu.Lock()
	defer mu.Unlock()

	elapsed := time.Since(start).Seconds()

	for _, gpu := range cluster {
		s := states[gpu.gpuID]

		if !gpu.hasJob {
			// Idle GPU — near-zero metrics with tiny noise
			s.utilization = 1.0 + rand.Float64()*2.0
			s.memoryUsed = 0.01 + rand.Float64()*0.02
			s.powerDraw = 25.0 + rand.Float64()*10.0
			s.temperature = 32.0 + rand.Float64()*3.0
			s.smOccupancy = 0.5 + rand.Float64()*1.0
			continue
		}

		// Base patterns per workload type
		switch gpu.jobType {
		case "training":
			// Training: high utilization (70-95%), high memory (60-85%), periodic dips (gradient sync)
			cycle := math.Sin(elapsed*0.1+float64(gpu.deviceIndex)) * 10
			s.utilization = 82.0 + cycle + rand.Float64()*5.0
			s.memoryUsed = 0.72 + math.Sin(elapsed*0.05)*0.05 + rand.Float64()*0.03
			s.powerDraw = 280.0 + cycle*3 + rand.Float64()*20.0
			s.temperature = 68.0 + s.utilization*0.15 + rand.Float64()*2.0
			s.smOccupancy = 65.0 + cycle*0.8 + rand.Float64()*5.0

		case "inference":
			// Inference: bursty (20-60%), low memory (15-35%), spiky
			burst := math.Abs(math.Sin(elapsed*0.3+float64(gpu.deviceIndex)*1.5)) * 30
			s.utilization = 25.0 + burst + rand.Float64()*8.0
			s.memoryUsed = 0.20 + math.Sin(elapsed*0.2)*0.05 + rand.Float64()*0.05
			s.powerDraw = 120.0 + burst*2 + rand.Float64()*15.0
			s.temperature = 45.0 + burst*0.3 + rand.Float64()*3.0
			s.smOccupancy = 20.0 + burst*0.6 + rand.Float64()*5.0

		case "fine_tuning":
			// Fine-tuning: moderate and steady (50-75%)
			drift := math.Sin(elapsed*0.08+float64(gpu.deviceIndex)*2) * 8
			s.utilization = 62.0 + drift + rand.Float64()*5.0
			s.memoryUsed = 0.55 + math.Sin(elapsed*0.06)*0.04 + rand.Float64()*0.03
			s.powerDraw = 220.0 + drift*3 + rand.Float64()*15.0
			s.temperature = 58.0 + s.utilization*0.12 + rand.Float64()*2.0
			s.smOccupancy = 48.0 + drift*0.7 + rand.Float64()*4.0
		}

		// Clamp values
		s.utilization = clamp(s.utilization, 0, 100)
		s.memoryUsed = clamp(s.memoryUsed, 0, 0.95)
		s.powerDraw = clamp(s.powerDraw, 20, 400)
		s.temperature = clamp(s.temperature, 25, 90)
		s.smOccupancy = clamp(s.smOccupancy, 0, 100)
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ---------------------------------------------------------------------------
// Extended metrics: interconnect, training, scheduler, cost, SLO
// ---------------------------------------------------------------------------

// histogram is a hand-rolled Prometheus histogram with cumulative bucket counts.
type histogram struct {
	bounds []float64 // upper bounds, ascending (excluding +Inf)
	counts []uint64  // cumulative counts aligned with bounds
	sum    float64
	count  uint64
}

func newHistogram(bounds []float64) *histogram {
	return &histogram{bounds: bounds, counts: make([]uint64, len(bounds))}
}

func (h *histogram) observe(v float64) {
	h.sum += v
	h.count++
	for i, b := range h.bounds {
		if v <= b {
			h.counts[i]++
		}
	}
}

// formatLE renders a bucket bound to Prometheus convention.
func formatLE(b float64) string {
	return strconv.FormatFloat(b, 'f', -1, 64)
}

// trainingJob is a synthetic training job descriptor.
type trainingJob struct {
	jobID     string
	jobName   string
	framework string
	gpuCount  int
	gpuIDs    []string
	// State
	steps          uint64
	loss           float64
	throughput     float64
	gradNorm       float64
	checkpointAge  float64
	checkpointPeak float64 // resets when this is reached
	stepHist       *histogram
}

var trainingJobs = []*trainingJob{
	{jobID: "train-001", jobName: "llama3-pretrain", framework: "pytorch", gpuCount: 2, gpuIDs: []string{"gpu-1a", "gpu-1b"}},
	{jobID: "train-002", jobName: "mixtral-finetune", framework: "pytorch", gpuCount: 1, gpuIDs: []string{"gpu-2a"}},
	{jobID: "train-003", jobName: "vit-pretrain", framework: "jax", gpuCount: 1, gpuIDs: []string{"gpu-3c"}},
}

// Histogram bucket sets.
var (
	stepDurationBuckets    = []float64{0.5, 1, 2, 5, 10, 30, 60, 120, 300}
	schedDecisionBuckets   = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	schedWaitBuckets       = []float64{1, 5, 10, 30, 60, 300, 600, 1800}
	schedQueueNames        = []string{"standard", "priority", "batch"}
	schedDecisionTypes     = []string{"scheduled", "deferred", "rejected"}
	schedPriorities        = []string{"high", "normal", "low"}
	schedPreemptionReasons = []string{"priority", "sla", "drain"}
	sloNames               = []string{"throughput", "scheduler_latency", "gpu_health"}
	sloSeverities          = []string{"minor", "major"}
)

// Interconnect counters (per gpu).
type linkPair struct {
	peerGPU string
	rxBytes uint64
	txBytes uint64
}

var (
	// gpuPeer maps gpuID -> single peer gpuID (for NVLink pairing).
	gpuPeer  = map[string]string{}
	nvlinkRX = map[string]uint64{}
	nvlinkTX = map[string]uint64{}
	pcieRX   = map[string]uint64{}
	pcieTX   = map[string]uint64{}
)

// Scheduler state.
var (
	schedDecisionCounts = map[string]map[string]uint64{}     // decision -> queue -> count
	schedDecisionHist   = map[string]*histogram{}            // queue -> histogram
	schedWaitHist       = map[string]map[string]*histogram{} // queue -> priority -> histogram
	schedQueueDepth     = map[string]map[string]float64{}    // queue -> priority -> depth
	schedPreemptions    = map[string]uint64{}                // reason -> count
)

// Cost state.
var (
	gpuPrice = map[string]float64{
		"a100_80gb": 2.40,
		"a100_40gb": 1.80,
		"h100_80gb": 4.00,
	}
	gpuSpend     = map[string]float64{} // gpuID -> spend USD
	clusterSpend float64
	burnRate     float64
	costBudget   = map[string]float64{
		"hourly":  30,
		"daily":   720,
		"monthly": 21600,
	}
)

// SLO state.
var (
	sloTarget = map[string]float64{
		"throughput":        0.95,
		"scheduler_latency": 0.99,
		"gpu_health":        0.999,
	}
	sloAttainment   = map[string]float64{}
	sloErrorBudget  = map[string]float64{}
	sloViolations   = map[string]map[string]uint64{} // slo -> severity -> count
	sloViolationAcc = 0.0                            // accumulator for occasional increments
)

func initExtendedState() {
	// Pair GPUs within a node: 1a<->1b, 1c<->1d, 2a<->2b, 2c<->2d, 3a<->3b, 3c<->3d.
	pairs := [][2]string{
		{"gpu-1a", "gpu-1b"}, {"gpu-1c", "gpu-1d"},
		{"gpu-2a", "gpu-2b"}, {"gpu-2c", "gpu-2d"},
		{"gpu-3a", "gpu-3b"}, {"gpu-3c", "gpu-3d"},
	}
	for _, p := range pairs {
		gpuPeer[p[0]] = p[1]
		gpuPeer[p[1]] = p[0]
	}
	for _, gpu := range cluster {
		nvlinkRX[gpu.gpuID] = 0
		nvlinkTX[gpu.gpuID] = 0
		pcieRX[gpu.gpuID] = 0
		pcieTX[gpu.gpuID] = 0
		gpuSpend[gpu.gpuID] = 0
	}

	for _, j := range trainingJobs {
		j.loss = 1.9
		j.throughput = 100
		j.gradNorm = 1.5
		j.checkpointAge = 0
		j.checkpointPeak = 110 + rand.Float64()*30
		j.stepHist = newHistogram(stepDurationBuckets)
	}

	for _, d := range schedDecisionTypes {
		schedDecisionCounts[d] = map[string]uint64{}
		for _, q := range schedQueueNames {
			schedDecisionCounts[d][q] = 0
		}
	}
	for _, q := range schedQueueNames {
		schedDecisionHist[q] = newHistogram(schedDecisionBuckets)
		schedWaitHist[q] = map[string]*histogram{}
		schedQueueDepth[q] = map[string]float64{}
		for _, p := range schedPriorities {
			schedWaitHist[q][p] = newHistogram(schedWaitBuckets)
			schedQueueDepth[q][p] = 0
		}
	}
	for _, r := range schedPreemptionReasons {
		schedPreemptions[r] = 0
	}

	for _, slo := range sloNames {
		sloAttainment[slo] = 0.96
		sloErrorBudget[slo] = 0.8
		sloViolations[slo] = map[string]uint64{}
		for _, sev := range sloSeverities {
			sloViolations[slo][sev] = 0
		}
	}
}

// tickCounters increments monotonic counters once per second.
func tickCounters() {
	// Interconnect bytes — only for GPUs with active jobs.
	for _, gpu := range cluster {
		if !gpu.hasJob {
			continue
		}
		// NVLink: training jobs push more peer-to-peer traffic than inference.
		var nvBase float64
		switch gpu.jobType {
		case "training":
			nvBase = 8e8 // ~800 MB/s
		case "fine_tuning":
			nvBase = 4e8
		case "inference":
			nvBase = 5e7 // ~50 MB/s
		}
		nvJitter := nvBase * (0.85 + rand.Float64()*0.3)
		nvlinkRX[gpu.gpuID] += uint64(nvJitter)
		nvlinkTX[gpu.gpuID] += uint64(nvJitter * (0.9 + rand.Float64()*0.2))

		// PCIe: host<->device transfers; inference is moderate, training spiky.
		var pcieBase float64
		switch gpu.jobType {
		case "training":
			pcieBase = 2e8
		case "fine_tuning":
			pcieBase = 1.5e8
		case "inference":
			pcieBase = 3e8 // serving has high host->device traffic
		}
		pcieJitter := pcieBase * (0.8 + rand.Float64()*0.4)
		pcieRX[gpu.gpuID] += uint64(pcieJitter)
		pcieTX[gpu.gpuID] += uint64(pcieJitter * (0.85 + rand.Float64()*0.3))
	}

	// Training step counts.
	for _, j := range trainingJobs {
		// Different frameworks/jobs progress at different rates per second.
		var stepInc uint64
		switch j.jobID {
		case "train-001":
			stepInc = 1 // one step every couple of seconds-ish (we add 1/tick for visibility)
		case "train-002":
			stepInc = 1
		case "train-003":
			stepInc = 2
		}
		j.steps += stepInc
	}

	// Cost: each active GPU accrues price/3600 per second.
	for _, gpu := range cluster {
		if !gpu.hasJob {
			continue
		}
		inc := gpuPrice[gpu.model] / 3600.0
		gpuSpend[gpu.gpuID] += inc
		clusterSpend += inc
	}

	// Scheduler decisions: realistic distribution per tick.
	for _, q := range schedQueueNames {
		// scheduled most common, deferred occasional, rejected rare.
		r := rand.Float64()
		var decision string
		switch {
		case r < 0.75:
			decision = "scheduled"
		case r < 0.93:
			decision = "deferred"
		default:
			decision = "rejected"
		}
		schedDecisionCounts[decision][q]++
	}

	// Preemptions: rare, ~1 every 10s on average across reasons.
	if rand.Float64() < 0.15 {
		reason := schedPreemptionReasons[rand.Intn(len(schedPreemptionReasons))]
		schedPreemptions[reason]++
	}

	// SLO violations: rare; minor more common than major.
	sloViolationAcc += rand.Float64()
	if sloViolationAcc > 7 {
		sloViolationAcc = 0
		slo := sloNames[rand.Intn(len(sloNames))]
		sev := "minor"
		if rand.Float64() < 0.25 {
			sev = "major"
		}
		sloViolations[slo][sev]++
	}
}

// tickHistograms records observations once per second.
func tickHistograms() {
	// Scheduler decision latency: 50-200ms, rare 1.5s spike.
	for _, q := range schedDecisionHist {
		var v float64
		if rand.Float64() < 0.02 {
			v = 1.2 + rand.Float64()*0.8 // spike 1.2s-2.0s
		} else {
			v = 0.05 + rand.Float64()*0.15
		}
		q.observe(v)
	}

	// Scheduler wait time per queue+priority.
	for _, q := range schedQueueNames {
		for _, p := range schedPriorities {
			var v float64
			switch p {
			case "high":
				v = 1 + rand.Float64()*14
			case "normal":
				v = 10 + rand.Float64()*60
			case "low":
				v = 60 + rand.Float64()*840
			}
			schedWaitHist[q][p].observe(v)
		}
	}

	// Training step duration per job.
	for _, j := range trainingJobs {
		var v float64
		switch j.jobID {
		case "train-001": // llama3 pytorch
			v = 2 + rand.Float64()*2
		case "train-002": // mixtral pytorch
			v = 1 + rand.Float64()*1
		case "train-003": // vit jax
			v = 0.6 + rand.Float64()*0.6
		}
		j.stepHist.observe(v)
	}
}

// tickGauges refreshes gauge values.
func tickGauges() {
	elapsed := time.Since(start).Seconds()

	// Training gauges.
	for i, j := range trainingJobs {
		offset := float64(i) * 0.1
		j.loss = 1.5*math.Exp(-elapsed/600.0) + 0.4 + offset + (rand.Float64()-0.5)*0.05

		switch j.jobID {
		case "train-001":
			j.throughput = 110 + math.Sin(elapsed*0.05)*15 + rand.Float64()*10
		case "train-002":
			j.throughput = 280 + math.Sin(elapsed*0.04)*40 + rand.Float64()*20
		case "train-003":
			j.throughput = 580 + math.Sin(elapsed*0.06)*100 + rand.Float64()*50
		}
		j.gradNorm = clamp(1.5+math.Sin(elapsed*0.07+offset)*1.2+rand.Float64()*0.4, 0.5, 4.0)

		j.checkpointAge += 1.0
		if j.checkpointAge >= j.checkpointPeak {
			j.checkpointAge = 0
			j.checkpointPeak = 110 + rand.Float64()*30
		}
	}

	// Scheduler queue depth.
	for _, q := range schedQueueNames {
		for _, p := range schedPriorities {
			var mean float64
			switch p {
			case "high":
				mean = 8
			case "normal":
				mean = 5
			case "low":
				mean = 2
			}
			d := mean + math.Sin(elapsed*0.05+float64(len(q)))*3 + (rand.Float64()-0.5)*2
			schedQueueDepth[q][p] = clamp(d, 0, 15)
		}
	}

	// SLO attainment with slow drift in [0.93, 0.99].
	for i, slo := range sloNames {
		base := 0.96 + math.Sin(elapsed*0.01+float64(i))*0.025
		sloAttainment[slo] = clamp(base+(rand.Float64()-0.5)*0.005, 0.93, 0.99)
		// Error budget remaining: model with sinusoidal in [0.4, 1.0].
		sloErrorBudget[slo] = clamp(0.7+math.Sin(elapsed*0.008+float64(i)*0.7)*0.3, 0.4, 1.0)
	}

	// Burn rate = sum(price[model]) over active GPUs (USD/hour).
	br := 0.0
	for _, gpu := range cluster {
		if gpu.hasJob {
			br += gpuPrice[gpu.model]
		}
	}
	burnRate = br
}

// ---------------------------------------------------------------------------
// Prometheus /metrics endpoint
// ---------------------------------------------------------------------------

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	updateMetrics()

	mu.Lock()
	defer mu.Unlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// GPU-level metrics
	fmt.Fprintln(w, "# HELP veltrix_gpu_utilization_percent Current GPU compute utilization percentage.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_utilization_percent gauge")
	for _, gpu := range cluster {
		s := states[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_utilization_percent{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q,strategy=%q,workload_type=%q} %.2f\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, gpu.strategy, gpu.jobType, s.utilization)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_memory_used_bytes Current GPU memory usage in bytes.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_memory_used_bytes gauge")
	for _, gpu := range cluster {
		s := states[gpu.gpuID]
		used := int64(s.memoryUsed * float64(gpu.vramTotal))
		fmt.Fprintf(w, "veltrix_gpu_memory_used_bytes{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q} %d\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, used)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_memory_total_bytes Total GPU memory in bytes.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_memory_total_bytes gauge")
	for _, gpu := range cluster {
		fmt.Fprintf(w, "veltrix_gpu_memory_total_bytes{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q} %d\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, gpu.vramTotal)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_power_draw_watts Current GPU power consumption in watts.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_power_draw_watts gauge")
	for _, gpu := range cluster {
		s := states[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_power_draw_watts{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q} %.1f\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, s.powerDraw)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_temperature_celsius Current GPU temperature in Celsius.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_temperature_celsius gauge")
	for _, gpu := range cluster {
		s := states[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_temperature_celsius{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q} %.1f\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, s.temperature)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_sm_occupancy_percent Current GPU SM occupancy percentage.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_sm_occupancy_percent gauge")
	for _, gpu := range cluster {
		s := states[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_sm_occupancy_percent{node_id=%q,gpu_id=%q,device_index=%q,gpu_model=%q} %.2f\n",
			gpu.nodeID, gpu.gpuID, fmt.Sprintf("%d", gpu.deviceIndex), gpu.model, s.smOccupancy)
	}

	// Cluster-level metrics
	fmt.Fprintln(w, "# HELP veltrix_cluster_gpus_total Total number of GPUs in the cluster.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_gpus_total gauge")
	fmt.Fprintf(w, "veltrix_cluster_gpus_total %d\n", len(cluster))

	allocated := 0
	for _, gpu := range cluster {
		if gpu.hasJob {
			allocated++
		}
	}
	fmt.Fprintln(w, "# HELP veltrix_cluster_gpus_allocated Number of GPUs with active workloads.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_gpus_allocated gauge")
	fmt.Fprintf(w, "veltrix_cluster_gpus_allocated %d\n", allocated)

	fmt.Fprintln(w, "# HELP veltrix_cluster_gpus_available Number of idle GPUs.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_gpus_available gauge")
	fmt.Fprintf(w, "veltrix_cluster_gpus_available %d\n", len(cluster)-allocated)

	fmt.Fprintln(w, "# HELP veltrix_jobs_running_total Number of currently running jobs.")
	fmt.Fprintln(w, "# TYPE veltrix_jobs_running_total gauge")
	fmt.Fprintf(w, "veltrix_jobs_running_total %d\n", allocated)

	fmt.Fprintln(w, "# HELP veltrix_jobs_queued_total Number of jobs waiting in the scheduler queue.")
	fmt.Fprintln(w, "# TYPE veltrix_jobs_queued_total gauge")
	fmt.Fprintf(w, "veltrix_jobs_queued_total %d\n", 3+rand.Intn(5))

	// Per-node aggregates
	fmt.Fprintln(w, "# HELP veltrix_node_gpus_total Total GPUs per node.")
	fmt.Fprintln(w, "# TYPE veltrix_node_gpus_total gauge")
	for _, nodeID := range []string{"node-1", "node-2", "node-3"} {
		fmt.Fprintf(w, "veltrix_node_gpus_total{node_id=%q} 4\n", nodeID)
	}

	// Strategy distribution
	strategies := map[string]int{}
	for _, gpu := range cluster {
		strategies[gpu.strategy]++
	}
	fmt.Fprintln(w, "# HELP veltrix_cluster_strategy_distribution GPUs by allocation strategy.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_strategy_distribution gauge")
	for strategy, count := range strategies {
		fmt.Fprintf(w, "veltrix_cluster_strategy_distribution{strategy=%q} %d\n", strategy, count)
	}

	// Workload type distribution
	workloads := map[string]int{}
	for _, gpu := range cluster {
		if gpu.jobType != "" {
			workloads[gpu.jobType]++
		}
	}
	fmt.Fprintln(w, "# HELP veltrix_cluster_workload_distribution GPUs by workload type.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_workload_distribution gauge")
	for wtype, count := range workloads {
		fmt.Fprintf(w, "veltrix_cluster_workload_distribution{workload_type=%q} %d\n", wtype, count)
	}

	// -----------------------------------------------------------------------
	// Interconnect: NVLink + PCIe byte counters
	// -----------------------------------------------------------------------
	fmt.Fprintln(w, "# HELP veltrix_gpu_nvlink_rx_bytes_total Total NVLink bytes received from a peer GPU.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_nvlink_rx_bytes_total counter")
	for _, gpu := range cluster {
		peer := gpuPeer[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_nvlink_rx_bytes_total{node_id=%q,gpu_id=%q,peer_gpu_id=%q} %d\n",
			gpu.nodeID, gpu.gpuID, peer, nvlinkRX[gpu.gpuID])
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_nvlink_tx_bytes_total Total NVLink bytes transmitted to a peer GPU.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_nvlink_tx_bytes_total counter")
	for _, gpu := range cluster {
		peer := gpuPeer[gpu.gpuID]
		fmt.Fprintf(w, "veltrix_gpu_nvlink_tx_bytes_total{node_id=%q,gpu_id=%q,peer_gpu_id=%q} %d\n",
			gpu.nodeID, gpu.gpuID, peer, nvlinkTX[gpu.gpuID])
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_pcie_rx_bytes_total Total PCIe bytes received host->device.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_pcie_rx_bytes_total counter")
	for _, gpu := range cluster {
		fmt.Fprintf(w, "veltrix_gpu_pcie_rx_bytes_total{node_id=%q,gpu_id=%q} %d\n",
			gpu.nodeID, gpu.gpuID, pcieRX[gpu.gpuID])
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_pcie_tx_bytes_total Total PCIe bytes transmitted device->host.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_pcie_tx_bytes_total counter")
	for _, gpu := range cluster {
		fmt.Fprintf(w, "veltrix_gpu_pcie_tx_bytes_total{node_id=%q,gpu_id=%q} %d\n",
			gpu.nodeID, gpu.gpuID, pcieTX[gpu.gpuID])
	}

	// -----------------------------------------------------------------------
	// Training metrics
	// -----------------------------------------------------------------------
	fmt.Fprintln(w, "# HELP veltrix_training_job_steps_total Number of completed training steps for a job.")
	fmt.Fprintln(w, "# TYPE veltrix_training_job_steps_total counter")
	for _, j := range trainingJobs {
		fmt.Fprintf(w, "veltrix_training_job_steps_total{job_id=%q,job_name=%q,framework=%q,gpu_count=%q} %d\n",
			j.jobID, j.jobName, j.framework, strconv.Itoa(j.gpuCount), j.steps)
	}

	fmt.Fprintln(w, "# HELP veltrix_training_step_duration_seconds Distribution of training step durations.")
	fmt.Fprintln(w, "# TYPE veltrix_training_step_duration_seconds histogram")
	for _, j := range trainingJobs {
		h := j.stepHist
		for i, b := range h.bounds {
			fmt.Fprintf(w, "veltrix_training_step_duration_seconds_bucket{job_id=%q,framework=%q,le=%q} %d\n",
				j.jobID, j.framework, formatLE(b), h.counts[i])
		}
		fmt.Fprintf(w, "veltrix_training_step_duration_seconds_bucket{job_id=%q,framework=%q,le=\"+Inf\"} %d\n",
			j.jobID, j.framework, h.count)
		fmt.Fprintf(w, "veltrix_training_step_duration_seconds_sum{job_id=%q,framework=%q} %g\n",
			j.jobID, j.framework, h.sum)
		fmt.Fprintf(w, "veltrix_training_step_duration_seconds_count{job_id=%q,framework=%q} %d\n",
			j.jobID, j.framework, h.count)
	}

	fmt.Fprintln(w, "# HELP veltrix_training_loss Current training loss value.")
	fmt.Fprintln(w, "# TYPE veltrix_training_loss gauge")
	for _, j := range trainingJobs {
		fmt.Fprintf(w, "veltrix_training_loss{job_id=%q,framework=%q} %.4f\n", j.jobID, j.framework, j.loss)
	}

	fmt.Fprintln(w, "# HELP veltrix_training_throughput_samples_per_second Current training throughput in samples/sec.")
	fmt.Fprintln(w, "# TYPE veltrix_training_throughput_samples_per_second gauge")
	for _, j := range trainingJobs {
		fmt.Fprintf(w, "veltrix_training_throughput_samples_per_second{job_id=%q,framework=%q} %.2f\n",
			j.jobID, j.framework, j.throughput)
	}

	fmt.Fprintln(w, "# HELP veltrix_training_gradient_norm Current gradient L2 norm.")
	fmt.Fprintln(w, "# TYPE veltrix_training_gradient_norm gauge")
	for _, j := range trainingJobs {
		fmt.Fprintf(w, "veltrix_training_gradient_norm{job_id=%q,framework=%q} %.4f\n",
			j.jobID, j.framework, j.gradNorm)
	}

	fmt.Fprintln(w, "# HELP veltrix_training_checkpoint_age_seconds Seconds since last successful checkpoint.")
	fmt.Fprintln(w, "# TYPE veltrix_training_checkpoint_age_seconds gauge")
	for _, j := range trainingJobs {
		fmt.Fprintf(w, "veltrix_training_checkpoint_age_seconds{job_id=%q} %.1f\n", j.jobID, j.checkpointAge)
	}

	// -----------------------------------------------------------------------
	// Scheduler metrics
	// -----------------------------------------------------------------------
	fmt.Fprintln(w, "# HELP veltrix_scheduler_decisions_total Scheduler decisions partitioned by outcome and queue.")
	fmt.Fprintln(w, "# TYPE veltrix_scheduler_decisions_total counter")
	for _, decision := range schedDecisionTypes {
		for _, q := range schedQueueNames {
			fmt.Fprintf(w, "veltrix_scheduler_decisions_total{decision=%q,queue=%q} %d\n",
				decision, q, schedDecisionCounts[decision][q])
		}
	}

	fmt.Fprintln(w, "# HELP veltrix_scheduler_decision_seconds Time taken to make a scheduling decision.")
	fmt.Fprintln(w, "# TYPE veltrix_scheduler_decision_seconds histogram")
	for _, q := range schedQueueNames {
		h := schedDecisionHist[q]
		for i, b := range h.bounds {
			fmt.Fprintf(w, "veltrix_scheduler_decision_seconds_bucket{queue=%q,le=%q} %d\n",
				q, formatLE(b), h.counts[i])
		}
		fmt.Fprintf(w, "veltrix_scheduler_decision_seconds_bucket{queue=%q,le=\"+Inf\"} %d\n", q, h.count)
		fmt.Fprintf(w, "veltrix_scheduler_decision_seconds_sum{queue=%q} %g\n", q, h.sum)
		fmt.Fprintf(w, "veltrix_scheduler_decision_seconds_count{queue=%q} %d\n", q, h.count)
	}

	fmt.Fprintln(w, "# HELP veltrix_scheduler_wait_seconds Time a job spends waiting in the scheduler queue.")
	fmt.Fprintln(w, "# TYPE veltrix_scheduler_wait_seconds histogram")
	for _, q := range schedQueueNames {
		for _, p := range schedPriorities {
			h := schedWaitHist[q][p]
			for i, b := range h.bounds {
				fmt.Fprintf(w, "veltrix_scheduler_wait_seconds_bucket{queue=%q,priority=%q,le=%q} %d\n",
					q, p, formatLE(b), h.counts[i])
			}
			fmt.Fprintf(w, "veltrix_scheduler_wait_seconds_bucket{queue=%q,priority=%q,le=\"+Inf\"} %d\n",
				q, p, h.count)
			fmt.Fprintf(w, "veltrix_scheduler_wait_seconds_sum{queue=%q,priority=%q} %g\n", q, p, h.sum)
			fmt.Fprintf(w, "veltrix_scheduler_wait_seconds_count{queue=%q,priority=%q} %d\n", q, p, h.count)
		}
	}

	fmt.Fprintln(w, "# HELP veltrix_scheduler_queue_depth Current number of jobs in a scheduler queue.")
	fmt.Fprintln(w, "# TYPE veltrix_scheduler_queue_depth gauge")
	for _, q := range schedQueueNames {
		for _, p := range schedPriorities {
			fmt.Fprintf(w, "veltrix_scheduler_queue_depth{queue=%q,priority=%q} %.2f\n",
				q, p, schedQueueDepth[q][p])
		}
	}

	fmt.Fprintln(w, "# HELP veltrix_scheduler_preemptions_total Number of preemptions partitioned by reason.")
	fmt.Fprintln(w, "# TYPE veltrix_scheduler_preemptions_total counter")
	for _, reason := range schedPreemptionReasons {
		fmt.Fprintf(w, "veltrix_scheduler_preemptions_total{reason=%q} %d\n", reason, schedPreemptions[reason])
	}

	// -----------------------------------------------------------------------
	// Cost metrics
	// -----------------------------------------------------------------------
	fmt.Fprintln(w, "# HELP veltrix_gpu_hourly_price_usd Hourly USD price for a GPU model.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_hourly_price_usd gauge")
	for model, price := range gpuPrice {
		fmt.Fprintf(w, "veltrix_gpu_hourly_price_usd{gpu_model=%q} %.4f\n", model, price)
	}

	fmt.Fprintln(w, "# HELP veltrix_gpu_spend_usd_total Cumulative USD spend per GPU.")
	fmt.Fprintln(w, "# TYPE veltrix_gpu_spend_usd_total counter")
	for _, gpu := range cluster {
		fmt.Fprintf(w, "veltrix_gpu_spend_usd_total{node_id=%q,gpu_id=%q,gpu_model=%q,workload_type=%q} %.6f\n",
			gpu.nodeID, gpu.gpuID, gpu.model, gpu.jobType, gpuSpend[gpu.gpuID])
	}

	fmt.Fprintln(w, "# HELP veltrix_cluster_spend_usd_total Cumulative USD spend across the cluster.")
	fmt.Fprintln(w, "# TYPE veltrix_cluster_spend_usd_total counter")
	fmt.Fprintf(w, "veltrix_cluster_spend_usd_total %.6f\n", clusterSpend)

	fmt.Fprintln(w, "# HELP veltrix_cost_budget_target_usd Target spend budget per window.")
	fmt.Fprintln(w, "# TYPE veltrix_cost_budget_target_usd gauge")
	for _, window := range []string{"hourly", "daily", "monthly"} {
		fmt.Fprintf(w, "veltrix_cost_budget_target_usd{window=%q} %.2f\n", window, costBudget[window])
	}

	fmt.Fprintln(w, "# HELP veltrix_cost_burn_rate_usd_per_hour Current USD/hour spend rate.")
	fmt.Fprintln(w, "# TYPE veltrix_cost_burn_rate_usd_per_hour gauge")
	fmt.Fprintf(w, "veltrix_cost_burn_rate_usd_per_hour %.4f\n", burnRate)

	// -----------------------------------------------------------------------
	// SLO metrics
	// -----------------------------------------------------------------------
	fmt.Fprintln(w, "# HELP veltrix_slo_target_ratio Target SLO attainment ratio.")
	fmt.Fprintln(w, "# TYPE veltrix_slo_target_ratio gauge")
	for _, slo := range sloNames {
		fmt.Fprintf(w, "veltrix_slo_target_ratio{slo=%q} %.4f\n", slo, sloTarget[slo])
	}

	fmt.Fprintln(w, "# HELP veltrix_slo_attainment_ratio Current SLO attainment ratio.")
	fmt.Fprintln(w, "# TYPE veltrix_slo_attainment_ratio gauge")
	for _, slo := range sloNames {
		fmt.Fprintf(w, "veltrix_slo_attainment_ratio{slo=%q} %.4f\n", slo, sloAttainment[slo])
	}

	fmt.Fprintln(w, "# HELP veltrix_slo_error_budget_remaining_ratio Remaining error budget for an SLO in [0,1].")
	fmt.Fprintln(w, "# TYPE veltrix_slo_error_budget_remaining_ratio gauge")
	for _, slo := range sloNames {
		fmt.Fprintf(w, "veltrix_slo_error_budget_remaining_ratio{slo=%q} %.4f\n", slo, sloErrorBudget[slo])
	}

	fmt.Fprintln(w, "# HELP veltrix_slo_violations_total Number of SLO violations partitioned by SLO and severity.")
	fmt.Fprintln(w, "# TYPE veltrix_slo_violations_total counter")
	for _, slo := range sloNames {
		for _, sev := range sloSeverities {
			fmt.Fprintf(w, "veltrix_slo_violations_total{slo=%q,severity=%q} %d\n",
				slo, sev, sloViolations[slo][sev])
		}
	}
}

func main() {
	log.Println("Veltrix GPU simulator starting on :9100")
	log.Println("Simulating 3 nodes, 12 GPUs (4x A100-80GB, 4x A100-40GB, 4x H100-80GB)")

	initExtendedState()

	go func() {
		t := time.NewTicker(1 * time.Second)
		defer t.Stop()
		for range t.C {
			mu.Lock()
			tickCounters()
			tickHistograms()
			tickGauges()
			mu.Unlock()
		}
	}()

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Veltrix GPU Simulator. Metrics at /metrics\n"))
	})

	log.Fatal(http.ListenAndServe(":9100", nil))
}

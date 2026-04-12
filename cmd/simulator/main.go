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
}

func main() {
	log.Println("Veltrix GPU simulator starting on :9100")
	log.Println("Simulating 3 nodes, 12 GPUs (4x A100-80GB, 4x A100-40GB, 4x H100-80GB)")

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Veltrix GPU Simulator. Metrics at /metrics\n"))
	})

	log.Fatal(http.ListenAndServe(":9100", nil))
}

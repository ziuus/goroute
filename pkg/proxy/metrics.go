package proxy

import (
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/ziuus/goroute/pkg/fast"
	"github.com/ziuus/goroute/pkg/quota"
)

var startTime = time.Now()

// handleMetrics exports standard Prometheus text format metrics for scrape targets.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptimeSeconds := time.Since(startTime).Seconds()
	summary := quota.GlobalLedger.Summary()

	rustAvailable := 0
	if fast.IsAvailable() {
		rustAvailable = 1
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")

	// Server info & uptime
	fmt.Fprintf(w, "# HELP goroute_uptime_seconds Total runtime of the GoRoute daemon in seconds.\n")
	fmt.Fprintf(w, "# TYPE goroute_uptime_seconds counter\n")
	fmt.Fprintf(w, "goroute_uptime_seconds %.2f\n\n", uptimeSeconds)

	// Memory & System metrics
	fmt.Fprintf(w, "# HELP goroute_memory_alloc_bytes Bytes allocated and still in use by Go runtime.\n")
	fmt.Fprintf(w, "# TYPE goroute_memory_alloc_bytes gauge\n")
	fmt.Fprintf(w, "goroute_memory_alloc_bytes %d\n\n", m.Alloc)

	fmt.Fprintf(w, "# HELP goroute_memory_sys_bytes Bytes obtained from system.\n")
	fmt.Fprintf(w, "# TYPE goroute_memory_sys_bytes gauge\n")
	fmt.Fprintf(w, "goroute_memory_sys_bytes %d\n\n", m.Sys)

	fmt.Fprintf(w, "# HELP goroute_goroutines Number of active goroutines.\n")
	fmt.Fprintf(w, "# TYPE goroute_goroutines gauge\n")
	fmt.Fprintf(w, "goroute_goroutines %d\n\n", runtime.NumGoroutine())

	// Accelerator status
	fmt.Fprintf(w, "# HELP goroute_rust_simd_active 1 if Rust SIMD C-ABI accelerator is active, 0 otherwise.\n")
	fmt.Fprintf(w, "# TYPE goroute_rust_simd_active gauge\n")
	fmt.Fprintf(w, "goroute_rust_simd_active %d\n\n", rustAvailable)

	// Requests & Traffic Ledger metrics
	fmt.Fprintf(w, "# HELP goroute_requests_total Total number of LLM inference requests received.\n")
	fmt.Fprintf(w, "# TYPE goroute_requests_total counter\n")
	fmt.Fprintf(w, "goroute_requests_total %d\n\n", summary.TotalRequests)

	fmt.Fprintf(w, "# HELP goroute_tokens_total Total tokens processed.\n")
	fmt.Fprintf(w, "# TYPE goroute_tokens_total counter\n")
	fmt.Fprintf(w, "goroute_tokens_total %d\n\n", summary.TotalTokens)

	fmt.Fprintf(w, "# HELP goroute_cost_usd_total Estimated upstream cost in USD.\n")
	fmt.Fprintf(w, "# TYPE goroute_cost_usd_total counter\n")
	fmt.Fprintf(w, "goroute_cost_usd_total %.4f\n\n", summary.TotalCostUSD)

	fmt.Fprintf(w, "# HELP goroute_saved_usd_total Estimated cost saved via routing/free models in USD.\n")
	fmt.Fprintf(w, "# TYPE goroute_saved_usd_total counter\n")
	fmt.Fprintf(w, "goroute_saved_usd_total %.4f\n\n", summary.TotalSavedUSD)
}

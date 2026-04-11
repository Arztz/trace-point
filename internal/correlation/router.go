package correlation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/trace-point/trace-point/internal/integration/signoz"
	"github.com/trace-point/trace-point/internal/utils/logger"
)

// Router handles route selection when multiple routes are active during a spike
type Router struct {
	logger      *logger.Logger
	correlation *signoz.CorrelationService
	jobPatterns []string
}

// NewRouter creates a new router with route selection logic
func NewRouter(logger *logger.Logger, correlation *signoz.CorrelationService) *Router {
	return &Router{
		logger:      logger,
		correlation: correlation,
		jobPatterns: []string{"/tasks/", "/batch/", "/jobs/"},
	}
}

// SelectRoute selects the culprit route when multiple routes are active during a spike
// Criteria: highest CPU + RAM consumption weighted by route activity
func (r *Router) SelectRoute(
	ctx context.Context,
	namespace, podName string,
	spikeTime time.Time,
	cpuPercent, ramPercent float64,
) (*RouteSelection, error) {

	result, err := r.correlation.CorrelateSpikeWithTraces(
		ctx, namespace, podName, spikeTime, cpuPercent, ramPercent, 5,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to correlate spike: %w", err)
	}

	if !result.FoundTraces {
		return &RouteSelection{
			FoundRoutes: false,
		}, nil
	}

	// Select culprit route
	var culprit *signoz.RouteActivity
	if len(result.RouteActivities) > 0 {
		culprit = &result.RouteActivities[0]
	}

	selection := &RouteSelection{
		FoundRoutes:      true,
		ActiveRoutes:     result.RouteActivities,
		TraceCount:       result.TraceCount,
		TraceID:          result.TraceID,
		CombinedResource: cpuPercent + ramPercent,
	}

	if culprit != nil {
		selection.CulpritRoute = culprit.Route
		selection.CulpritResourceWeight = culprit.ResourceWeight

		// Tag as suspected job if matches patterns
		if r.isJobRoute(culprit.Route) {
			selection.CulpritRoute = "[Suspected Job] " + culprit.Route
			selection.IsJobRoute = true
		}
	}

	r.logger.Info("Route selection for %s/%s: culprit=%s, routes=%d, CPU=%.2f%%, RAM=%.2f%%",
		namespace, podName, selection.CulpritRoute, len(result.RouteActivities), cpuPercent, ramPercent)

	return selection, nil
}

// isJobRoute checks if a route matches job patterns
func (r *Router) isJobRoute(route string) bool {
	for _, pattern := range r.jobPatterns {
		if strings.HasPrefix(route, pattern) {
			return true
		}
	}
	return false
}

// RouteSelection holds the result of route selection
type RouteSelection struct {
	FoundRoutes           bool                   `json:"found_routes"`
	ActiveRoutes          []signoz.RouteActivity `json:"active_routes"`
	CulpritRoute          string                 `json:"culprit_route"`
	CulpritResourceWeight float64                `json:"culprit_resource_weight"`
	IsJobRoute            bool                   `json:"is_job_route"`
	TraceCount            int                    `json:"trace_count"`
	TraceID               string                 `json:"trace_id"`
	CombinedResource      float64                `json:"combined_resource"`
}

// FilterSpikeRoutes filters and tags routes during a spike event
func (r *Router) FilterSpikeRoutes(routes []signoz.RouteActivity) []signoz.RouteActivity {
	filtered := make([]signoz.RouteActivity, 0)

	for _, route := range routes {
		// Skip routes with no meaningful activity
		if route.TraceCount == 0 {
			continue
		}

		// Tag job routes
		taggedRoute := r.tagJobRoute(route.Route)
		route.Route = taggedRoute

		filtered = append(filtered, route)
	}

	return filtered
}

// tagJobRoute adds [Suspected Job] prefix if matching patterns
func (r *Router) tagJobRoute(route string) string {
	for _, pattern := range r.jobPatterns {
		if strings.HasPrefix(route, pattern) {
			return "[Suspected Job] " + route
		}
	}
	return route
}

// GetTopRoutes returns the top N routes by resource weight
func (r *Router) GetTopRoutes(routes []signoz.RouteActivity, count int) []signoz.RouteActivity {
	if count <= 0 {
		count = 5
	}

	if len(routes) <= count {
		return routes
	}

	return routes[:count]
}

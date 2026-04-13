package prometheus

import (
	"fmt"
	"strings"
)

// Query builders for Prometheus metrics

// AggregationLevel defines the level of aggregation for Prometheus queries
type AggregationLevel string

const (
	// AggregationByPod groups metrics by individual pod (default, backward compatible)
	AggregationByPod AggregationLevel = "pod"
	// AggregationByDeployment groups metrics by deployment (aggregated using label_replace)
	AggregationByDeployment AggregationLevel = "deployment"
)

// BuildCPUUtilizationQuery builds a query for CPU utilization percentage
// Formula: container_cpu_usage_seconds_total / kube_pod_container_resource_requests (cpu)
// Returns: percentage of CPU used vs CPU request
// Uses AggregationByPod by default. Pass AggregationByDeployment for deployment-level aggregation.
func BuildCPUUtilizationQuery(namespaces []string, excludePatterns []string) string {
	return BuildCPUUtilizationQueryWithAggregation(namespaces, excludePatterns, AggregationByPod)
}

// BuildCPUUtilizationQueryWithAggregation builds a query with specified aggregation level.
// level: AggregationByPod (default) or AggregationByDeployment
// Using deployment-level saves significant performance by reducing cardinality.
func BuildCPUUtilizationQueryWithAggregation(namespaces []string, excludePatterns []string, level AggregationLevel) string {
	// Build selector conditions for metrics
	var selectorConditions []string

	// Always exclude empty containers
	selectorConditions = append(selectorConditions, `container!=""`)

	// Filter by namespace if specified - put inside metric selector
	if len(namespaces) > 0 {
		nsConditions := make([]string, len(namespaces))
		for i, ns := range namespaces {
			nsConditions[i] = fmt.Sprintf(`namespace="%s"`, ns)
		}
		selectorConditions = append(selectorConditions, fmt.Sprintf("%s", strings.Join(nsConditions, "|")))
	}

	// Exclude patterns if specified - put inside metric selector
	for _, pattern := range excludePatterns {
		selectorConditions = append(selectorConditions, fmt.Sprintf(`pod!~"%s"`, pattern))
	}

	selector := strings.Join(selectorConditions, ", ")

	// Base query: CPU usage / CPU request as percentage
	// Using kube_pod_container_resource_requests for CPU request instead of container_spec_cpu_quota
	// Fix: Use subquery to ensure proper 1:1 matching between usage and request
	// This prevents issues when Prometheus returns multiple container entries per pod

	if level == AggregationByDeployment {
		// Deployment-level aggregation: use label_replace to extract deployment name,
		// then sum by (deployment, namespace, container) to aggregate all pods in a deployment
		// Pattern (.*)-[a-z0-9]+-[a-z0-9]+ extracts "deployment" from "deployment-hash-uid"
		query := fmt.Sprintf(
			`(sum by (deployment, namespace, container) (
				label_replace(
					rate(container_cpu_usage_seconds_total{%s}[5m]),
					"deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
				)
			)) / 
			(sum by (deployment, namespace, container) (
				label_replace(
					kube_pod_container_resource_requests{%s, resource="cpu"},
					"deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
				)
			)) * 100`,
			selector, selector,
		)
		return query
	}

	// Pod-level aggregation (default, backward compatible)
	query := fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="cpu"})) * 100`,
		selector, selector,
	)

	return query
}

// BuildRAMUtilizationQuery builds a query for RAM utilization percentage
// Formula: container_memory_working_set_bytes / kube_pod_container_resource_requests (memory)
// Returns: percentage of RAM used vs RAM request
// Uses AggregationByPod by default. Pass AggregationByDeployment for deployment-level aggregation.
func BuildRAMUtilizationQuery(namespaces []string, excludePatterns []string) string {
	return BuildRAMUtilizationQueryWithAggregation(namespaces, excludePatterns, AggregationByPod)
}

// BuildRAMUtilizationQueryWithAggregation builds a query with specified aggregation level.
// level: AggregationByPod (default) or AggregationByDeployment
// Using deployment-level saves significant performance by reducing cardinality.
func BuildRAMUtilizationQueryWithAggregation(namespaces []string, excludePatterns []string, level AggregationLevel) string {
	// Build selector conditions for metrics
	var selectorConditions []string

	// Always exclude empty containers
	selectorConditions = append(selectorConditions, `container!=""`)

	// Filter by namespace if specified - put inside metric selector
	if len(namespaces) > 0 {
		nsConditions := make([]string, len(namespaces))
		for i, ns := range namespaces {
			nsConditions[i] = fmt.Sprintf(`namespace="%s"`, ns)
		}
		selectorConditions = append(selectorConditions, fmt.Sprintf("%s", strings.Join(nsConditions, "|")))
	}

	// Exclude patterns if specified - put inside metric selector
	for _, pattern := range excludePatterns {
		selectorConditions = append(selectorConditions, fmt.Sprintf(`pod!~"%s"`, pattern))
	}

	selector := strings.Join(selectorConditions, ", ")

	// Base query: Working set / Memory request as percentage
	// Using kube_pod_container_resource_requests for memory request instead of container_spec_memory_limit
	// Note: kube_pod_container_resource_requests is a gauge metric (requested amount at point in time),
	// NOT a counter, so rate() should NOT be applied to it (unlike container_memory_working_set_bytes which is a gauge but works with rate())
	// Fix: Use subquery to ensure proper 1:1 matching between usage and request

	if level == AggregationByDeployment {
		// Deployment-level aggregation: use label_replace to extract deployment name,
		// then sum by (deployment, namespace, container) to aggregate all pods in a deployment
		// Pattern (.*)-[a-z0-9]+-[a-z0-9]+ extracts "deployment" from "deployment-hash-uid"
		query := fmt.Sprintf(
			`(sum by (deployment, namespace, container) (
				label_replace(
					rate(container_memory_working_set_bytes{%s}[5m]),
					"deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
				)
			)) / 
			(sum by (deployment, namespace, container) (
				label_replace(
					kube_pod_container_resource_requests{%s, resource="memory"},
					"deployment", "$1", "pod", "(.*)-[a-z0-9]+-[a-z0-9]+"
				)
			)) * 100`,
			selector, selector,
		)
		return query
	}

	// Pod-level aggregation (default, backward compatible)
	query := fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_memory_working_set_bytes{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="memory"})) * 100`,
		selector, selector,
	)

	return query
}

// BuildContainerCPUQuery builds a query for a specific container's CPU usage
func BuildContainerCPUQuery(podName, namespace, containerName string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s", container="%s"`, podName, namespace, containerName)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="cpu"})) * 100`,
		selector, selector,
	)
}

// BuildContainerRAMQuery builds a query for a specific container's RAM usage
func BuildContainerRAMQuery(podName, namespace, containerName string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s", container="%s"`, podName, namespace, containerName)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_memory_working_set_bytes{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="memory"})) * 100`,
		selector, selector,
	)
}

// BuildPodCPUQuery builds a query for all containers in a pod
func BuildPodCPUQuery(podName, namespace string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s"`, podName, namespace)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="cpu"})) * 100`,
		selector, selector,
	)
}

// BuildPodRAMQuery builds a query for all containers in a pod
func BuildPodRAMQuery(podName, namespace string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s"`, podName, namespace)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (container_memory_working_set_bytes{%s}) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="memory"})) * 100`,
		selector, selector,
	)
}

// BuildNamespaceCPUQuery builds a query for all pods in a namespace
func BuildNamespaceCPUQuery(namespace string) string {
	selector := fmt.Sprintf(`namespace="%s"`, namespace)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (rate(container_cpu_usage_seconds_total{%s}[5m])) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="cpu"})) * 100`,
		selector, selector,
	)
}

// BuildNamespaceRAMQuery builds a query for all pods in a namespace
func BuildNamespaceRAMQuery(namespace string) string {
	selector := fmt.Sprintf(`namespace="%s"`, namespace)
	return fmt.Sprintf(
		`(sum by (pod, namespace, container) (container_memory_working_set_bytes{%s}) / 
		  sum by (pod, namespace, container) (kube_pod_container_resource_requests{%s, resource="memory"})) * 100`,
		selector, selector,
	)
}

// BuildTimeRangeQuery builds a time range query for historical data
func BuildTimeRangeQuery(baseQuery string, start, end int64, step string) string {
	return fmt.Sprintf("%s[%d:%d:%s]", baseQuery, start, end, step)
}

// BuildRateQuery builds a rate query for a metric
func BuildRateQuery(metricName string, window string) string {
	return fmt.Sprintf("rate(%s[%s])", metricName, window)
}

// BuildAverageQuery builds an average query over a time window
func BuildAverageQuery(metricName string, window string) string {
	return fmt.Sprintf("avg_over_time(%s[%s])", metricName, window)
}

// BuildMaxQuery builds a max query over a time window
func BuildMaxQuery(metricName string, window string) string {
	return fmt.Sprintf("max_over_time(%s[%s])", metricName, window)
}

// BuildMinQuery builds a min query over a time window
func BuildMinQuery(metricName string, window string) string {
	return fmt.Sprintf("min_over_time(%s[%s])", metricName, window)
}

// BuildSumQuery builds a sum query over a time window
func BuildSumQuery(metricName string, window string) string {
	return fmt.Sprintf("sum_over_time(%s[%s])", metricName, window)
}

// FilterByLabel adds a label filter to a query
func FilterByLabel(query, labelName, labelValue string) string {
	return fmt.Sprintf("%s{%s=\"%s\"}", query, labelName, labelValue)
}

// FilterByLabelRegex adds a regex label filter to a query
func FilterByLabelRegex(query, labelName, labelPattern string) string {
	return fmt.Sprintf("%s{%s=~\"%s\"}", query, labelName, labelPattern)
}

// GroupBy groups results by specified labels
func GroupBy(query string, labels []string) string {
	return fmt.Sprintf("sum(%s) by (%s)", query, strings.Join(labels, ", "))
}

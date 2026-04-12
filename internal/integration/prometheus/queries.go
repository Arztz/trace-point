package prometheus

import (
	"fmt"
	"strings"
)

// Query builders for Prometheus metrics

// BuildCPUUtilizationQuery builds a query for CPU utilization percentage
// Formula: container_cpu_usage_seconds_total / kube_pod_container_resource_requests (cpu)
// Returns: percentage of CPU used vs CPU request
func BuildCPUUtilizationQuery(namespaces []string, excludePatterns []string) string {
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
	query := fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{%s}[5m])) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="cpu"}) by (pod, namespace, container) * 100`,
		selector, selector,
	)

	return query
}

// BuildRAMUtilizationQuery builds a query for RAM utilization percentage
// Formula: container_memory_working_set_bytes / kube_pod_container_resource_requests (memory)
// Returns: percentage of RAM used vs RAM request
func BuildRAMUtilizationQuery(namespaces []string, excludePatterns []string) string {
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
	query := fmt.Sprintf(
		`sum(rate(container_memory_working_set_bytes{%s}[5m])) by (pod, namespace, container) / sum(rate(kube_pod_container_resource_requests{%s, resource="memory"}[5m])) by (pod, namespace, container) * 100`,
		selector, selector,
	)

	return query
}

// BuildContainerCPUQuery builds a query for a specific container's CPU usage
func BuildContainerCPUQuery(podName, namespace, containerName string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s", container="%s"`, podName, namespace, containerName)
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{%s}[5m])) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="cpu"}) by (pod, namespace, container) * 100`,
		selector, selector,
	)
}

// BuildContainerRAMQuery builds a query for a specific container's RAM usage
func BuildContainerRAMQuery(podName, namespace, containerName string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s", container="%s"`, podName, namespace, containerName)
	return fmt.Sprintf(
		`sum(rate(container_memory_working_set_bytes{%s}[5m])) by (pod, namespace, container) / sum(rate(kube_pod_container_resource_requests{%s, resource="memory"}[5m])) by (pod, namespace, container) * 100`,
		selector, selector,
	)
}

// BuildPodCPUQuery builds a query for all containers in a pod
func BuildPodCPUQuery(podName, namespace string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s"`, podName, namespace)
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{%s}[5m])) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="cpu"}) by (pod, namespace, container) * 100`,
		selector, selector,
	)
}

// BuildPodRAMQuery builds a query for all containers in a pod
func BuildPodRAMQuery(podName, namespace string) string {
	selector := fmt.Sprintf(`pod="%s", namespace="%s"`, podName, namespace)
	return fmt.Sprintf(
		`sum(container_memory_working_set_bytes{%s}) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="memory"}) by (pod, namespace, container) * 100`,
		selector, selector,
	)
}

// BuildNamespaceCPUQuery builds a query for all pods in a namespace
func BuildNamespaceCPUQuery(namespace string) string {
	selector := fmt.Sprintf(`namespace="%s"`, namespace)
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{%s}[5m])) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="cpu"}) by (pod, namespace, container) * 100`,
		selector, selector,
	)
}

// BuildNamespaceRAMQuery builds a query for all pods in a namespace
func BuildNamespaceRAMQuery(namespace string) string {
	selector := fmt.Sprintf(`namespace="%s"`, namespace)
	return fmt.Sprintf(
		`sum(container_memory_working_set_bytes{%s}) by (pod, namespace, container) / sum(kube_pod_container_resource_requests{%s, resource="memory"}) by (pod, namespace, container) * 100`,
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

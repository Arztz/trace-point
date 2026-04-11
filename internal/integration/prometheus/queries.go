package prometheus

import (
	"fmt"
	"strings"
)

// Query builders for Prometheus metrics

// BuildCPUUtilizationQuery builds a query for CPU utilization percentage
// Formula: container_cpu_usage_seconds_total / container_spec_cpu_quota
// Returns: percentage of CPU used vs CPU limit
func BuildCPUUtilizationQuery(namespaces []string, excludePatterns []string) string {
	var conditions []string

	// Filter by namespace if specified
	if len(namespaces) > 0 {
		nsConditions := make([]string, len(namespaces))
		for i, ns := range namespaces {
			nsConditions[i] = fmt.Sprintf(`namespace="%s"`, ns)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(nsConditions, "|")))
	}

	// Exclude patterns if specified
	for _, pattern := range excludePatterns {
		conditions = append(conditions, fmt.Sprintf(`pod!~"%s"`, pattern))
	}

	// Base query: CPU usage / CPU limit as percentage
	query := `sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod, namespace, container) / sum(container_spec_cpu_quota{container!=""}) by (pod, namespace, container) * 100`

	if len(conditions) > 0 {
		query += fmt.Sprintf(" and %s", strings.Join(conditions, ", "))
	}

	return query
}

// BuildRAMUtilizationQuery builds a query for RAM utilization percentage
// Formula: container_memory_working_set_bytes / container_spec_memory_limit
// Returns: percentage of RAM used vs RAM limit
func BuildRAMUtilizationQuery(namespaces []string, excludePatterns []string) string {
	var conditions []string

	// Filter by namespace if specified
	if len(namespaces) > 0 {
		nsConditions := make([]string, len(namespaces))
		for i, ns := range namespaces {
			nsConditions[i] = fmt.Sprintf(`namespace="%s"`, ns)
		}
		conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(nsConditions, "|")))
	}

	// Exclude patterns if specified
	for _, pattern := range excludePatterns {
		conditions = append(conditions, fmt.Sprintf(`pod!~"%s"`, pattern))
	}

	// Base query: Working set / Memory limit as percentage
	query := `container_memory_working_set_bytes{container!=""} / container_spec_memory_limit{container!=""} * 100`

	if len(conditions) > 0 {
		query += fmt.Sprintf(" and %s", strings.Join(conditions, ", "))
	}

	return query
}

// BuildContainerCPUQuery builds a query for a specific container's CPU usage
func BuildContainerCPUQuery(podName, namespace, containerName string) string {
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{pod="%s", namespace="%s", container="%s"}[5m])) by (pod, namespace, container) / sum(container_spec_cpu_quota{pod="%s", namespace="%s", container="%s"}) by (pod, namespace, container) * 100`,
		podName, namespace, containerName, podName, namespace, containerName,
	)
}

// BuildContainerRAMQuery builds a query for a specific container's RAM usage
func BuildContainerRAMQuery(podName, namespace, containerName string) string {
	return fmt.Sprintf(
		`container_memory_working_set_bytes{pod="%s", namespace="%s", container="%s"} / container_spec_memory_limit{pod="%s", namespace="%s", container="%s"} * 100`,
		podName, namespace, containerName, podName, namespace, containerName,
	)
}

// BuildPodCPUQuery builds a query for all containers in a pod
func BuildPodCPUQuery(podName, namespace string) string {
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{pod="%s", namespace="%s"}[5m])) by (pod, namespace, container) / sum(container_spec_cpu_quota{pod="%s", namespace="%s"}) by (pod, namespace, container) * 100`,
		podName, namespace, podName, namespace,
	)
}

// BuildPodRAMQuery builds a query for all containers in a pod
func BuildPodRAMQuery(podName, namespace string) string {
	return fmt.Sprintf(
		`sum(container_memory_working_set_bytes{pod="%s", namespace="%s"}) by (pod, namespace, container) / sum(container_spec_memory_limit{pod="%s", namespace="%s"}) by (pod, namespace, container) * 100`,
		podName, namespace, podName, namespace,
	)
}

// BuildNamespaceCPUQuery builds a query for all pods in a namespace
func BuildNamespaceCPUQuery(namespace string) string {
	return fmt.Sprintf(
		`sum(rate(container_cpu_usage_seconds_total{namespace="%s"}[5m])) by (pod, namespace, container) / sum(container_spec_cpu_quota{namespace="%s"}) by (pod, namespace, container) * 100`,
		namespace, namespace,
	)
}

// BuildNamespaceRAMQuery builds a query for all pods in a namespace
func BuildNamespaceRAMQuery(namespace string) string {
	return fmt.Sprintf(
		`sum(container_memory_working_set_bytes{namespace="%s"}) by (pod, namespace, container) / sum(container_spec_memory_limit{namespace="%s"}) by (pod, namespace, container) * 100`,
		namespace, namespace,
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

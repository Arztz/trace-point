# Application Requirement Document (ARD): Resource-to-Code Correlation Engine

| Document Information | |
|---|---|
| **Document Title** | Application Requirement Document: Resource-to-Code Correlation Engine |
| **Version** | 1.0 |
| **Date** | 2026-04-11 |
| **Author** | Business Analysis Team |
| **Status** | Approved for Development |
| **Classification** | Internal Technical Document |

---

## 1. Executive Summary

The Resource-to-Code Correlation Engine is an automated tool designed to solve the persistent "Resource Imbalance" problem in Kubernetes-based microservices environments. This application bridges the gap between infrastructure-level resource spikes and code-level root cause analysis by correlating data from multiple observability platforms.

The tool addresses a critical operational challenge: when a single service hosts multiple API endpoints with vastly different CPU and RAM footprints, traditional monitoring tools can identify that a service is under stress, but cannot pinpoint which specific API route or code function is responsible. This leads to extended Root Cause Analysis (RCA) cycles, manual debugging efforts, and delayed resolution of performance regressions.

The Resource-to-Code Correlation Engine automated this entire workflow through a Golang-based backend that polls Prometheus for resource metrics, correlates findings with Signoz/Clickhouse trace data, and enriches alerts with Gcloud Profiler samples. The system provides a unified dashboard for visualization, Discord webhook notifications for real-time alerting, and JSON export capabilities for documentation and refactoring planning.

**Key Business Outcomes:**
- Reduction in RCA time from hours to minutes
- Automated identification of culprit code functions
- Proactive identification of services requiring architectural refactoring
- Elimination of manual cross-referencing between observability tools

---

## 2. Business Background and Problem Statement

### 2.1 Problem Context

Modern Kubernetes deployments often consolidate multiple API endpoints within single microservices. While this architectural approach simplifies deployment and management, it creates a significant observability challenge: when the overall service experiences CPU or memory spikes, operations teams cannot easily determine which specific API route or code function is responsible.

Consider a service exposing the following endpoints:
- `/v1/health` (minimal resource footprint)
- `/v1/users` (moderate resource usage)
- `/v1/summarize` (heavy computational processing)

When the service's CPU utilization spikes to 95%, operations teams must manually:
1. Identify that a spike occurred (via Prometheus)
2. Determine which API routes were active during that period (via logging)
3. Query trace systems to understand request patterns (via Signoz)
4. Analyze profiling data to identify specific functions (via Gcloud Profiler)
5. Correlate all findings to produce a conclusion

This manual process typically takes 2-4 hours per incident and requires expertise across multiple tooling systems.

### 2.2 Business Problem Statement

**Problem:** The current operational workflow requires manual correlation between infrastructure metrics, tracing data, and profiling information to identify the root cause of resource spikes. This manual process is time-consuming, error-prone, and delays incident resolution.

**Impact:**
- Extended downtime during resource-related incidents
- Increased MTTR (Mean Time To Resolution) for performance issues
- Inefficient utilization of SRE and DevOps engineering time
- Delayed identification of services requiring architectural refactoring
- Alert fatigue from incomplete or unactionable notifications

**Root Cause:** Lack of automated correlation between infrastructure-level metrics and code-level profiling data, requiring engineers to manually cross-reference multiple observability platforms.

### 2.3 Proposed Solution

The Resource-to-Code Correlation Engine automates the entire correlation workflow by:
1. Continuously polling Prometheus for resource utilization metrics
2. Automatically detecting threshold breaches based on configurable moving averages
3. Waiting for the Gcloud Profiler sampling cycle to complete (5-10 minute buffer)
4. Querying Signoz/Clickhouse for active trace IDs during the spike period
5. Fetching Gcloud Profiler samples to identify specific culprit functions
6. Generating unified alerts with complete RCA information
7. Providing a dashboard for historical analysis and drill-down capabilities

---

## 3. Stakeholders

### 3.1 Primary Stakeholders

| Stakeholder | Role | Responsibilities | Interests |
|---|---|---|---|
| **SRE Team** | Primary User | Respond to alerts, perform RCA, coordinate incident response | Rapid incident resolution, actionable alerts, reduced manual effort |
| **DevOps Engineers** | Primary User | Monitor infrastructure health, configure alerting thresholds | Clear visibility into resource utilization, proactive issue identification |
| **Backend Developers** | Secondary User | Understand code performance impact, refactor problematic functions | Clear identification of performance bottlenecks, actionable profiling data |
| **Tech Lead / Architect** | Secondary User | Make architectural decisions, plan refactoring initiatives | Resource Gravity Scores, JSON export for planning, service separation recommendations |

### 3.2 Stakeholder Requirements Summary

- **SRE/DevOps:** Require real-time alerts with complete RCA information delivered to Discord
- **Developers:** Need visual dashboards with drill-down capability to trace IDs and profiler data
- **Architects:** Require JSON exports for JIRA integration and strategic planning

---

## 4. Functional Requirements

### 4.1 Dashboard and Visualization Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| FR-001 | The system shall display a unified timeline graph showing CPU utilization as a line chart and RAM utilization as an area chart, synchronized on a common time axis | Must Have |
| FR-002 | The system shall overlay visual markers on the timeline indicating which API routes were active during specific resource spikes | Must Have |
| FR-003 | The dashboard shall support time range selection with options for last 1 hour, 6 hours, 24 hours, and 7 days | Must Have |
| FR-004 | The system shall provide interactive drill-down capability where clicking on a spike reveals associated Trace IDs | Must Have |
| FR-005 | The drill-down view shall display the Gcloud Profiler flamegraph or function table for the selected spike period | Must Have |
| FR-006 | The dashboard shall include a manual refresh button to reload data on demand | Should Have |
| FR-007 | The system shall display resource utilization as a percentage of Kubernetes-defined limits | Must Have |
| FR-008 | The dashboard shall support filtering by namespace and pod name | Should Have |

### 4.2 Detection and Alerting Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| FR-010 | The system shall continuously poll Prometheus at configurable intervals (default: 30 seconds) for CPU and RAM metrics | Must Have |
| FR-011 | The system shall detect resource spikes using the formula: `Current Usage > Moving Average + (Threshold %)` | Must Have |
| FR-011a | The system shall calculate moving average using a 30-minute rolling window | Must Have |
| FR-011b | The system shall implement a 30-minute baseline learning period before spike detection begins | Must Have |
| FR-012 | The system shall allow users to configure the spike detection threshold percentage (default: 50%) | Must Have |
| FR-013 | The system shall implement a reconciliation buffer of 5-10 minutes between spike detection and alert generation | Must Have |
| FR-014 | The reconciliation buffer shall ensure Gcloud Profiler data is available before alert finalization | Must Have |
| FR-015 | The system shall send alerts via Discord webhook containing: Route Name, Resource Impact (CPU/RAM delta), Culprit Function/File Path | Must Have |
| FR-016 | The system shall implement a cooldown period between alerts for the same resource to prevent alert fatigue (default: 15 minutes) | Must Have |
| FR-017 | The system shall allow configuration of cooldown period duration | Should Have |
| FR-018 | The system shall store spike metadata in SQLite for 7 days | Must Have |
| FR-019 | The system shall generate alert summaries in JSON format for export | Must Have |

### 4.3 Refactoring Intelligence Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| FR-020 | The system shall calculate a Resource Gravity Score using formula: `High Score = (High Resource Peak) × (Low Call Frequency)` | Must Have |
| FR-021 | The system shall identify routes matching naming patterns (regex: `/tasks/*`, `/batch/*`, `/jobs/*`) | Must Have |
| FR-022 | The system shall auto-tag routes with high latency and resource usage as `[Suspected Job]` | Must Have |
| FR-023 | The system shall provide JSON export of refactoring recommendations including: service name, suggested separation strategy, resource gravity score | Must Have |
| FR-024 | The export shall include rationale for microservice separation recommendations | Should Have |

### 4.4 Data Correlation Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| FR-025 | The system shall map Pod spikes to active API routes via Signoz/Clickhouse trace data | Must Have |
| FR-025a | When multiple routes are active during a spike, the system shall identify the culprit route as the one with highest resource consumption (CPU + RAM) | Must Have |
| FR-026 | The system shall correlate trace IDs with Gcloud Profiler samples | Must Have |
| FR-027 | The final output shall follow the correlation chain: Pod Spike → Route → Trace ID → Function Name | Must Have |
| FR-028 | The system shall query Signoz/Clickhouse for trace data within a configurable time window (default: 5 minutes around spike) | Should Have |

### 4.5 Data Storage and Export Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| FR-030 | The system shall store spike events with timestamp, pod name, namespace, CPU delta, RAM delta, route name, trace ID, and culprit function | Must Have |
| FR-031 | The SQLite database shall retain data for 7 days before auto-purging | Must Have |
| FR-032 | The system shall provide JSON export of spike history for the past 7 days | Must Have |
| FR-033 | The export shall include culprit function details for documentation purposes | Must Have |
| FR-034 | The system shall support export to file with timestamp-based naming convention | Should Have |

---

## 5. Non-Functional Requirements

### 5.1 Performance Requirements

| Requirement ID | Requirement Description | Target Metric |
|---|---|---|
| NFR-001 | Prometheus polling shall complete within 5 seconds | ≤ 5 seconds |
| NFR-002 | Signoz/Clickhouse query response time shall not exceed 10 seconds | ≤ 10 seconds |
| NFR-003 | Gcloud Profiler API fetch shall complete within 15 seconds | ≤ 15 seconds |
| NFR-004 | Dashboard page load time shall not exceed 3 seconds | ≤ 3 seconds |
| NFR-005 | The system shall support concurrent monitoring of up to 100 pods | 100 pods |
| NFR-006 | Alert generation from spike detection shall complete within 2 minutes | ≤ 2 minutes |
| NFR-007 | The system shall handle up to 1000 spikes per day without performance degradation | 1000 spikes/day |

### 5.2 Scalability Requirements

| Requirement ID | Requirement Description | Target Metric |
|---|---|---|
| NFR-010 | The system shall scale horizontally to monitor additional pods without code changes | Linear scaling |
| NFR-011 | SQLite database shall handle up to 10,000 spike records without performance degradation | 10,000 records |
| NFR-012 | Dashboard shall render timeline data for up to 7 days without browser performance issues | 7 days data |

### 5.3 Security Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| NFR-020 | Authentication shall rely on local gcloud authentication (`gcloud auth application-default login`) | Must Have |
| NFR-021 | All API credentials shall be stored in local environment variables | Must Have |
| NFR-022 | Database file shall be stored in user home directory with appropriate file permissions (600) | Must Have |
| NFR-023 | Discord webhook URL shall be stored in environment variable, never logged or displayed | Must Have |

### 5.4 Reliability Requirements

| Requirement ID | Requirement Description | Target Metric |
|---|---|---|
| NFR-030 | System uptime shall be 99.5% excluding planned maintenance | 99.5% |
| NFR-031 | Automatic recovery from Prometheus connection failure shall occur within 60 seconds | ≤ 60 seconds |
| NFR-032 | Graceful degradation shall occur if Signoz/Clickhouse is unavailable (alert without trace data) | Graceful |
| NFR-033 | Spike detection shall continue during Gcloud Profiler unavailability | Continuous |

### 5.5 Usability Requirements

| Requirement ID | Requirement Description | Priority |
|---|---|---|
| NFR-040 | Dashboard shall be accessible via web browser on localhost | Must Have |
| NFR-041 | Configuration shall be managed via configuration file (config.yaml) | Must Have |
| NFR-042 | User documentation shall be provided for all major features | Must Have |

---

## 6. Technical Constraints

### 6.1 Architecture Constraints

| Constraint ID | Description |
|---|---|
| TC-001 | Backend must be implemented in Golang |
| TC-002 | Database must be SQLite for local storage |
| TC-003 | Frontend must run in web browser (locally) |
| TC-004 | Database retention period is fixed at 7 days |
| TC-005 | System must support offline operation for data viewing |

### 6.2 Integration Constraints

| Constraint ID | Description |
|---|---|
| TC-010 | System must connect to Prometheus via internal DNS |
| TC-011 | System must connect to Signoz/Clickhouse via internal DNS |
| TC-012 | System must utilize Gcloud SDK for Profiler access |
| TC-013 | System must work within Kubernetes environment monitoring 100 pods across 3-4 namespaces |
| TC-014 | Prometheus integration requires CPU/RAM utilization vs. K8s Limits metric access |
| TC-015 | Signoz integration requires request-level data and Trace IDs |
| TC-016 | Gcloud Profiler integration requires code-level execution samples |

### 6.3 Infrastructure Constraints

| Constraint ID | Description |
|---|---|
| TC-020 | System is designed for local deployment (not cloud-hosted) |
| TC-021 | System relies on user-local gcloud authentication |
| TC-022 | System requires network access to internal DNS addresses for Prometheus, Signoz |

---

## 7. User Stories

### 7.1 Dashboard User Stories

| Story ID | User Story | Acceptance Criteria |
|---|---|---|
| US-001 | As an SRE, I want to view a unified timeline of CPU and RAM utilization so that I can quickly identify resource spikes | Dashboard displays synchronized CPU line chart and RAM area chart; spikes are visually identifiable |
| US-002 | As a DevOps engineer, I want to see which API routes were active during resource spikes so that I can correlate usage patterns | Route overlays appear on timeline at spike timestamps |
| US-003 | As an SRE, I want to click on a spike and see the associated Trace ID and Profiler data so that I can perform root cause analysis | Click on spike displays trace ID and profiler flamegraph/table |
| US-004 | As a developer, I want to filter dashboard data by namespace and pod so that I can focus on specific services | Filter controls allow namespace and pod selection |

### 7.2 Alerting User Stories

| Story ID | User Story | Acceptance Criteria |
|---|---|---|
| US-010 | As an SRE, I want to receive Discord alerts with complete RCA information so that I can respond immediately | Discord alert contains: Route Name, Resource Impact, Culprit Function |
| US-011 | As a DevOps engineer, I want to configure spike detection thresholds so that I can tune sensitivity | Threshold percentage is configurable via config file |
| US-012 | As an SRE, I want to avoid alert fatigue so that I remain responsive to genuine issues | Cooldown period prevents duplicate alerts within 15 minutes |

### 7.3 Refactoring Intelligence User Stories

| Story ID | User Story | Acceptance Criteria |
|---|---|---|
| US-020 | As a Tech Lead, I want to see Resource Gravity Scores so that I can identify services needing refactoring | Resource Gravity Scores displayed on dashboard |
| US-021 | As an Architect, I want to export refactoring recommendations as JSON so that I can create JIRA tickets | JSON export available with service separation recommendations |

### 7.4 Data Management User Stories

| Story ID | User Story | Acceptance Criteria |
|---|---|---|
| US-030 | As a user, I want to export spike history as JSON so that I can document incidents | JSON export available for past 7 days |
| US-031 | As a user, I want the system to automatically purge old data so that I don't manage storage | Data automatically purged after 7 days |

---

## 8. Use Cases

### 8.1 Primary Use Cases

#### Use Case UC-001: Automatic Spike Detection and Alerting

**Actor:** System (automated)

**Preconditions:**
- System is running and connected to Prometheus
- Gcloud authentication is configured
- Discord webhook URL is configured

**Flow:**
1. System polls Prometheus for CPU and RAM metrics every 30 seconds
2. System calculates moving average for each monitored pod
3. System compares current usage to moving average + threshold
4. If spike detected, system logs spike event with timestamp
5. System waits 5-10 minutes (reconciliation buffer)
6. System queries Signoz/Clickhouse for trace data during spike period
7. System fetches Gcloud Profiler samples for spike period
8. System correlates all data to identify culprit function
9. System sends Discord alert with complete RCA information
10. System applies cooldown period before detecting next spike

**Postconditions:**
- Spike event stored in SQLite
- Discord alert sent with Route Name, Resource Impact, Culprit Function

**Exception Handling:**
- If Signoz/Clickhouse unavailable: Alert sent without trace data
- If Gcloud Profiler unavailable: Alert sent without culprit function

#### Use Case UC-002: Dashboard Visualization and Drill-Down

**Actor:** SRE, Developer

**Preconditions:**
- Web browser accessing localhost dashboard
- SQLite database contains spike data

**Flow:**
1. User navigates to dashboard URL
2. Dashboard loads timeline with default 24-hour view
3. User sees CPU and RAM charts with route overlays
4. User clicks on a spike in the timeline
5. System displays drill-down panel with Trace ID
6. User clicks to view Profiler data
7. System displays flamegraph or function table

**Postconditions:**
- User has performed root cause analysis

#### Use Case UC-003: Refactoring Intelligence Export

**Actor:** Tech Lead, Architect

**Preconditions:**
- System has collected at least 7 days of data

**Flow:**
1. User requests JSON export from dashboard
2. System calculates Resource Gravity Scores for all monitored routes
3. System identifies routes matching job-like patterns
4. System generates JSON with separation recommendations
5. User downloads JSON file for JIRA integration

**Postconditions:**
- JSON file available for strategic planning

---

## 9. Data Requirements

### 9.1 Data Entities

#### Spike Event Entity

| Field | Type | Description | Source |
|---|---|---|---|
| ID | UUID | Unique identifier | Auto-generated |
| Timestamp | DateTime | When spike occurred | Prometheus |
| PodName | String | Name of affected pod | Prometheus |
| Namespace | String | Kubernetes namespace | Prometheus |
| CpuUsagePercent | Float | CPU utilization % | Prometheus |
| CpuLimitPercent | Float | CPU limit % | Prometheus |
| RamUsagePercent | Float | RAM utilization % | Prometheus |
| RamLimitPercent | Float | RAM limit % | Prometheus |
| ThresholdPercent | Float | Configured threshold | Config |
| MovingAveragePercent | Float | Calculated average | Calculated |
| RouteName | String | Primary route during spike | Signoz/Clickhouse |
| TraceID | String | Associated trace | Signoz/Clickhouse |
| CulpritFunction | String | Function from profiler | Gcloud Profiler |
| CulpritFilePath | String | File path from profiler | Gcloud Profiler |
| AlertSent | Boolean | Whether Discord alert sent | System |
| CooldownEnd | DateTime | End of alert cooldown | System |

#### Configuration Entity

| Field | Type | Description |
|---|---|---|
| PrometheusURL | String | Prometheus endpoint |
| SignozURL | String | Signoz/Clickhouse endpoint |
| GcloudProjectID | String | GCP project identifier |
| SpikeThresholdPercent | Float | Detection threshold |
| PollingIntervalSeconds | Integer | Poll frequency |
| ReconciliationBufferMinutes | Integer | Buffer before alert |
| CooldownMinutes | Integer | Alert cooldown period |
| DiscordWebhookURL | String | Discord endpoint |
| DatabasePath | String | SQLite file location |

### 9.2 Data Retention

- **Spike Events:** 7 days in SQLite, then auto-purged
- **Configuration:** Persisted in config.yaml
- **Logs:** Application logs retained per system rotation policy

### 9.3 Data Flow

```
Prometheus → Golang Backend → SQLite (Storage)
                ↓
         Signoz/Clickhouse → Correlation Logic
                ↓
         Gcloud Profiler → Enrichment
                ↓
         Discord Webhook → Alert
                ↓
         Web Dashboard → Visualization
```

---

## 10. Integration Requirements

### 10.1 Prometheus Integration

| Requirement ID | Integration Requirement |
|---|---|
| INT-001 | System shall connect to Prometheus via HTTP/HTTPS |
| INT-002 | System shall query `container_cpu_usage_seconds_total` metric |
| INT-003 | System shall query `container_memory_working_set_bytes` metric |
| INT-004 | System shall retrieve Kubernetes limits from pod specifications |
| INT-005 | System shall handle Prometheus authentication via internal DNS |

### 10.2 Signoz/Clickhouse Integration

| Requirement ID | Integration Requirement |
|---|---|
| INT-010 | System shall connect to Signoz/Clickhouse via HTTP |
| INT-011 | System shall query trace data by time range and pod selector |
| INT-012 | System shall retrieve trace IDs active during spike period |
| INT-013 | System shall extract route information from trace spans |
| INT-014 | System shall handle Signoz authentication via internal DNS |

### 10.3 Gcloud Profiler Integration

| Requirement ID | Integration Requirement |
|---|---|
| INT-020 | System shall utilize Gcloud SDK for profiler access |
| INT-021 | System shall query profiler by time range and service name |
| INT-022 | System shall retrieve flamegraph data for spike period |
| INT-023 | System shall extract function names and file paths from profiler |
| INT-024 | System shall authenticate via `gcloud auth application-default` |

### 10.4 Discord Integration

| Requirement ID | Integration Requirement |
|---|---|
| INT-030 | System shall send POST requests to Discord webhook URL |
| INT-031 | Payload shall be formatted as Discord embed |
| INT-032 | Alert shall include: title, description, fields for Route/Impact/Culprit |
| INT-033 | System shall handle webhook failures gracefully |

---

## 11. Acceptance Criteria

### 11.1 Dashboard Acceptance Criteria

| ID | Criterion | Test Method |
|---|---|---|
| AC-001 | Dashboard displays CPU line chart and RAM area chart synchronized on time axis | Visual inspection |
| AC-002 | Route overlays appear on timeline at spike timestamps | Generate test spike, verify overlay |
| AC-003 | Clicking spike reveals trace ID and profiler data | Manual click test |
| AC-004 | Time range selection works for 1h, 6h, 24h, 7d | Select each option, verify data |
| AC-005 | Dashboard loads within 3 seconds | Performance measurement |
| AC-006 | Filter by namespace and pod functions correctly | Apply filters, verify results |

### 11.2 Alerting Acceptance Criteria

| ID | Criterion | Test Method |
|---|---|---|
| AC-010 | Spike detection triggers when usage > average + threshold | Inject test data, verify detection |
| AC-011 | Reconciliation buffer delays alert 5-10 minutes | Time detection to alert |
| AC-012 | Discord alert contains Route, Impact, Culprit | Inspect received alert |
| AC-013 | Cooldown prevents duplicate alerts | Generate spikes within cooldown |
| AC-014 | Threshold is configurable via config file | Modify config, verify behavior |

### 11.3 Refactoring Intelligence Acceptance Criteria

| ID | Criterion | Test Method |
|---|---|---|
| AC-020 | Resource Gravity Score calculated correctly | Verify formula implementation |
| AC-021 | Routes matching `/tasks/*` pattern are tagged | Create test routes, verify tagging |
| AC-022 | JSON export contains separation recommendations | Generate export, verify content |

### 11.4 Integration Acceptance Criteria

| ID | Criterion | Test Method |
|---|---|---|
| AC-030 | Prometheus connection successful | Verify metrics retrieved |
| AC-031 | Signoz connection successful | Verify traces retrieved |
| AC-032 | Gcloud Profiler accessible | Verify samples retrieved |
| AC-033 | Discord webhook delivers alerts | Verify message received |

### 11.5 System Acceptance Criteria

| ID | Criterion | Test Method |
|---|---|---|
| AC-040 | System runs on Golang backend | Verify binary executes |
| AC-041 | SQLite stores 7 days of data | Wait 7 days, verify data |
| AC-042 | Browser-accessible dashboard | Access via localhost |
| AC-043 | gcloud authentication sufficient | Verify without additional auth |

---

## 12. Risk Assessment

### 12.1 Risk Identification

| Risk ID | Risk Description | Likelihood | Impact | Mitigation |
|---|---|---|---|---|
| R-001 | Gcloud Profiler data unavailable during spike | Medium | High | Send alert without culprit function; continue monitoring |
| R-002 | Signoz/Clickhouse connection failure | Medium | Medium | Cache recent trace data; graceful degradation |
| R-003 | Prometheus query performance degradation | Low | High | Implement query optimization; increase polling interval |
| R-004 | Discord webhook rate limiting | Low | Medium | Implement exponential backoff; queue alerts |
| R-005 | SQLite database growth beyond capacity | Low | Medium | Auto-purge at 7 days; implement size monitoring |
| R-006 | Kubernetes namespace permission issues | Medium | High | Validate permissions during setup; clear error messages |
| R-007 | Gcloud authentication token expiration | Medium | High | Implement token refresh; monitor authentication status |
| R-008 | Network latency affecting correlation accuracy | Low | Low | Increase reconciliation buffer; log correlation delays |

### 12.2 Risk Matrix

| Impact / Likelihood | Low | Medium | High |
|---|---|---|---|
| **High** | - | R-002, R-004 | R-001, R-006, R-007 |
| **Medium** | R-005 | R-008 | - |
| **Low** | R-003 | - | - |

### 12.3 Contingency Plans

- **R-001 (Profiler unavailable):** System continues monitoring but sends alerts without culprit function data; logs warning for later investigation
- **R-002 (Signoz unavailable):** System uses cached trace data when available; degrades to metric-only alerts when no cache exists
- **R-006 (K8s permissions):** Setup validation checks permissions before monitoring; provides clear error messages for missing access

---

## 13. Glossary of Terms

| Term | Definition |
|---|---|
| **CPU Utilization** | The percentage of CPU resources consumed by a container relative to its defined limit |
| **RAM Utilization** | The amount of memory used by a container relative to its defined limit |
| **Moving Average** | Statistical calculation averaging metric values over a rolling time window (typically 5 minutes) |
| **Spike Detection** | Automated identification when current resource usage significantly exceeds historical baseline |
| **Reconciliation Buffer** | Delay between spike detection and alert generation to allow profiling data to become available |
| **Trace ID** | Unique identifier for a distributed trace, correlating requests across services |
| **Flamegraph** | Visualization of profiling data showing function call hierarchy and resource consumption |
| **Resource Gravity Score** | Calculated metric identifying APIs with high resource usage but low call frequency |
| **Root Cause Analysis (RCA)** | Process of identifying the underlying cause of a system issue |
| **MTTR** | Mean Time To Resolution - average time to resolve an incident |
| **Prometheus** | Open-source monitoring and alerting toolkit for Kubernetes |
| **Signoz** | Open-source distributed tracing platform |
| **Clickhouse** | Columnar database used by Signoz for trace storage |
| **Gcloud Profiler** | Google Cloud profiling tool for identifying code-level performance bottlenecks |
| **Kubernetes Pod** | Smallest deployable unit in Kubernetes, containing one or more containers |
| **Namespace** | Kubernetes namespace for organizing and isolating resources |
| **Webhook** | User-defined HTTP callback for automated notifications |

| **Moving Average Window** | Rolling window for calculating baseline CPU/RAM averages - currently set to 30 minutes |
| **Primary Route Logic** | When multiple routes are active during spike, culprit is route with highest CPU consumption (primary) + RAM (secondary) | ✅ Resolved |
| **Baseline Learning Period** | Initial period (30 minutes) where spike detection is paused to build baseline - data collected silently, no alerts sent | ✅ Resolved |
| **Post-Restart Behavior** | Resume immediately using historical data from SQLite | ✅ Resolved |

---

## 14. Appendices

### Appendix A: Configuration File Template

```
prometheus:
  url: "http://prometheus.monitoring.svc.cluster.local:9090"
  
signoz:
  url: "http://signoz-otel-collector.monitoring.svc.cluster.local:4318"
  
gcloud:
  project_id: "your-gcp-project-id"
  profiler_enabled: true
  
detection:
  threshold_percent: 50
  polling_interval_seconds: 30
  moving_average_window_minutes: 30    # FR-011a: baseline calculation window
  baseline_learning_period_minutes: 30   # FR-011b: initial learning mode
  route_selection_method: "cpu_primary" # FR-025a: highest CPU route
  reconciliation_buffer_minutes: 8
  cooldown_minutes: 15
  
discord:
  webhook_url: "${DISCORD_WEBHOOK_URL}"
  
database:
  path: "${HOME}/.trace-point/spike-data.db"
  
namespaces:
  - "production"
  - "staging"
  
pods:
  filter_pattern: ".*"
```

### Appendix B: API Endpoints

| Endpoint | Method | Description |
|---|---|---|
| `/api/v1/spikes` | GET | List spike events |
| `/api/v1/spikes/:id` | GET | Get spike details |
| `/api/v1/timeline` | GET | Get timeline data |
| `/api/v1/export` | GET | Export JSON data |
| `/api/v1/config` | GET | Get current configuration |

### Appendix C: Discord Alert Format

```
Title: [CRITICAL] Resource Spike Detected - pod-name

Fields:
- Route: /v1/summarize
- CPU Impact: +45% (from 30% to 75%)
- RAM Impact: +20% (from 40% to 60%)
- Culprit: processBatch() in workers/processor.go
- Timestamp: 2026-04-11T14:30:00Z
- Trace ID: abc123-xyz789
```

---

## 15. Feasibility Analysis Summary

This section summarizes the feasibility assessment conducted by product management, backend development, and frontend development teams.

### 15.1 Product Management Assessment

**Overall Verdict: 有条件可行 (Feasible with Conditions)**

| Assessment Area | Finding | Priority |
|-----------------|---------|----------|
| Feature Coverage | All major features addressed (Dashboard, Alerting, Refactoring Intelligence) | ✅ Complete |
| Stakeholder Alignment | SRE/DevOps, Developers, Architects requirements mapped | ✅ Complete |
| Risk Coverage | 8 risks identified with mitigations | ✅ Adequate |
| Moving Average Window | Not defined - requires clarification | 🔴 High |
| Primary Route Logic | Missing - how to determine culprit route | 🔴 High |
| Baseline Learning Period | Not specified - implementation blocker | 🔴 High |

**Recommendations:**
1. Define moving average calculation window (recommended: 5 minutes)
2. Clarify primary route identification logic when multiple routes active
3. Specify baseline learning period before spike detection (recommended: 30 minutes minimum)

### 15.2 Backend Technical Assessment

**Overall Verdict: TECHNICALLY FEASIBLE**

| Component | Technical Feasibility | Notes |
|-----------|----------------------|-------|
| Golang concurrent polling | ✅ Fully Feasible | Goroutine model handles 100+ pods |
| SQLite (10,000 records) | ✅ Fully Feasible | Well within operational range |
| Prometheus integration | ✅ Fully Feasible | HTTP API with client library |
| Signoz/Clickhouse integration | ✅ Fully Feasible | HTTP API query |
| Gcloud Profiler integration | ⚠️ Conditional | Requires GCP environment |
| Discord webhook | ✅ Fully Feasible | Standard HTTP POST |
| Spike detection logic | ✅ Fully Feasible | Rolling window algorithm |
| 10-minute reconciliation buffer | ✅ Fully Feasible | State machine pattern |

**Technical Challenges Identified:**
1. **Correlation Accuracy:** Timestamp alignment across services requires fuzzy matching
2. **Gcloud Profiler Availability:** Sampling-based - may not have data for every spike
3. **SQLite Concurrency:** Use WAL mode for concurrent writes
4. **Environment Check:** Verify Kubernetes runs on GCP for Profiler integration

**Performance Targets Feasibility:**
| Target | Assessment |
|--------|------------|
| Prometheus polling ≤5s | ✅ Achievable |
| Signoz query ≤10s | ✅ Achievable |
| Gcloud Profiler fetch ≤15s | ✅ Achievable |
| Dashboard load ≤3s | ✅ Achievable |
| Alert generation ≤2min | ✅ Achievable |

### 15.3 Frontend UI/UX Assessment

**Overall Verdict: FEASIBLE**

| Component | Feasibility | Technology Approach |
|-----------|------------|--------------------|
| CPU line chart | ✅ Fully Achievable | Recharts/Chart.js |
| RAM area chart | ✅ Fully Achievable | Same |
| Route overlay markers | ✅ Fully Achievable | Custom annotations |
| Drill-down to Trace/Profiler | ✅ Fully Achievable | Click handlers |
| Time range selection | ✅ Fully Achievable | Dropdown controls |
| Namespace/Pod filtering | ✅ Fully Achievable | Multi-select |
| JSON export | ✅ Fully Achievable | Blob API |
| 7-day data rendering | ⚠️ Challenge | Backend aggregation needed |

**UI/UX Challenges:**
1. **Data Aggregation:** 7 days at 30s intervals = ~20k points - requires server-side aggregation
2. **Flamegraph Visualization:** Use D3.js or flamebearer library
3. **Real-time Updates:** Implement incremental updates, not full re-render

**Technology Recommendations:**
- Framework: React 18+ or Vue 3+
- Charts: Recharts (primary) + D3.js (flamegraph)
- State: TanStack Query for polling/caching
- Data: Downsample on backend, render max ~1000 points client-side

### 15.4 Overall Feasibility Conclusion

| Perspective | Verdict | Key Conditions |
|------------|---------|-------------|
| Product Management | 有条件可行 | Clarify: moving average window, route logic, learning period |
| Backend Development | TECHNICALLY FEASIBLE | Environment check for GCP Profiler |
| Frontend Development | FEASIBLE | Backend data aggregation required |

**Final Decision: Proceed to Development**

The requirements are fundamentally sound and achievable. Before development begins:

1. **Product Clarifications (High Priority):**
   - Define moving average time window (FR-011)
   - Define primary route identification logic (FR-025)
   - Define baseline learning period

2. **Technical Prerequisites (Medium Priority):**
   - Verify Kubernetes environment runs on GCP (for Profiler)
   - Confirm Prometheus/Signoz HTTP API access
   - Define data aggregation strategy for 7-day views

3. **Implementation Order (Recommended):**
   Phase 1: Core polling and spike detection
   Phase 2: Database and storage layer
   Phase 3: Prometheus integration
   Phase 4: Signoz correlation
   Phase 5: Gcloud Profiler integration
   Phase 6: Discord alerting
   Phase 7: Dashboard frontend
   Phase 8: Refactoring intelligence export

---

**Document Approval:**

| Role | Name | Date | Signature |
|---|---|---|---|
| Author | Business Analysis Team | 2026-04-11 | |
| Tech Lead | | | |
| Product Owner | | | |

---

*End of Application Requirement Document*
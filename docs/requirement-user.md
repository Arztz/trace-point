# Product Requirements Document: Resource-to-Code Correlation Engine

## 1. Project Vision
To solve the "Resource Imbalance" problem where a single service hosts multiple APIs with vastly different CPU/RAM footprints. This tool will automate the correlation between infrastructure spikes, specific API routes, and the offending code-level functions.

## 2. Technical Architecture
* **Backend:** Golang (for high-concurrency polling and reconciliation).
* **Frontend:** Web Browser (capable of running locally).
* **Database:** SQLite (local storage for 7 days of spike metadata and summaries).
* **Networking:** Connects via Internal DNS to Prometheus and Signoz; utilizes Gcloud SDK for Profiler access.
* **Auth:** Relies on local environment (`gcloud auth application-default login`).

## 3. Infrastructure & Integration Scope
* **Environment:** Kubernetes (monitoring ~100 pods across 3-4 namespaces).
* **Metrics:** Prometheus (CPU/RAM utilization vs. K8s Limits).
* **Tracing:** Signoz / Clickhouse (Request-level data and Trace IDs).
* **Profiling:** Gcloud Profiler (Code-level execution samples).

## 4. Functional Requirements

### 4.1. The "Unified View" Dashboard
* **Correlated Timeline:** A synchronized graph displaying CPU (Line), RAM (Area), and Time. 
* **Route Overlays:** Visual markers on the graph identifying which API routes were active during specific resource spikes.
* **Drill-down Capability:** Manual refresh interface that allows users to click a spike and view the associated Trace ID and Profiler Flamegraph/Table.

### 4.2. Detection & Alerting Logic
* **Spike Logic:** Triggered when `Current Usage > Moving Average + X%` (Threshold is user-configurable).
* **Reconciliation Strategy (The 10-Minute Buffer):** * The system detects a spike via Prometheus but waits **5–10 minutes** before finalizing the report.
    * This ensures Gcloud Profiler has completed its sampling cycle and data is available for retrieval.
* **Discord Webhook:** Sends an alert containing:
    * Route Name (e.g., `/v1/summarize`).
    * Resource Impact (Delta in CPU/RAM).
    * **The Culprit:** Specific function/file path identified by the Profiler.
    * Cooldown period to prevent alert fatigue.

### 4.3. Refactoring Intelligence (Resource Gravity Score)
The tool will identify "Job-like" behavior or heavy APIs that should be separated into dedicated services using the following logic:
* **Formula:** High Score = (High Resource Peak) x (Low Call Frequency).
* **Identification:** * **Naming:** Filters by regex (e.g., `/tasks/*`).
    * **Behavioral:** Auto-tags routes with high latency/resource usage as `[Suspected Job]`.
* **Report:** Provides a JSON export of recommendations for microservice separation.

## 5. Data Flow & Root Cause Analysis
1.  **Detection:** Go backend polls Prometheus; identifies a breach of threshold relative to K8s Limits.
2.  **Correlation:** Queries Signoz/Clickhouse for the Trace IDs active on that Pod at that exact timestamp.
3.  **Deep Dive:** Fetches the Gcloud Profiler sample for that period.
4.  **Final Output:** Maps the Pod Spike → Route → Trace → Function Name.

## 6. Target Deliverables
* **Dashboard:** Visualizing the relationship between code and infrastructure.
* **Discord Alerts:** actionable RCA (Root Cause Analysis) sent to SRE/DevOps teams.
* **JSON Export:** Summary of spike history and culprit functions for JIRA/documentation.

---

### Summary for Tech Lead
The objective is to eliminate manual RCA for "Heavy" APIs. By implementing a 10-minute reconciliation loop in Go, the tool ensures that by the time an SRE receives a Discord alert, the "Culprit Function" from Gcloud Profiler is already attached. This allows for immediate identification of code-level regressions or services that require architectural refactoring (splitting into microservices).
# Goxec - Distributed Remote Code Execution Engine

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)
![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react)
![Docker](https://img.shields.io/badge/Docker-DooD-2496ED?style=flat&logo=docker)
![Redis](https://img.shields.io/badge/Redis-Streams-DC382D?style=flat&logo=redis)
![WebSocket](https://img.shields.io/badge/WebSocket-Realtime-blue?style=flat)

> Distributed, fault-tolerant remote code execution engine with containerized isolation.

---

## 🏗️ Architecture

![Architecture Diagram](assets/goxec_architecture.png)

### Core Components
*   **API Gateway:** HTTP/WebSocket ingress using **Token Bucket** rate limiting for traffic shaping and DDoS protection.
*   **Scheduler:** **Redis Streams** based distributor utilizing Consumer Groups to guarantee exactly-once processing (processing) and at-least-once delivery (retries).
*   **Worker Pool:** Stateless Go workers utilizing **Docker-out-of-Docker (DooD)** patterns to spawn ephemeral, isolated runtimes on demand.

---

## 📐 Design & Constraints

### Concurrency Model
Goxec employs an **Event-Driven Architecture** to strictly decouple job ingestion from execution.
*   The API layer is non-blocking; it pushes events to the Stream and immediately acknowledges receipt (`202 Accepted` / `queued`).
*   Workers consume independently, allowing the system to handle bursts of traffic by buffering backpressure in Redis rather than crashing the compute nodes.

### Isolation Strategy
Security is enforced via **Ephemeral Containerization**.
*   **Mechanism:** Docker-out-of-Docker (DooD) via socket mounting (`/var/run/docker.sock`). This avoids density issues of full VMs while maintaining process-level isolation.
*   **Lifecycle:** Each job spawns a pristine, network-restricted Alpine container that exists solely for the duration of the execution.
*   **Resource Limits:** Containers are constrained by cgroups (Memory/CPU) to prevent noisy neighbor effects.

### Reliability
Fault tolerance is built into the queue consumption protocol.
*   **Crash Recovery:** Jobs are tracked in the **Pending Entry List (PEL)**. If a worker crashes mid-execution, the job remains in the pending state and can be claimed by a recovery consumer.
*   **Dead Letter Queue:** Malformed payloads that fail repeatedly are eventually moved to a **Dead Letter Queue (DLQ)** for inspection.

---

## ⚖️ Trade-offs

*   **Security vs. Performance:** Goxec utilizes **Docker-out-of-Docker (DooD)** by mounting the host socket.
    *   *Benefit:* Extremely low latency and low overhead compared to full VMs.
    *   *Cost:* Weaker isolation than microVMs (like Firecracker) since containers share the host kernel. (Mitigated by strictly limiting container capabilities/resources).
*   **Consistency vs. Availability:** The system prioritizes **Availability (AP)**. In the event of a Redis partition/failure, the design accepts that real-time logs might be dropped (Pub/Sub is fire-and-forget), but job execution state is strictly preserved via Streams to ensure eventual consistency.

---

## 🛠️ Tech Stack

*   **Runtime:** Go 1.25 (API/Worker)
*   **Frontend:** React 19, TypeScript 5.9, Vite 7
*   **Editor/Terminal:** Monaco Editor, XTerm.js
*   **Infrastructure:** Docker Compose, Redis 7 (Streams + Pub/Sub)

---

## 🚀 How to Run

### Prerequisites
*   Docker Desktop (with Docker Compose)
*   Git

### Quick Start
```bash
# 1. Clone the repository
git clone https://github.com/dontdude/goxec.git
cd goxec

# 2. Build and Start Services
docker-compose up --build

# 3. Access the Application
# The frontend is served on port 80.
# Open: http://localhost
```

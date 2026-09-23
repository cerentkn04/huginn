# Huginn

**A lightweight game server fleet manager for indie and small studios — "Agones without Kubernetes."**

Huginn runs your dedicated game servers as Docker containers, tracks their health and player counts through heartbeats, and scales them up and down automatically. When a machine fills up, it provisions a new one on Google Cloud; when a machine sits idle, it removes it. One Go binary, one YAML file, no Kubernetes cluster.

> Huginn currently supports **Google Cloud Platform** only.



## Features

**Fleet management**
- Config-driven: describe your fleet in YAML, or let `huginn init` generate it
- Every game server instance is a Docker container
- Instance-level auto-scaling within your `min_instances` / `max_instances` and buffer settings
- Unhealthy instances (missed heartbeats) are detected and replaced
- Join codes and quick-join for clients

**Multi-host & cloud scaling (GCP)**
- Manages multiple machines at once, placing instances across hosts round-robin
- **Scale-up:** when every host is over the CPU/memory threshold, Huginn creates a new VM, installs Docker, sets up TLS, pulls your image, and adds it to the pool — fully unattended
- **Scale-down:** hosts idle (zero instances) for a configurable time are drained and deleted
- **Self-healing:** hosts deleted outside Huginn are detected and removed, whether mid-provisioning or already running, with a dashboard notification explaining why
- **Startup reconciliation:** after a crash or restart, Huginn re-adopts running containers and flags leftover VMs
- Firewall rules kept in sync with the ports actually in use
- The primary host is never scaled down; host auto-scaling is **off by default** so configuring GCP never causes surprise billing

**Dashboard**
- **Fleet:** live instances, player counts, capacity, addresses, join codes, live log streaming, per-instance player history, restart/stop, plus fleet-wide stats including peak concurrent players
- **Hosts:** per-host CPU and memory gauges, which instances are running where, the primary host clearly badged, and a "Create Host" button for provisioning on demand
- **Config:** most settings apply live, without a restart

**Security**
- Two separate tokens, not one: an **admin token** (full dashboard control — start/stop/restart instances, change config, create/delete hosts) and a **client token** (read-only — lets a game client discover an available server and nothing else). Ship only the client token to players; the admin token never leaves your infrastructure.

**Setup automation**
- `install.sh` checks prerequisites, offers to install Docker if missing, builds, and launches setup
- `huginn init` generates your config and both tokens, and can optionally:
  - enable the required GCP APIs, grant IAM roles, and register an SSH key for provisioning
  - generate and install a systemd service so Huginn survives crashes and reboots

Every automated step asks first, shows what it will change, and is safe to re-run.

---

## How it works

```mermaid
flowchart LR
    Client["Game client<br/>(HuginnClient.cs)"]
    subgraph Primary["Primary VM"]
        Core["Huginn core<br/>API + dashboard :8080<br/>heartbeats UDP :9000"]
        G1["Game server<br/>containers"]
    end
    subgraph Extra["Auto-provisioned VMs"]
        G2["Game server<br/>containers"]
    end
    GCP["GCP Compute API"]

    Client -- "1. ask for a server (client token)" --> Core
    Client -- "2. connect directly (UDP)" --> G1
    Client -. "or" .-> G2
    G1 -- heartbeats --> Core
    G2 -- heartbeats --> Core
    Core -- "Docker API over TLS :2376" --> G2
    Core -- "create / delete hosts" --> GCP
```

1. Game servers send heartbeats (player count, health) to Huginn.
2. A client asks Huginn's API — using its low-privilege client token — for an available server and gets back a real `ip:port`.
3. The client connects **directly** to that server — Huginn is never in the game traffic path (the same allocation pattern Agones uses).

---

## Quickstart

**Before you start**
- A GCP project with billing enabled
- A GCE VM to run Huginn, created with the **`cloud-platform` access scope**, with Docker, Go, and the gcloud CLI installed
- Your game server, integrated with the Huginn SDK, built as a Docker image and pushed to **Artifact Registry** in `us-central1`
  (new to Docker? see [DOCKER_IMAGE_GUIDE.md](DOCKER_IMAGE_GUIDE.md) for a copy-paste walkthrough)
- `gcloud auth login` as a project **owner** (needed only for the automated IAM setup)

**Install**
```bash
git clone <this-repo>
cd hugin
./install.sh
```

`install.sh` checks for Docker (offering to install it on Debian/Ubuntu if missing), builds Huginn, and runs `huginn init`. Answer the prompts; saying yes to the automated GCP setup and the systemd install means Huginn is running when `init` finishes.

**Open the dashboard** at `http://<vm-public-ip>:8080` and log in with the **admin `auth_token`** printed by `init` (also in `config.yaml`).

**Connect your game** — ship a `huginn.json` next to your client executable, using the **client token**, not the admin token:
```json
{
  "huginnBaseUrl": "http://<vm-public-ip>:8080",
  "huginnToken": "<client_auth_token from config.yaml>"
}
```
The client token can only ask "is a server available" — it cannot stop, restart, or reconfigure anything. Never ship `auth_token` to players.

➡️ **Full walkthrough, including manual alternatives for every automated step: [GET_STARTED.md](GET_STARTED.md)**

---

## Example config

```yaml
game: my-fps-game
image: us-central1-docker.pkg.dev/my-project/my-repo/my-server:latest
min_instances: 2
max_instances: 10
buffer_size: 3
max_players: 16
port: 7778
gcp_project: my-project
gcp_zone: us-central1-a
host_auto_scaling_enabled: false       # opt in from the dashboard when ready
host_scale_up_threshold_percent: 90
host_scale_down_idle_minutes: 10
auth_token: <generated by huginn init — admin, keep private>
client_auth_token: <generated by huginn init — safe to ship to game clients>
```

---

## Known limitations

- **You build and push your own Docker image** — Huginn does not package game builds.
- **`max_instances` is fleet-wide**, not per host.
- **New hosts can only pull from `us-central1` Artifact Registry.**
- **Huginn itself is a single point of failure.** Running game sessions keep going if it stops, but no new allocations or scaling happen until it's back. Acceptable at the target scale.
- **No rate limiting** on the client-facing discovery endpoints.
- **GCP only** — AWS and bare-metal are not supported.

## Out of scope

Kubernetes support, matchmaking, and billing/cost tooling.

---

## Status

Actively developed as a final-year Software Engineering project. Core fleet management, multi-host support, automatic host scaling in both directions, self-healing, scoped authentication, and setup automation are implemented and tested live on GCP.

# Huginn

A lightweight game server fleet manager for indie and small game studios — think **"Agones without Kubernetes."**

Huginn takes a YAML config describing your dedicated server, spins up instances as Docker containers, tracks their health and player counts via heartbeats, and auto-scales up or down within limits you set. It can also run across multiple machines and provision new ones automatically as load grows — no Kubernetes cluster, no manual server management, just Docker and a single binary.


## Quickstart

**Requirements:** Docker installed and running, and a dedicated-server Docker image for your game.

```bash
# Interactively generate a config
huginn init

# Start your fleet
huginn start config.yaml
```

`huginn init` will ask a few questions (game name, image, min/max instances, max players, port) and write a working `config.yaml` — no need to hand-write YAML or guess at field names.

## What it does

- **Config-driven provisioning** — describe your fleet in YAML, spin it up with one command
- **Docker-based** — each server instance is a container; manage anywhere from 1 to dozens per host
- **Live dashboard** — a built-in web UI showing:
  - Real-time fleet overview (instance count, total players, availability)
  - Per-instance details: address, join code, container ID, health state
  - **Live log streaming** straight from each container
  - **Player-count history**, as an interactive time-series chart (hover for exact values, scroll to zoom)
  - **Restart** and **stop** controls per instance
- **Auto-scaling** — spins up new instances as players fill existing servers, scales back down when idle, within your configured min/max
- **Multi-host, with automatic scale-out** — Huginn can manage more than one machine at once, routing new instances across hosts. When configured with cloud credentials, it monitors each host's real CPU/memory usage and automatically provisions a new host — TLS-secured and ready to run containers — when existing capacity is exhausted. Currently supports GCP; more providers planned.
- **Game-agnostic** — works with any dedicated server binary; integrate via a small SDK or by parsing your server's existing logs
- **Resilient by design** — if Huginn crashes or restarts, it reconciles with already-running containers across every host instead of colliding with them. Pairs well with a systemd unit (not included — see the deployment notes below) for automatic process recovery.

## Example config

```yaml
game: my-fps-game
image: my-server:latest
min_instances: 2
max_instances: 10
max_players: 16
port: 7778
```

(`huginn init` generates a complete, working version of this for you — the above is illustrative.)

### Multi-host & auto-scaling (optional)

To let Huginn manage more than one machine and provision new hosts automatically, add:

```yaml
gcp_project: your-gcp-project-id
gcp_zone: us-central1-a
host_scale_up_threshold_percent: 90
```

This requires one-time cloud setup (IAM roles, a container registry for your game server image, and a generated TLS certificate chain for secure host-to-host connections) — not yet automated by `huginn init`. Single-host use works out of the box with no cloud setup at all.

## Status

Huginn is under active development. v1's original scope — single-host provisioning, health tracking, auto-scaling, and an operable dashboard — is done. Since then, multi-host support and automatic host-level scaling (GCP) have landed as well. Not yet in scope: Kubernetes support, matchmaking, and billing/cost tooling. AWS support is planned but not yet implemented.

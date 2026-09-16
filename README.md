# Huginn

A lightweight game server fleet manager for indie and small multiplayer game studios — think **"Agones without Kubernetes."**

Huginn takes a YAML config describing your dedicated server, spins up instances as Docker containers, tracks their health and player counts via heartbeats, and auto-scales up or down within limits you set. No Kubernetes cluster, no cloud lock-in — just Docker and a single binary.

## Why

Small teams shipping a multiplayer game are usually stuck between two bad options: manually SSH-ing into a VPS to start/stop servers by hand, or adopting Agones — powerful, but built on Kubernetes, which is real operational overhead for a team without dedicated infra staff. Huginn aims for the middle ground: self-hosted, Docker-based, and simple enough to run on a single machine.

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
- **Docker-based** — each server instance is a container; manage anywhere from 1 to dozens on a single host
- **Live dashboard** — a built-in web UI showing:
  - Real-time fleet overview (instance count, total players, availability)
  - Per-instance details: address, join code, container ID, health state
  - **Live log streaming** straight from each container
  - **Player-count history**, as an interactive time-series chart (hover for exact values, scroll to zoom)
  - **Restart** and **stop** controls per instance
- **Auto-scaling** — spins up new instances as players fill existing servers, scales back down when idle, within your configured min/max
- **Game-agnostic** — works with any dedicated server binary; integrate via a small SDK or by parsing your server's existing logs
- **Resilient by design** — if Huginn crashes or restarts, it reconciles with already-running containers instead of colliding with them; pair with the included systemd unit for automatic recovery

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

## Status

Huginn is under active development. Current focus (v1) is a solid single-host experience: provisioning, health tracking, auto-scaling, and an operable dashboard. Not yet in scope: Kubernetes support, multi-node clusters, matchmaking, and billing/cost tooling — see the project's requirements doc for the full picture.


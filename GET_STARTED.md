# Getting Started with Huginn

This walks through everything needed to go from a fresh GCP project to a running, player-connectable game server fleet. Huginn currently supports **Google Cloud Platform only.**

Read this top to bottom the first time. After that, only [Step 4](#step-4-huginn-init) and later are things you'll do again.

---

## Before you start

You'll need:

- **A GCP project** with billing enabled.
- **A GCE VM to run Huginn on.** This machine runs Huginn itself and, by default, also runs game server instances (it's the "primary" host). Create it with:
  - The **`cloud-platform` access scope** — required. Huginn calls the Compute API using this VM's own credentials, and without this scope those calls are refused no matter what IAM roles you grant later. This can't be changed after the VM is created without stopping it, so get it right at creation time.
  - Debian or Ubuntu (Docker auto-install below assumes `apt`).
- **Docker, Go, and the gcloud CLI** on that VM. `install.sh` (below) can install Docker for you if it's missing; Go and gcloud need to be present already.
- **Your game server as a Docker image**, pushed to **Artifact Registry in `us-central1`**. New hosts Huginn provisions are currently only configured to pull from that region — using a different region will leave auto-provisioned hosts unable to pull your image.
- **`gcloud auth login`, done as a project owner** (a human account, not a service account) — only needed if you want Huginn's automated GCP setup (Step 5) to run. A service account usually can't grant IAM roles to itself.

---

## Step 1: Integrate the heartbeat contract

Before building your image, your game server needs to talk to Huginn. This is deliberately game-agnostic — Huginn doesn't care what engine or language your server is written in, as long as it does this:

- Send a periodic heartbeat reporting current player count and health.
- The bundled sidecar handles this for you in **log-parse mode**: it tails your server's log file and picks up any line containing `player_count=<number>` — so the only change your server needs is printing that one line whenever the player count changes.

---

## Step 2: Build and push your image

If you're not familiar with Docker, see **[DOCKER_IMAGE_GUIDE.md](DOCKER_IMAGE_GUIDE.md)** for a full copy-and-fill-in walkthrough — it covers the `Dockerfile` and `start.sh` needed to package your server together with Huginn's sidecar.

Once you have an image built:

```bash
gcloud artifacts repositories create <repo-name> \
  --repository-format=docker \
  --location=us-central1

docker build -t my-server:latest .
docker tag my-server:latest us-central1-docker.pkg.dev/<project-id>/<repo-name>/my-server:latest
docker push us-central1-docker.pkg.dev/<project-id>/<repo-name>/my-server:latest
```

Keep the full image path handy — `huginn init` will ask for it.

---

## Step 3: Get Huginn and run the installer

```bash
git clone <this-repo>
cd hugin
./install.sh
```

`install.sh`:

1. Checks for Docker. If it's missing and you're on a Debian/Ubuntu system, it offers to install it via Docker's official script and add your user to the `docker` group — this requires `sudo`, and it asks before doing anything. If it just installed Docker, it will tell you to log out and back in (group membership doesn't apply to your current shell) and re-run the script.
2. Checks Docker is actually running and that your user can reach it.
3. Checks for the `gcloud` CLI.
4. Builds the `huginn` binary.
5. Runs `huginn init`.

If Docker was already installed and running, all of that is silent and it goes straight to building.

---

## Step 4: `huginn init`

This is interactive. It asks for:

- **Game name, Docker image** (the full path from Step 2), **min/max instances, max players per instance, UDP port.**
- **GCP project ID and zone** — format-checked immediately (letters/digits/hyphens, valid zone shape). No GCP API calls happen yet at this point.

Then it offers **automated GCP setup**, off by default since it makes real changes to your project:

> "Run automated GCP setup now? (enables APIs, grants IAM roles, adds an SSH key — makes real changes to your project)"

If you say yes, it:

1. Confirms `gcloud` is authenticated. If it's a service account (not a human), it warns you — service accounts usually can't grant IAM roles — and asks whether to continue anyway.
2. Confirms the project ID is real and accessible.
3. Enables the Compute Engine and Artifact Registry APIs (free, no-op if already on).
4. Detects the service account Huginn will run as — the VM's own account if running on GCE, otherwise the project's default compute account.
5. Grants that account three roles: `roles/compute.instanceAdmin.v1`, `roles/artifactregistry.writer`, and `roles/iam.serviceAccountUser` (scoped to the account itself, not the whole project).
6. Generates a passphrase-less SSH key at `~/.ssh/huginn_automation_key` (or reuses one that's already there) — needed because Huginn provisions new hosts non-interactively, with no human around to type a passphrase.
7. Adds that key to your project's SSH metadata. **This step always shows you how many keys are already there and asks again before writing** — it only ever adds to the existing list, never replaces it.

If you say no to any of this, `init` tells you exactly what to run manually and keeps going — nothing else is blocked.

It then prints **two generated tokens** — read this carefully, it matters:

```
Two tokens were generated:
  auth_token         (admin — dashboard login, full control): <...>
  client_auth_token  (players — server discovery only):       <...>
Put ONLY client_auth_token in huginn.json for your game client. Never ship auth_token to players.
```

- **`auth_token`** logs into the dashboard and can stop/restart/reconfigure your whole fleet. Keep it private.
- **`client_auth_token`** can only ask "is a server available" — nothing else. This is the one your game ships to players.

Both are also saved in `config.yaml`, and `client_auth_token` is visible (with a copy button) on the dashboard's Config tab if you need to grab it again later.

Then it offers a **systemd service file**:

- "Generate a systemd service file for this fleet?" (yes by default) — writes a `huginn.service` file with correct absolute paths for wherever you're running from.
- "Install and enable it now via sudo?" (no by default) — if yes, copies it to `/etc/systemd/system/`, reloads systemd, and starts it. If a `huginn.service` is already installed, it warns you and asks before overwriting.

If you say yes to installing, **Huginn is already running by the time `init` finishes.** If you said no, start it yourself:

```bash
./huginn start config.yaml
```

---

## Step 5: Open the dashboard

```
http://<vm-public-ip>:8080
```

Log in with the **admin `auth_token`** from `config.yaml`.

- **Fleet** — live instances, player counts, capacity, join codes, live log streaming, per-instance player history, restart/stop. Summary cards at the top show total instances, current players, available slots, and peak concurrent players for the day.
- **Hosts** — every machine in the fleet (your primary VM, badged "Primary," plus any auto-provisioned GCP hosts), with live CPU/memory gauges and which instances are running on each. A "+ Create Host" button lets you manually provision an extra host on demand. If Huginn detects a host was removed outside its control, a notification explains why.
- **Config** — most settings apply live without a restart (the page tells you which ones need one). Your `client_auth_token` is shown here with a copy button.

### Turning on host auto-scaling (optional)

Off by default, so configuring GCP never causes surprise billing on its own. To enable it: go to Config, check "Enable automatic host scaling," and set a scale-down idle threshold (how long an empty host sits idle before Huginn deletes it — default 10 minutes).

Once on:

- Huginn creates a new host when **every** existing host is over the CPU/memory threshold (default 90%).
- A non-primary host with zero instances for longer than the idle threshold gets drained and deleted.
- If a host is deleted outside Huginn (manually, or some other event), Huginn detects this — whether it happens mid-provisioning or after the host was already active — and cleans up its own records automatically.
- Your primary VM is never auto-scaled down.

---

## Step 6: Connect your game client

Ship a `huginn.json` file next to your client's executable, using the **client token**, never the admin token:

```json
{
  "huginnBaseUrl": "http://<vm-public-ip>:8080",
  "huginnToken": "<client_auth_token from config.yaml>"
}
```

For Unity clients, `HuginnClient.cs` reads this on startup (alternatively via a `-huginn <url>` / `-huginntoken <token>` command-line flag, or Inspector fields) and exposes:

- `HuginnClient.Instance.FindServer(onSuccess, onError)` — quick-join, returns any server with free slots.
- `HuginnClient.Instance.FindServerByCode(code, onSuccess, onError)` — join a specific server by its join code (shown in the Fleet tab).

Both return a real `ip:port` address. Point your networking layer at it and call `StartClient()` — the client connects **directly** to the game server; Huginn is never in the traffic path.

The client token can only reach these two discovery endpoints — even if a player extracts it from your game's files, it grants no ability to touch your fleet.

---

## Day-to-day

Once running, Huginn handles automatically: instance scale-up/down within your min/max, replacing unhealthy instances, firewall port ranges, TLS between Huginn and remote hosts, and (if enabled) host-level scaling in both directions.

To ship a new game version: push the updated image to the same tag, then restart Huginn (`sudo systemctl restart huginn` if you installed the service).

---

## Known limitations

- You build and push your own Docker image — Huginn doesn't build it for you (though [DOCKER_IMAGE_GUIDE.md](DOCKER_IMAGE_GUIDE.md) makes this copy-paste simple even with no Docker experience).
- `max_instances` is a fleet-wide cap, not per-host.
- Auto-provisioned hosts can currently only pull from `us-central1` Artifact Registry.
- Huginn itself is a single point of failure — running sessions keep going if it goes down, but no new scaling or allocation happens until it's back.
- No rate limiting on the client-facing discovery endpoints.
- GCP only. No AWS, no bare-metal.

If something doesn't match what you see, or a step is unclear, that's a documentation bug — please flag it.

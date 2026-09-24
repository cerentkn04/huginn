# Getting Started with Huginn

This walks through everything needed to go from a fresh GCP project to a running, player-connectable game server fleet. Huginn currently supports **Google Cloud Platform only.**

Read this top to bottom the first time. After that, only [Step 4](#step-4-huginn-init) and later are things you'll do again.

---

## Before you start

You'll need:

- **A GCP project** with billing enabled.
- **A GCE VM to run Huginn on.** See [Creating the VM](#creating-the-vm) below if you've never done this — the access-scope setting there is the single most common thing to get wrong.
- **Docker, Go, and the gcloud CLI** on that VM. `install.sh` (below) can install Docker for you if it's missing; `gcloud` is preinstalled on GCP's own VM images. Go is only needed if you're building from source — skip it if you're using the [prebuilt release binary](https://github.com/cerentkn04/huginn/releases/latest).
- **Your game server as a Docker image**, pushed to **Artifact Registry in `us-central1`**. New hosts Huginn provisions are currently only configured to pull from that region — using a different region will leave auto-provisioned hosts unable to pull your image.
- **`gcloud auth login`, done as a project owner** (a human account, not a service account) — only needed if you want Huginn's automated GCP setup (Step 4) to run. A service account usually can't grant IAM roles to itself.

### Creating the VM

1. In the GCP Console, go to **Compute Engine → VM instances → Create Instance**.
2. Give it a name, pick a region/zone (`us-central1-a` matches this guide's examples, but any region works as long as you're consistent).
3. Machine type: `e2-small` or `e2-medium` is enough for testing.
4. Boot disk: **Debian or Ubuntu** — required, since `install.sh`'s Docker auto-install assumes `apt`.
5. **Expand "Identity and API access"** and set **Access scopes** to **"Allow full access to all Cloud APIs."** This is the `cloud-platform` scope. It's easy to miss since this section is usually collapsed by default.
   - **This cannot be added later without stopping the VM.** If you skip it, Huginn will start but every GCP-touching feature — firewall management, public IP discovery, host provisioning — will fail with a `403: Request had insufficient authentication scopes` error, no matter what IAM roles you grant afterward. See [Troubleshooting](#troubleshooting) if this happens to you.
6. Create the VM, then connect via its **SSH** button in the Console (no key setup needed).

---

## Step 1: Integrate the heartbeat contract

Before building your image, your game server needs to talk to Huginn. This is deliberately game-agnostic — Huginn doesn't care what engine or language your server is written in, as long as it does this:

- Send a periodic heartbeat reporting current player count and health.
- The bundled sidecar handles this for you in **log-parse mode**: it tails your server's log file and picks up any line containing `player_count=<number>` — so the only change your server needs is printing that one line whenever the player count changes.

---

## Step 2: Build and push your image

If you're not familiar with Docker, see **[DOCKER_IMAGE_GUIDE.md](DOCKER_IMAGE_GUIDE.md)** for a full copy-and-fill-in walkthrough — it covers the `Dockerfile` and `start.sh` needed to package your server together with Huginn's sidecar, using only prebuilt binaries if you don't have Go installed.

Once you have an image built:

```bash
gcloud artifacts repositories create <repo-name> \
  --repository-format=docker \
  --location=us-central1

gcloud auth configure-docker us-central1-docker.pkg.dev

docker build -t my-server:latest .
docker tag my-server:latest us-central1-docker.pkg.dev/<project-id>/<repo-name>/my-server:latest
docker push us-central1-docker.pkg.dev/<project-id>/<repo-name>/my-server:latest
```

`gcloud auth configure-docker` is a one-time step per machine — without it, `docker push` fails with an authentication error.

**Finding your project ID:** click the project selector at the top of the GCP Console, or run `gcloud config get-value project`. This is different from the project's display name (what you typed when creating it) and different from the Artifact Registry repo name you just created — all three can look similar but are three separate values.

Keep the full `docker tag` path handy — `huginn init` will ask for it as your Docker image.

---

## Step 3: Get Huginn and run the installer

**Option A — build from source:**
```bash
git clone <this-repo>
cd hugin
./install.sh
```

**Option B — use the prebuilt binary (no Go needed):**
```bash
curl -L -o install.sh https://github.com/cerentkn04/huginn/releases/latest/download/install.sh
chmod +x install.sh
./install.sh
```
`install.sh` downloads the release binary automatically if no `huginn` binary exists in the current directory and Go isn't installed.

Either way, `install.sh`:

1. Checks for Docker. If it's missing and you're on a Debian/Ubuntu system, it offers to install it via Docker's official script and add your user to the `docker` group — this requires `sudo`, and it asks before doing anything. If it just installed Docker, it will tell you to log out and back in (group membership doesn't apply to your current shell) and re-run the script.
2. Checks Docker is actually running and that your user can reach it.
3. Checks for the `gcloud` CLI.
4. Gets the `huginn` binary (builds from source, or downloads the release, whichever applies).
5. Runs `huginn init`.

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
5. Grants that account IAM roles for provisioning and image access.
6. Generates a passphrase-less SSH key at `~/.ssh/huginn_automation_key` (or reuses one that's already there) — needed because Huginn provisions new hosts non-interactively, with no human around to type a passphrase.
7. Adds that key to your project's SSH metadata. **This step always shows you how many keys are already there and asks again before writing** — it only ever adds to the existing list, never replaces it.

> **Known gap:** as of this writing, the automated setup does not yet grant `roles/compute.securityAdmin`, which Huginn needs to manage firewall rules. If you see `firewall: missing permission to manage GCP firewall rules` in the logs after setup, run the command Huginn prints in that log line (it's a one-line `gcloud projects add-iam-policy-binding` with the exact role), then restart Huginn. This will be automated in a future version.

If you say no to any of this, `init` tells you exactly what to run manually and keeps going — nothing else is blocked.

It then prints **two generated tokens** — read this carefully, it matters:Two tokens were generated:
auth_token (admin — dashboard login, full control): <...>
client_auth_token (players — server discovery only): <...>
Put ONLY client_auth_token in huginn.json for your game client. Never ship auth_token to players.
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

## Step 5: Open the dashboardhttp://<vm-public-ip>:8080
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

## Troubleshooting

**Dashboard times out / can't be reached, and logs show `403: Request had insufficient authentication scopes`.**
Your VM was created without the `cloud-platform` access scope. Confirm with:
```bash
curl -s -H "Metadata-Flavor: Google" \
  "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/scopes"
```
If `https://www.googleapis.com/auth/cloud-platform` is missing from the list, stop the VM and either edit it in the Console (Access scopes → "Allow full access to all Cloud APIs") or run:
```bash
gcloud compute instances stop <vm-name> --zone=<zone>
gcloud compute instances set-service-account <vm-name> --zone=<zone> --scopes=cloud-platform
gcloud compute instances start <vm-name> --zone=<zone>
```
The VM's external IP may change after restarting — check it again before retrying the dashboard.

**`firewall: missing permission to manage GCP firewall rules` keeps appearing in the logs.**
See the note in [Step 4](#step-4-huginn-init) — the automated setup currently misses one IAM role. Run the `gcloud projects add-iam-policy-binding` command Huginn prints in that log line, then restart.

**"Editing VM instance failed: Supplied fingerprint does not match current metadata fingerprint."**
A stale GCP Console cache, not a real error. Reload the VM's edit page and try again.

---

## Known limitations

- You build and push your own Docker image — Huginn doesn't build it for you (though [DOCKER_IMAGE_GUIDE.md](DOCKER_IMAGE_GUIDE.md) makes this copy-paste simple even with no Docker experience).
- `max_instances` is a fleet-wide cap, not per-host.
- Auto-provisioned hosts can currently only pull from `us-central1` Artifact Registry.
- Huginn itself is a single point of failure — running sessions keep going if it goes down, but no new scaling or allocation happens until it's back.
- No rate limiting on the client-facing discovery endpoints.
- The automated GCP setup doesn't yet grant `roles/compute.securityAdmin` (see Step 4).
- GCP only. No AWS, no bare-metal.

If something doesn't match what you see, or a step is unclear, that's a documentation bug — please flag it.

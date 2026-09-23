# Building Your Game Server Image

You don't need to understand Docker to do this — just copy the template below, change a few things, and run three commands.

## What you need first

- Your dedicated server build, exported for **Linux** (in Unity: `File → Build Settings → Linux → Dedicated Server`).
- Docker installed on the machine you're building from (this can be your own laptop, or the same VM running Huginn — either works).

## 1. Make your server report its player count

Huginn's sidecar watches your server's log output for a line containing the exact text:

```
player_count=<number>
```

anywhere in that line — for example, a log line like `[12:03:01] status: player_count=3 tick=884` works fine. It can be surrounded by other text; the sidecar just looks for that pattern and reads the number.

Add one line to your server that prints this whenever the player count changes (or just periodically, e.g. once a second):

```csharp
Debug.Log($"player_count={currentPlayerCount}");
```

That's the only code change your server needs. Everything else — heartbeats, health status, talking to Huginn — is handled by the sidecar, which you're about to add to your image.

## 2. Set up your folder

Put your exported Linux server build in a folder by itself. You should see your server's executable in there (for a Unity build, something like `YourGame.x86_64`) plus its supporting `_Data` folder.

In that same folder, create two new files: `Dockerfile` and `start.sh`.

## 3. `start.sh` — starts your server and the sidecar together

```bash
#!/bin/sh
/app/YourGame.x86_64 -batchmode -nographics -logFile /app/server.log &
export SIDECAR_LOG_PATH=/app/server.log
exec /app/sidecar
```

Change only `YourGame.x86_64` to your actual executable's filename. Nothing else in this file needs to change.

This starts your game server in the background, tells it to write its log to `/app/server.log`, then starts the sidecar pointed at that same file — which is where it reads the `player_count=` lines from Step 1.

## 4. `Dockerfile` — packages your server into an image

```dockerfile
FROM golang:1.25 AS sidecar-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /sidecar ./cmd/sidecar

FROM unitymultiplay/linux-base-image:1.0.1
USER root
WORKDIR /app
COPY . /app
RUN chmod +x /app/YourGame.x86_64
RUN chmod +x /app/start.sh

COPY --from=sidecar-builder /sidecar /app/sidecar
RUN chmod +x /app/sidecar

ENTRYPOINT ["/app/start.sh"]
EXPOSE 7778/udp
```

`unitymultiplay/linux-base-image` is the base image Unity provides specifically for Linux dedicated servers — use it if you're building a Unity server, as it includes runtime libraries a bare OS image doesn't have. If you're building for a different engine, replace it with a base image appropriate for your server's dependencies (or a plain `ubuntu:22.04` if your build has no special requirements).

Three things to change:
1. `YourGame.x86_64` (both places) — your actual executable filename, matching what you put in `start.sh`.
2. `EXPOSE 7778/udp` — your game's actual port. Match whatever `port:` is set to in Huginn's config.
3. The first block (`FROM golang:1.25 AS sidecar-builder` through the `go build` line) builds Huginn's sidecar directly from Huginn's own source — this works as long as `cmd/`, `internal/`, `go.mod`, and `go.sum` from the Huginn repo are available in the same folder you're building from. If you were given a pre-built `sidecar` binary instead, delete this whole block and replace `COPY --from=sidecar-builder /sidecar /app/sidecar` with `COPY sidecar /app/sidecar` (with that binary sitting in your folder alongside everything else).

## 5. Build, tag, and push

Run these three commands from inside the folder with your `Dockerfile`, one at a time, replacing the placeholders in `<angle brackets>`:

```bash
docker build -t my-server:latest .

docker tag my-server:latest us-central1-docker.pkg.dev/<your-gcp-project-id>/<your-repo-name>/my-server:latest

docker push us-central1-docker.pkg.dev/<your-gcp-project-id>/<your-repo-name>/my-server:latest
```

(If you haven't created the repository yet, run this once first: `gcloud artifacts repositories create <your-repo-name> --repository-format=docker --location=us-central1`.)

Copy the full path from the `docker tag` line — that's the exact string `huginn init` will ask you for as your Docker image.

That's it. The only thing you had to write yourself was one log line; everything else is copy-and-fill-in.

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
COPY serverBuild/ /app
RUN chmod +x /app/Mon.x86_64
RUN chmod +x /app/start.sh

COPY --from=sidecar-builder /sidecar /app/sidecar
RUN chmod +x /app/sidecar

ENTRYPOINT ["/app/start.sh"]
EXPOSE 7778/udp

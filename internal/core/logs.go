package core

import (
	"io"
	"net/http"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/pkg/stdcopy"
)

func LogFleetStream(mux *http.ServeMux, reg *Registry,hostPool *HostPool ) {
	mux.HandleFunc("/api/servers/logs/", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Path[len("/api/servers/logs/"):]
		if code == "" {
			http.Error(w, "missing logs", http.StatusBadRequest)
			return
		}
		inst, ok := reg.GetByCode(code)
		if !ok {
			http.Error(w, "server not found", http.StatusNotFound)
			return
		}
		cli, err := hostPool.Get(inst.HostID)
		if err != nil {
			http.Error(w, "host unavailable", http.StatusInternalServerError)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}

		resp, err := cli.ContainerLogs(r.Context(), inst.ContainerID, types.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if err != nil {
			http.Error(w, "failed to get logs", http.StatusInternalServerError)
			return
		}
		defer resp.Close()

		fw := flushWriter{w: w, flusher: flusher}
		if _, err := stdcopy.StdCopy(fw, fw, resp); err != nil {
			log.Printf("huginn: logs: stream ended for %s: %v", inst.ID, err)
		}
	})
}

type flushWriter struct {
	w       io.Writer
	flusher http.Flusher
}

func (fw flushWriter) Write(p []byte) (int, error) {
	n, err := fw.w.Write(p)
	fw.flusher.Flush()
	return n, err
}

import { useState, useEffect, useRef } from "react";
import uPlot from "uplot";
import "uplot/dist/uPlot.min.css";
import { authFetch } from "../auth";



function HistoryChart({ data, maxPlayers }) {
  const containerRef = useRef(null);
  const plotRef = useRef(null);

  useEffect(() => {
    if (!containerRef.current) return;
    if (!data || data.length < 2) return;

    const times = data.map((s) => s.t);
    const counts = data.map((s) => s.c);
    const peak = Math.max(maxPlayers || 1, ...counts);

    if (plotRef.current) {
      plotRef.current.setData([times, counts]);
      plotRef.current.setScale("y", { min: 0, max: peak });
      return;
    }

    const rect = containerRef.current.getBoundingClientRect();

    const opts = {
      width: rect.width,
      height: 220,
      scales: {
        x: { time: true },
        y: { range: () => [0, peak] },
      },
      series: [
        {},
        {
          label: "Players",
          stroke: "#4ade80",
          width: 2,
          fill: "rgba(74, 222, 128, 0.12)",
        },
      ],
      axes: [
        { stroke: "#6b7280", grid: { stroke: "#1f2937", width: 1 }, font: "11px monospace" },
        { stroke: "#6b7280", grid: { stroke: "#1f2937", width: 1 }, font: "11px monospace" },
      ],
      hooks: {
        draw: [
          (u) => {
            const { ctx } = u;
            const y = u.valToPos(maxPlayers, "y", true);
            ctx.save();
            ctx.strokeStyle = "rgba(248, 113, 113, 0.45)";
            ctx.lineWidth = 1;
            ctx.setLineDash([4, 4]);
            ctx.beginPath();
            ctx.moveTo(u.bbox.left, y);
            ctx.lineTo(u.bbox.left + u.bbox.width, y);
            ctx.stroke();
            ctx.restore();
          },
        ],
      },
    };

    plotRef.current = new uPlot(opts, [times, counts], containerRef.current);

    const resizeObserver = new ResizeObserver(() => {
      if (!containerRef.current || !plotRef.current) return;
      plotRef.current.setSize({
        width: containerRef.current.getBoundingClientRect().width,
        height: 220,
      });
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      plotRef.current?.destroy();
      plotRef.current = null;
    };
  }, [data, maxPlayers]);

  if (!data || data.length < 2) {
    return <div style={styles.sparklineEmpty}>Not enough data yet</div>;
  }
  return <div ref={containerRef} />;
}



const stateColors = {
  ready: "#4ade80",
  starting: "#facc15",
  draining: "#fb923c",
  unhealthy: "#f87171",
};

function timeAgo(isoString) {
  const seconds = Math.floor((Date.now() - new Date(isoString)) / 1000);
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.floor(seconds / 60);
  return `${minutes}m ago`;
}

function formatHeartbeat(isoString) {
  const d = new Date(isoString);
  const clock = d.toLocaleTimeString([], { hour12: false });
  return `${timeAgo(isoString)} · ${clock}`;
}

function Field({ label, value, onCopy }) {
  return (
    <div style={styles.field}>
      <div style={styles.fieldLabel}>{label}</div>
      <div style={styles.fieldValueRow}>
        <span style={styles.fieldValue}>{value}</span>
        {onCopy && (
          <button style={styles.copyButton} onClick={onCopy}>
            Copy
          </button>
        )}
      </div>
    </div>
  );
}

export default function Fleet({ instances, loaded,  peakToday,  hostFilter, ostFilter, onClearFilter }) {
  const [selectedId, setSelectedId] = useState(null);
  const [confirmingStop, setConfirmingStop] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [stopError, setStopError] = useState(null);
  const [detailTab, setDetailTab] = useState("details");
  const [logs, setLogs] = useState("");
const [restarting, setRestarting] = useState(false);
const [restartError, setRestartError] = useState(null); 
  const totalInstances = instances.length;
  const totalPlayers = instances.reduce((sum, i) => sum + i.PlayerCount, 0);
  const available = instances.filter(
    (i) => i.State === "ready" && i.PlayerCount < i.MaxPlayers
  ).length;

  const selected = instances.find((i) => i.ID === selectedId) || null;  // ← moved up
    const displayedInstances = hostFilter
    ? instances.filter((i) => i.HostID === hostFilter)
    : instances;
    useEffect(() => {
    if (!selected) return;
    

    setLogs("");
    const controller = new AbortController();

      authFetch(`/api/servers/logs/${selected.JoinCode}`, { signal: controller.signal })
      .then((res) => {
        const reader = res.body.getReader();
        const decoder = new TextDecoder();
        function read() {
          reader.read()
            .then(({ done, value }) => {
              if (done) return;
              setLogs((prev) => prev + decoder.decode(value));
              read();
            })
            .catch(() => {});
        }
        read();
      })
      .catch(() => {});

    return () => controller.abort();
  },  [selected?.ID, detailTab]);

  useEffect(() => {
    setConfirmingStop(false);
    setStopError(null);
     setDetailTab("details");
     
  }, [selectedId]);

  const handleStop = async (id) => {
    setStopping(true);
    setStopError(null);
    try {
      const res = await   await authFetch(`/api/servers/stop/${id}`, { method: "POST" });
	if (!res.ok) {
        	const text = await res.text();
        	throw new Error(text.trim() || `HTTP ${res.status}`);
      }
      setSelectedId(null);
      setConfirmingStop(false);
    } catch (err) {
      setStopError(err.message);
    } finally {
      setStopping(false);
    }
  };
const handleRestart = async (id) => {
  setRestarting(true);
  setRestartError(null);
  try {
    const res = await authFetch(`/api/servers/restart/${id}`, { method: "POST" });
    if (!res.ok) {
      const text = await res.text();
      throw new Error(text.trim() || `HTTP ${res.status}`);
    }
  } catch (err) {
    setRestartError(err.message);
  } finally {
    setRestarting(false);
  }
};
  return (
    <div style={styles.container}>
<div style={styles.summaryRow}>
  <div style={{ ...styles.statCard, borderLeftColor: "#60a5fa" }}>
    <div style={styles.statValue}>{loaded ? totalInstances : "—"}</div>
    <div style={styles.statLabel}>Instances</div>
  </div>
  <div style={{ ...styles.statCard, borderLeftColor: "#4ade80" }}>
    <div style={{ ...styles.statValue, color: "#4ade80" }}>{loaded ? totalPlayers : "—"}</div>
    <div style={styles.statLabel}>Players</div>
  </div>
  <div style={{ ...styles.statCard, borderLeftColor: "#facc15" }}>
    <div style={styles.statValue}>{loaded ? available : "—"}</div>
    <div style={styles.statLabel}>Available</div>
  </div>
	    <div style={{ ...styles.statCard, borderLeftColor: "#c084fc" }}>
    <div style={styles.statValue}>{loaded ? peakToday : "—"}</div>
    <div style={styles.statLabelMuted}>Peak Today</div>
  </div>
</div>

{loaded && totalInstances > 0 && (
  <div style={styles.statePillsRow}>
    {Object.entries(stateColors).map(([state, color]) => {
      const count = instances.filter((i) => i.State === state).length;
      if (count === 0) return null;
      return (
        <div key={state} style={styles.statePill}>
          <span style={{ ...styles.dot, background: color }} />
          {count} {state}
        </div>
      );
    })}
  </div>
)}
      <div style={styles.splitView}>
        <div style={styles.sidebar}>
          {!loaded ? (
            <div style={styles.sidebarNote}>Loading fleet…</div>
          ) : displayedInstances.length === 0 ? (
            <div style={styles.sidebarNote}>
              {hostFilter ? `No instances on ${hostFilter}` : "No instances running"}
            </div>
          ) : (
            displayedInstances.map((inst) => (
              <div
                key={inst.ID}
                style={{
                  ...styles.row,
                  background: inst.ID === selectedId ? "#1f2937" : "#161b22",
                }}
                onClick={() => setSelectedId(inst.ID)}
              >
                <span style={{ ...styles.dot, background: stateColors[inst.State] }} />
                <span style={styles.id}>{inst.ID}</span>
                <span style={styles.players}>
                  {inst.PlayerCount}/{inst.MaxPlayers}
                </span>
                <span style={styles.heartbeat}>{timeAgo(inst.LastHeartbeat)}</span>
              </div>
            ))
          )}
        </div>

        <div style={styles.detailPane}>
	  {hostFilter && (
  <div style={styles.filterBanner}>
    Filtered by host: <strong>{hostFilter}</strong>
    <button style={styles.clearFilterButton} onClick={onClearFilter}>Clear</button>
  </div>
)}
          {!selected ? (
  <div style={styles.placeholder}>
    {!loaded
      ? "Connecting…"
      : displayedInstances.length === 0
      ? hostFilter
        ? `No instances running on ${hostFilter}.`
        : "No instances running yet. Check your config's min_instances, or wait for the fleet to start."
      : "Select an instance to view details"}
  </div>		  
          ) : (
            <div>
              <div style={styles.detailHeader}>
                <h3 style={styles.detailTitle}>{selected.ID}</h3>
                <span
                  style={{
                    ...styles.badge,
                    background: stateColors[selected.State],
                  }}
                >
                  {selected.State}
                </span>
              </div>
              <div style={styles.tabRow}>
                <button
                  style={{ ...styles.tabButton, ...(detailTab === "details" ? styles.tabButtonActive : {}) }}
                  onClick={() => setDetailTab("details")}
                >
                  Details
                </button>
                <button
                  style={{ ...styles.tabButton, ...(detailTab === "logs" ? styles.tabButtonActive : {}) }}
                  onClick={() => setDetailTab("logs")}
                >
                  Logs
                </button>
             </div>

              {detailTab === "details" ? (
                <>
                  <div style={styles.capacityBlock}>
                    <div style={styles.capacityLabel}>
                      {selected.PlayerCount} of {selected.MaxPlayers} slots filled
                    </div>
                    <div style={styles.barTrack}>
                      <div
                        style={{
                          ...styles.barFill,
                          width: `${(selected.PlayerCount / selected.MaxPlayers) * 100}%`,
                        }}
                      />
                    </div>
                  </div>
                  <div style={styles.fieldGrid}>
                    <Field label="Container" value={selected.ContainerID.slice(0, 12)} />
                    <Field
                      label="Address"
                      value={selected.Address}
                      onCopy={() => navigator.clipboard.writeText(selected.Address)}
                    />
                    <Field
                      label="Join Code"
                      value={selected.JoinCode}
                      onCopy={() => navigator.clipboard.writeText(selected.JoinCode)}
                    />
                    <Field label="Last Heartbeat" value={formatHeartbeat(selected.LastHeartbeat)} />
                  </div>
                </>
              ) : (
                <pre style={styles.logPane}>{logs}</pre>
              )}

              {stopError && <div style={styles.error}>{stopError}</div>}
              {restartError && <div style={styles.error}>{restartError}</div>}

              {!confirmingStop ? (

                <div style={{ display: "flex", gap: "8px", marginTop: "16px" }}>
                  <button
                    style={{
                      ...styles.button,
                      ...styles.secondaryButton,
                      ...(restarting ? styles.buttonDisabled : {}),
                    }}
                    disabled={restarting}
                    onClick={() => handleRestart(selected.ID)}
                  >
                    {restarting ? "Restarting…" : "Restart Instance"}
                  </button>
                  <button
                    style={{ ...styles.button, ...styles.dangerButton }}
                    onClick={() => setConfirmingStop(true)}
                  >
                    Stop Instance
                  </button>
                </div>
                              
                              
              ) : (
                <div style={styles.confirmBox}>
                  <div style={styles.confirmText}>
                    Stop <strong>{selected.ID}</strong>?
                    {selected.PlayerCount > 0 &&
                      ` ${selected.PlayerCount} player(s) will be disconnected.`}
                  </div>
                  <div style={styles.confirmActions}>
                    <button
                      style={{
                        ...styles.button,
                        ...styles.dangerButton,
                        ...(stopping ? styles.buttonDisabled : {}),
                      }}
                      disabled={stopping}
                      onClick={() => handleStop(selected.ID)}
                    >
                      {stopping ? "Stopping…" : "Yes, stop"}
                    </button>
                    <button
                      style={{
                        ...styles.button,
                        ...styles.secondaryButton,
                        ...(stopping ? styles.buttonDisabled : {}),
                      }}
                      disabled={stopping}
                      onClick={() => setConfirmingStop(false)}
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
{selected && (
  <div style={styles.historyPane}>
    <h4 style={styles.historyTitle}>Player History</h4>
<HistoryChart data={selected.PlayerHistory} maxPlayers={selected.MaxPlayers} />	
  </div>
)}
      </div>
    </div>
  );
}

const styles = {
container: { padding: "24px" },
  summaryRow: { display: "flex", gap: "16px", marginBottom: "24px", maxWidth: "900px" },
	statCard: {
  background: "#161b22",
  minWidth: "160px",
  border: "1px solid #21262d",
  color: "#fff",
  padding: "16px 24px",
  borderRadius: "8px",
  textAlign: "center",
  borderLeft: "3px solid #30363d",
},
    statValue: { fontSize: "24px", fontWeight: "bold" },
  statLabel: { fontSize: "13px", color: "#9ca3af" },
  statLabelMuted: { fontSize: "12px", color: "#6b7280", fontStyle: "italic" },
splitView: { display: "flex", gap: "16px", alignItems: "flex-start" },

sidebar: {
  width: "260px",
  flexShrink: 0,
  display: "flex",
  flexDirection: "column",
  gap: "6px",
},
  sidebarNote: {
    color: "#6b7280",
    fontSize: "13px",
    fontStyle: "italic",
    padding: "12px 14px",
  },
  row: {
    display: "flex",
    alignItems: "center",
    gap: "10px",
    padding: "12px 14px",
    borderRadius: "8px",
    cursor: "pointer",
    color: "#fff",
  },
  dot: { width: "10px", height: "10px", borderRadius: "50%", flexShrink: 0 },
  id: { fontWeight: "bold", marginRight: "auto", fontSize: "13px" },
  players: { color: "#ccc", fontSize: "12px" },
  heartbeat: { color: "#9ca3af", fontSize: "11px" },
detailPane: {
  flex: "1.3 1 0",
  background: "#161b22",
  borderRadius: "8px",
  padding: "24px",
  color: "#fff",
  minHeight: "300px",
},
   detailHeader: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    gap: "12px",
  },
  detailTitle: { marginTop: 0 },
  badge: {
    padding: "2px 10px",
    borderRadius: "999px",
    fontSize: "12px",
    fontWeight: "bold",
    color: "#0d1117",
  },
  tabRow: { display: "flex", gap: "4px", marginTop: "16px", borderBottom: "1px solid #30363d" },
tabButton: {
  background: "transparent",
  border: "none",
  color: "#9ca3af",
  padding: "8px 14px",
  cursor: "pointer",
  fontSize: "13px",
  fontFamily: "inherit",
  borderBottom: "2px solid transparent",
},
tabButtonActive: { color: "#fff", borderBottom: "2px solid #4ade80" },
  capacityBlock: { margin: "16px 0" },
  capacityLabel: { fontSize: "13px", color: "#9ca3af", marginBottom: "4px" },
  barTrack: {
    width: "100%",
    height: "8px",
    background: "#0d1117",
    borderRadius: "4px",
    overflow: "hidden",
  },
  barFill: { height: "100%", background: "#4ade80" },

  fieldGrid: {
    display: "grid",
    gridTemplateColumns: "1fr 1fr",
    gap: "16px",
    marginTop: "16px",
  },
  field: {},
  fieldLabel: {
    fontSize: "11px",
    textTransform: "uppercase",
    letterSpacing: "0.05em",
    color: "#6b7280",
    marginBottom: "2px",
  },
	historyPane: {
  flex: "1.7 1 0",
  minWidth: "320px",
  background: "#161b22",
  borderRadius: "8px",
  padding: "24px",
  color: "#fff",
  minHeight: "300px",
},
	statePillsRow: {
  display: "flex",
  gap: "10px",
  marginBottom: "24px",
  marginTop: "-12px",
},
statePill: {
  display: "flex",
  alignItems: "center",
  gap: "6px",
  background: "#161b22",
  border: "1px solid #30363d",
  borderRadius: "999px",
  padding: "4px 12px",
  fontSize: "12px",
  color: "#9ca3af",
},
  	filterBanner: {
  display: "flex",
  alignItems: "center",
  gap: "10px",
  background: "#161b22",
  border: "1px solid #30363d",
  borderRadius: "6px",
  padding: "8px 14px",
  marginBottom: "16px",
  color: "#9ca3af",
  fontSize: "13px",
},
clearFilterButton: {
  background: "#374151",
  color: "#e5e7eb",
  border: "none",
  padding: "4px 10px",
  borderRadius: "4px",
  cursor: "pointer",
  fontSize: "12px",
},
  historyTitle: { marginTop: 0, marginBottom: "16px", fontSize: "14px", color: "#9ca3af" },
  sparklineEmpty: { color: "#6b7280", fontStyle: "italic", marginTop: "16px", fontSize: "13px" },
  fieldValueRow: { display: "flex", alignItems: "center", gap: "8px" },
  fieldValue: { color: "#fff", fontSize: "14px" },
  copyButton: {
    background: "#1f2937",
    color: "#ccc",
    border: "none",
    padding: "2px 8px",
    borderRadius: "4px",
    fontSize: "11px",
    cursor: "pointer",
    fontFamily: "inherit",
  },

  button: {
    border: "none",
    padding: "8px 16px",
    borderRadius: "6px",
    cursor: "pointer",
    fontSize: "13px",
    fontWeight: 500,
    fontFamily: "inherit",
    lineHeight: 1.2,
  },
  dangerButton: { background: "#dc2626", color: "#fff" },
  secondaryButton: { background: "#374151", color: "#e5e7eb" },
  buttonDisabled: { opacity: 0.5, cursor: "not-allowed" },

  confirmBox: {
    marginTop: "20px",
    padding: "14px 16px",
    background: "rgba(220, 38, 38, 0.08)",
    border: "1px solid rgba(220, 38, 38, 0.3)",
    borderRadius: "8px",
    maxWidth: "420px",
  },
  confirmText: {
    fontSize: "13px",
    color: "#fbbf24",
    marginBottom: "12px",
    lineHeight: 1.5,
  },
  confirmActions: { display: "flex", gap: "8px" },
logPane: {
  marginTop: "16px",
  background: "#0a0d12",
  border: "1px solid #30363d",
  borderRadius: "6px",
  padding: "14px 16px",
  maxHeight: "360px",
  overflowY: "auto",
  fontSize: "12.5px",
  lineHeight: "1.6",
  fontFamily: "'SF Mono', Monaco, 'Cascadia Code', Consolas, monospace",
  color: "#8b949e",
  whiteSpace: "pre-wrap",
  wordBreak: "break-word",
},
  error: {
    display: "inline-block",
    color: "#fca5a5",
    background: "rgba(220, 38, 38, 0.12)",
    border: "1px solid rgba(220, 38, 38, 0.35)",
    padding: "6px 12px",
    borderRadius: "6px",
    marginTop: "16px",
    fontSize: "13px",
  },

  placeholder: { color: "#6b7280", fontStyle: "italic" },
};

import { useState, useEffect } from "react";

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

export default function Fleet({ instances, loaded }) {
  const [selectedId, setSelectedId] = useState(null);
  const [confirmingStop, setConfirmingStop] = useState(false);
  const [stopping, setStopping] = useState(false);
  const [stopError, setStopError] = useState(null);

  const totalInstances = instances.length;
  const totalPlayers = instances.reduce((sum, i) => sum + i.PlayerCount, 0);
  const available = instances.filter(
    (i) => i.State === "ready" && i.PlayerCount < i.MaxPlayers
  ).length;

  useEffect(() => {
    setConfirmingStop(false);
    setStopError(null);
  }, [selectedId]);

  const selected = instances.find((i) => i.ID === selectedId) || null;

  const handleStop = async (id) => {
    setStopping(true);
    setStopError(null);
    try {
      const res = await fetch(`/api/servers/stop/${id}`, { method: "POST" });
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

  return (
    <div style={styles.container}>
      <div style={styles.summaryRow}>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{loaded ? totalInstances : "—"}</div>
          <div style={styles.statLabel}>Instances</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{loaded ? totalPlayers : "—"}</div>
          <div style={styles.statLabel}>Players</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{loaded ? available : "—"}</div>
          <div style={styles.statLabel}>Available</div>
        </div>
      </div>

      <div style={styles.splitView}>
        <div style={styles.sidebar}>
          {!loaded ? (
            <div style={styles.sidebarNote}>Loading fleet…</div>
          ) : instances.length === 0 ? (
            <div style={styles.sidebarNote}>No instances running</div>
          ) : (
            instances.map((inst) => (
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
          {!selected ? (
            <div style={styles.placeholder}>Select an instance to view details</div>
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

              <div style={styles.capacityBlock}>
                <div style={styles.capacityLabel}>
                  Capacity: {selected.PlayerCount}/{selected.MaxPlayers}
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
                <Field label="Last Heartbeat" value={selected.LastHeartbeat} />
              </div>

              {stopError && <div style={styles.error}>{stopError}</div>}

              {!confirmingStop ? (
                <button
                  style={{ ...styles.button, ...styles.dangerButton, marginTop: "16px" }}
                  onClick={() => setConfirmingStop(true)}
                >
                  Stop Instance
                </button>
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
      </div>
    </div>
  );
}

const styles = {
  container: { padding: "24px" },
  summaryRow: { display: "flex", gap: "16px", marginBottom: "24px" },
  statCard: {
    background: "#161b22",
    color: "#fff",
    padding: "16px 24px",
    borderRadius: "8px",
    textAlign: "center",
  },
  statValue: { fontSize: "24px", fontWeight: "bold" },
  statLabel: { fontSize: "13px", color: "#9ca3af" },

  splitView: { display: "flex", gap: "16px" },

  sidebar: {
    width: "280px",
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
    flex: 1,
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
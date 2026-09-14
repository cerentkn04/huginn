// src/components/Fleet.jsx
import { useState } from "react";

// Mock data — matches the real Instance JSON shape from the Go backend.
// Replace this with live data on Day 3.
const mockInstances = [
  {
    ID: "huginn-inst-0",
    ContainerID: "5a3c1b839b997b3ed5641fe873db2464bf130ce61aa44d75f052a5673809cf46",
    State: "ready",
    PlayerCount: 3,
    MaxPlayers: 5,
    LastHeartbeat: "2026-09-11T09:14:52Z",
    JoinCode: "751f",
    Address: "34.30.42.253:7778",
  },
  {
    ID: "huginn-inst-1",
    ContainerID: "7aee53e5af52a3a5aefff59776f6cd3cd394c649218907f09d8f6116a75ef7ed",
    State: "starting",
    PlayerCount: 0,
    MaxPlayers: 5,
    LastHeartbeat: "2026-09-11T09:14:50Z",
    JoinCode: "590f",
    Address: "34.30.42.253:7779",
  },
  {
    ID: "huginn-inst-2",
    ContainerID: "d6525e1805b89fdd93f2b12345abcd6789ef0123456789abcdef0123456789a",
    State: "draining",
    PlayerCount: 0,
    MaxPlayers: 5,
    LastHeartbeat: "2026-09-11T09:14:49Z",
    JoinCode: "0ad1",
    Address: "34.30.42.253:7780",
  },
  {
    ID: "huginn-inst-3",
    ContainerID: "9afdd93f2b6ba59204f047649218907f09d8f6116a75ef7ed1234567890abcd",
    State: "unhealthy",
    PlayerCount: 1,
    MaxPlayers: 5,
    LastHeartbeat: "2026-09-11T09:12:10Z",
    JoinCode: "3fc2",
    Address: "34.30.42.253:7781",
  },
];

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
export default function Fleet({ instances }) {
  const [selectedId, setSelectedId] = useState(null);


  const totalInstances = instances.length;
  const totalPlayers = instances.reduce((sum, i) => sum + i.PlayerCount, 0);
  const available = instances.filter(
    (i) => i.State === "ready" && i.PlayerCount < i.MaxPlayers
  ).length;

  const selected = instances.find((i) => i.ID === selectedId) || null;

  const handleStop = (id) => {
    // wired to POST /api/servers/stop/{id} on Day 4 — no-op for now
    console.log("stop requested for", id);
  };

  return (
    <div style={styles.container}>
      <div style={styles.summaryRow}>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{totalInstances}</div>
          <div style={styles.statLabel}>Instances</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{totalPlayers}</div>
          <div style={styles.statLabel}>Players</div>
        </div>
        <div style={styles.statCard}>
          <div style={styles.statValue}>{available}</div>
          <div style={styles.statLabel}>Available</div>
        </div>
      </div>

      <div style={styles.splitView}>
        <div style={styles.sidebar}>
          {instances.map((inst) => (
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
          ))}
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

              <button style={styles.stopButton} onClick={() => handleStop(selected.ID)}>
                Stop Instance
              </button>
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
    detailHeader: { display: "flex",  justifyContent: "space-between",
         alignItems: "center", gap: "12px" },
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
  },
  placeholder: { color: "#6b7280", fontStyle: "italic" },
  detailTitle: { marginTop: 0 },
  detailRow: { marginBottom: "8px", color: "#ccc" },
  stopButton: {
    marginTop: "16px",
    background: "#dc2626",
    color: "#fff",
    border: "none",
    padding: "8px 16px",
    borderRadius: "4px",
    cursor: "pointer",
  },
};
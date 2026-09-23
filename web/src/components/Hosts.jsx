import { useState, useEffect } from "react";
import { authEventSource, authFetch } from "../auth";

const stateColors = {
  ready: "#4ade80",
  starting: "#facc15",
  unhealthy: "#f87171",
  draining: "#fb923c",
};
function Gauge({ label, value }) {
  const clamped = Math.max(0, Math.min(100, value));

  // SVG arc goes from 180° to 0°.
  const radius = 52;
  const circumference = Math.PI * radius;
  const progress = (clamped / 100) * circumference;

  return (
    <div style={styles.gaugeContainer}>
      <div style={styles.gaugeLabel}>{label}</div>

      <svg
        width="140"
        height="80"
        viewBox="0 0 140 80"
        style={styles.gauge}
      >
        {/* Background arc */}
        <path
          d="M 18 65 A 52 52 0 0 1 122 65"
          fill="none"
          stroke="#30363d"
          strokeWidth="10"
          strokeLinecap="round"
        />

        {/* Usage arc */}
        <path
          d="M 18 65 A 52 52 0 0 1 122 65"
          fill="none"
          stroke={getGaugeColor(clamped)}
          strokeWidth="10"
          strokeLinecap="round"
          strokeDasharray={`${progress} ${circumference}`}
        />

        {/* Value */}
        <text
          x="70"
          y="58"
          textAnchor="middle"
          fill="#fff"
          fontSize="20"
          fontWeight="600"
        >
          {clamped.toFixed(0)}%
        </text>
      </svg>
    </div>
  );
}
function getGaugeColor(value) {
  if (value >= 90) return "#f87171";
  if (value >= 70) return "#facc15";
  return "#4ade80";
}

export default function Hosts({ onSelectHost, instances = [] }) {
  const [hosts, setHosts] = useState([]);
  const [loaded, setLoaded] = useState(false);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState(null);
  const [toasts, setToasts] = useState([]);

  useEffect(() => {
    const es = authEventSource("/api/hosts/stream");

    es.onmessage = (event) => {
      try {
        setHosts(JSON.parse(event.data));
        setLoaded(true);
      } catch (err) {
        console.error("huginn: bad host snapshot", err);
      }
    };

    es.addEventListener("removed", (event) => {
      try {
        const removals = JSON.parse(event.data);
        setToasts((prev) => {
          const existingIds = new Set(prev.map((t) => t.hostID + t.t));
          const fresh = removals.filter((r) => !existingIds.has(r.hostID + r.t));
          return [...prev, ...fresh.map((r) => ({ ...r, id: r.hostID + r.t }))];
        });
      } catch (err) {
        console.error("huginn: bad removal event", err);
      }
    });

    return () => es.close();
  }, []);

  useEffect(() => {
    if (toasts.length === 0) return;
    const timer = setTimeout(() => {
      setToasts((prev) => prev.slice(1));
    }, 6000);
    return () => clearTimeout(timer);
  }, [toasts]);

  const handleCreateHost = async () => {
    setCreating(true);
    setCreateError(null);
    try {
      const res = await authFetch("/api/hosts/create", { method: "POST" });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text.trim() || `HTTP ${res.status}`);
      }
    } catch (err) {
      setCreateError(err.message);
    } finally {
      setCreating(false);
    }
  };

  return (
    <div style={styles.container}>

      <div style={styles.toastContainer}>
        {toasts.map((t) => (
          <div key={t.id} style={styles.toast}>
            <div style={styles.toastTitle}>Host removed: {t.hostID}</div>
            <div style={styles.toastReason}>{t.reason}</div>
          </div>
        ))}
      </div>
      <div style={styles.headerRow}>
        <button
          style={{
            ...styles.createButton,
            ...(creating ? styles.buttonDisabled : {}),
          }}
          disabled={creating}
          onClick={handleCreateHost}
        >
          {creating ? "Requesting…" : "+ Create Host"}
        </button>
      </div>

      {loaded && hosts.length > 0 && (
        <div style={styles.summaryRow}>
          <div style={{ ...styles.statCard, borderLeftColor: "#60a5fa" }}>
            <div style={styles.statValue}>{hosts.length}</div>
            <div style={styles.statLabel}>Hosts</div>
          </div>
          <div style={{ ...styles.statCard, borderLeftColor: "#4ade80" }}>
            <div style={styles.statValue}>
              {hosts.reduce((sum, h) => sum + h.InstanceCount, 0)}
            </div>
            <div style={styles.statLabel}>Instances</div>
          </div>
        </div>
      )}

      {createError && (
        <div style={styles.error}>Failed to create host: {createError}</div>
      )}

      {!loaded ? (
        <div style={styles.note}>Loading hosts…</div>
      ) : hosts.length === 0 ? (
        <div style={styles.note}>No hosts</div>
      ) : (
        <div style={styles.list}>
          {hosts.map((h) => (
            <div
              key={h.ID}
              style={{ ...styles.card, cursor: "pointer" }}
              onClick={() => onSelectHost(h.ID)}
            >
              <div style={styles.cardHeader}>
                <span
                  style={{
                    ...styles.dot,
                    background: stateColors[h.State],
                  }}
                />

                <span style={styles.id}>{h.ID}</span>

                {h.IsPrimary && (
                  <span style={styles.primaryBadge}>Primary</span>
                )}

                <span style={styles.state}>{h.State}</span>
              </div>

              <div style={styles.gauges}>
                <Gauge
                  label="CPU"
                  value={h.CPUPercent}
                />

                <Gauge
                  label="Memory"
                  value={h.MemoryPercent}
                />

                <div style={styles.instanceCount}>
                  <div style={styles.instanceNumber}>
                    {h.InstanceCount}
                  </div>

                  <div style={styles.instanceLabel}>
                    instances
                  </div>
                </div>
              </div>

              {instances.filter((i) => i.HostID === h.ID).length > 0 && (
                <div style={styles.instanceList}>
                  {instances
                    .filter((i) => i.HostID === h.ID)
                    .map((i) => (
                      <span key={i.ID} style={styles.instanceChip}>
                        {i.ID} · {i.PlayerCount}/{i.MaxPlayers}
                      </span>
                    ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const styles = {
  container: {
    padding: "24px",
  },

  headerRow: {
    display: "flex",
    alignItems: "center",
    justifyContent: "flex-end",
    marginBottom: "20px",
  maxWidth: "620px",
  },

  heading: {
    color: "#fff",
  },

  createButton: {
    background: "#238636",
    color: "#fff",
    border: "none",
    borderRadius: "6px",
    padding: "8px 16px",
    fontSize: "14px",
    fontWeight: "600",
    cursor: "pointer",
  },

  buttonDisabled: {
    opacity: 0.6,
    cursor: "not-allowed",
  },

  error: {
    color: "#f87171",
    marginBottom: "16px",
    fontSize: "14px",
  },

  note: {
    color: "#6b7280",
    fontStyle: "italic",
  },

  list: {
    display: "flex",
    flexDirection: "column",
    gap: "12px",
  },

  card: {
  maxWidth: "620px",
    background: "#161b22",
    borderRadius: "8px",
    padding: "16px 20px",
    color: "#fff",
  },

  cardHeader: {
    display: "flex",
    alignItems: "center",
    gap: "10px",
    marginBottom: "12px",
  },

  dot: {
    width: "10px",
    height: "10px",
    borderRadius: "50%",
  },

  id: {
    fontWeight: "bold",
    fontSize: "15px",
  },

  primaryBadge: {
    background: "rgba(96, 165, 250, 0.15)",
    color: "#60a5fa",
    border: "1px solid rgba(96, 165, 250, 0.35)",
    borderRadius: "4px",
    padding: "2px 8px",
    fontSize: "10px",
    fontWeight: "600",
    textTransform: "uppercase",
    letterSpacing: "0.05em",
  },

  state: {
    color: "#9ca3af",
    fontSize: "13px",
    marginLeft: "auto",
  },

  gauges: {
    display: "flex",
    alignItems: "center",
    gap: "28px",
  },

  gaugeContainer: {
    width: "140px",
    textAlign: "center",
  },

  gaugeLabel: {
    color: "#9ca3af",
    fontSize: "12px",
    marginBottom: "2px",
    textTransform: "uppercase",
    letterSpacing: "0.08em",
  },

  gauge: {
    display: "block",
  },

  instanceCount: {
    marginLeft: "auto",
    textAlign: "center",
    minWidth: "90px",
  },

  instanceNumber: {
    color: "#fff",
    fontSize: "24px",
    fontWeight: "600",
  },

  instanceLabel: {
    color: "#6b7280",
    fontSize: "12px",
  },

  summaryRow: {
    display: "flex",
    gap: "16px",
    marginBottom: "20px",
    maxWidth: "460px",
  },

  statCard: {
    flex: "1 1 0",
    background: "#161b22",
    border: "1px solid #21262d",
    borderLeft: "3px solid #30363d",
    borderRadius: "8px",
    padding: "14px 20px",
    textAlign: "center",
    color: "#fff",
  },

  statValue: { fontSize: "22px", fontWeight: "bold" },
  statLabel: { fontSize: "12px", color: "#9ca3af" },

  instanceList: {
    display: "flex",
    flexWrap: "wrap",
    gap: "6px",
    marginTop: "14px",
    paddingTop: "12px",
    borderTop: "1px solid #21262d",
  },

  instanceChip: {
    background: "#0d1117",
    border: "1px solid #21262d",
    borderRadius: "4px",
    padding: "3px 8px",
    fontSize: "11px",
    color: "#8b949e",
    fontFamily: "ui-monospace, Consolas, monospace",
  },

  toastContainer: {
    position: "fixed",
    bottom: "24px",
    right: "24px",
    display: "flex",
    flexDirection: "column",
    gap: "10px",
    zIndex: 1000,
  },

  toast: {
    background: "#1c1017",
    border: "1px solid rgba(248, 113, 113, 0.4)",
    borderLeft: "3px solid #f87171",
    borderRadius: "8px",
    padding: "12px 16px",
    minWidth: "280px",
    maxWidth: "360px",
    boxShadow: "0 8px 24px rgba(0, 0, 0, 0.4)",
  },

  toastTitle: {
    color: "#fff",
    fontSize: "13px",
    fontWeight: "600",
    marginBottom: "4px",
  },

  toastReason: {
    color: "#9ca3af",
    fontSize: "12px",
  },
};

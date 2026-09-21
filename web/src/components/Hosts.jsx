import { useState, useEffect } from "react";
import { authEventSource } from "../auth";

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

export default function Hosts({ onSelectHost }) {
  const [hosts, setHosts] = useState([]);
  const [loaded, setLoaded] = useState(false);

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

    return () => es.close();
  }, []);

  return (
    <div style={styles.container}>
      <h2 style={styles.heading}>Hosts</h2>

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

  heading: {
    color: "#fff",
    marginBottom: "20px",
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
};



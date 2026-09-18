import { useState, useEffect } from "react";

const stateColors = {
  ready: "#4ade80",
  starting: "#facc15",
  unhealthy: "#f87171",
  draining: "#fb923c",
};

export default function Hosts({ onSelectHost }) {
  const [hosts, setHosts] = useState([]);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    const es = new EventSource("/api/hosts/stream");

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
		  <div key={h.ID} style={{ ...styles.card, cursor: "pointer" }} onClick={() => onSelectHost(h.ID)}>
              <div style={styles.cardHeader}>
                <span style={{ ...styles.dot, background: stateColors[h.State] }} />
                <span style={styles.id}>{h.ID}</span>
                <span style={styles.state}>{h.State}</span>
              </div>
              <div style={styles.stats}>
                <span>CPU: {h.CPUPercent.toFixed(1)}%</span>
                <span>Memory: {h.MemoryPercent.toFixed(1)}%</span>
                <span>{h.InstanceCount} instance(s)</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

const styles = {
  container: { padding: "24px" },
  heading: { color: "#fff", marginBottom: "20px" },
  note: { color: "#6b7280", fontStyle: "italic" },
  list: { display: "flex", flexDirection: "column", gap: "12px" },
  card: {
    background: "#161b22",
    borderRadius: "8px",
    padding: "16px 20px",
    color: "#fff",
  },
  cardHeader: { display: "flex", alignItems: "center", gap: "10px", marginBottom: "8px" },
  dot: { width: "10px", height: "10px", borderRadius: "50%" },
  id: { fontWeight: "bold", fontSize: "15px" },
  state: { color: "#9ca3af", fontSize: "13px", marginLeft: "auto" },
  stats: { display: "flex", gap: "20px", fontSize: "13px", color: "#ccc" },
};

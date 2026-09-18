const statusDisplay = {
  live: { color: "#4ade80", label: "Live" },
  connecting: { color: "#facc15", label: "Reconnecting…" },
  offline: { color: "#f87171", label: "Disconnected" },
};

export default function Sidebar({ activeTab, onTabChange, status }) {
  const s = statusDisplay[status] ?? statusDisplay.offline;
  return (
    <nav style={styles.sidebar}>
      <div style={styles.brand}>Huginn</div>
      <div style={styles.links}>
        <button
          style={activeTab === "fleet" ? styles.linkActive : styles.link}
          onClick={() => onTabChange("fleet")}
        >
          Fleet
        </button>
        <button
          style={activeTab === "hosts" ? styles.linkActive : styles.link}
          onClick={() => onTabChange("hosts")}
        >
          Hosts
        </button>
        <button
          style={activeTab === "config" ? styles.linkActive : styles.link}
          onClick={() => onTabChange("config")}
        >
          Config
        </button>
      </div>
      <div style={styles.status}>
        <span style={{ color: s.color }}>● {s.label}</span>
      </div>
    </nav>
  );
}

const styles = {
  sidebar: {
    width: "200px",
    minHeight: "100vh",
    background: "#0d1117",
    color: "#fff",
    display: "flex",
    flexDirection: "column",
    padding: "20px 0",
    borderRight: "1px solid #21262d",
  },
  brand: {
    fontWeight: "bold",
    fontSize: "20px",
    padding: "0 20px 20px 20px",
  },
  links: {
    display: "flex",
    flexDirection: "column",
    gap: "4px",
    flex: 1,
  },
  link: {
    background: "none",
    border: "none",
    color: "#9ca3af",
    cursor: "pointer",
    fontSize: "15px",
    textAlign: "left",
    padding: "10px 20px",
  },
  linkActive: {
    background: "#161b22",
    border: "none",
    borderLeft: "3px solid #4ade80",
    color: "#fff",
    fontWeight: "bold",
    cursor: "pointer",
    fontSize: "15px",
    textAlign: "left",
    padding: "10px 17px",
  },
  status: {
    padding: "12px 20px 0 20px",
    fontSize: "13px",
    borderTop: "1px solid #21262d",
  },
};

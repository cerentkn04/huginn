// src/components/Navbar.jsx
export default function Navbar({ activeTab, onTabChange, connected }) {
  return (
    <nav style={styles.nav}>
      <div style={styles.left}>
        <span style={styles.brand}>Huginn</span>
        <button
          style={activeTab === "fleet" ? styles.linkActive : styles.link}
          onClick={() => onTabChange("fleet")}
        >
          Fleet
        </button>
        <button
          style={activeTab === "config" ? styles.linkActive : styles.link}
          onClick={() => onTabChange("config")}
        >
          Config
        </button>
      </div>
      <div style={styles.right}>
        <span style={{ color: connected ? "#4ade80" : "#f87171" }}>
          ● {connected ? "Live" : "Disconnected"}
        </span>
      </div>
    </nav>
  );
}

const styles = {
  nav: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    padding: "12px 24px",
    background: "#0d1117",
    color: "#fff",
  },
  left: { display: "flex", alignItems: "center", gap: "24px" },
  right: { display: "flex", alignItems: "center", gap: "16px" },
  brand: { fontWeight: "bold", fontSize: "20px" },
  link: {
    background: "none",
    border: "none",
    color: "#ccc",
    cursor: "pointer",
    fontSize: "15px",
  },
  linkActive: {
    background: "none",
    border: "none",
    color: "#fff",
    fontWeight: "bold",
    cursor: "pointer",
    fontSize: "15px",
  },
};
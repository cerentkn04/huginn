// src/components/Config.jsx
import { useState } from "react";

const defaultConfig = {
  game: "",
  image: "",
  min_instances: 2,
  max_instances: 5,
  buffer_size: 2,
  max_players: 5,
  port: 7778,
  heartbeat_timeout_seconds: 15,
  udp_listen_addr: "0.0.0.0:9000",
  http_listen_addr: ":8080",
  mode: "sdk",
  cloud_provider: "gcp", // default
  public_host: "",
};

export default function Config() {
  const [config, setConfig] = useState(defaultConfig);

  const handleChange = (field, value) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
  };

  const handleSave = () => {
    // wired to POST /api/config on Day 5 — no-op for now
    console.log("save requested", config);
  };

  return (
    <div style={styles.container}>
      <h2 style={styles.heading}>Fleet Configuration</h2>

      <Field label="Game">
        <input
          style={styles.input}
          value={config.game}
          onChange={(e) => handleChange("game", e.target.value)}
        />
      </Field>

      <Field label="Image">
        <input
          style={styles.input}
          value={config.image}
          onChange={(e) => handleChange("image", e.target.value)}
        />
      </Field>

      <div style={styles.row}>
        <Field label="Min Instances">
          <input
            type="number"
            style={styles.input}
            value={config.min_instances}
            onChange={(e) => handleChange("min_instances", Number(e.target.value))}
          />
        </Field>
        <Field label="Max Instances">
          <input
            type="number"
            style={styles.input}
            value={config.max_instances}
            onChange={(e) => handleChange("max_instances", Number(e.target.value))}
          />
        </Field>
        <Field label="Buffer Size">
          <input
            type="number"
            style={styles.input}
            value={config.buffer_size}
            onChange={(e) => handleChange("buffer_size", Number(e.target.value))}
          />
        </Field>
      </div>

      <div style={styles.row}>
        <Field label="Max Players">
          <input
            type="number"
            style={styles.input}
            value={config.max_players}
            onChange={(e) => handleChange("max_players", Number(e.target.value))}
          />
        </Field>
        <Field label="Port">
          <input
            type="number"
            style={styles.input}
            value={config.port}
            onChange={(e) => handleChange("port", Number(e.target.value))}
          />
        </Field>
        <Field label="Heartbeat Timeout (s)">
          <input
            type="number"
            style={styles.input}
            value={config.heartbeat_timeout_seconds}
            onChange={(e) => handleChange("heartbeat_timeout_seconds", Number(e.target.value))}
          />
        </Field>
      </div>

      <div style={styles.row}>
        <Field label="UDP Listen Addr">
          <input
            style={styles.input}
            value={config.udp_listen_addr}
            onChange={(e) => handleChange("udp_listen_addr", e.target.value)}
          />
        </Field>
        <Field label="HTTP Listen Addr">
          <input
            style={styles.input}
            value={config.http_listen_addr}
            onChange={(e) => handleChange("http_listen_addr", e.target.value)}
          />
        </Field>
      </div>

      <Field label="Mode">
        <select
          style={styles.input}
          value={config.mode}
          onChange={(e) => handleChange("mode", e.target.value)}
        >
          <option value="sdk">sdk</option>
          <option value="log_parse">log_parse</option>
        </select>
      </Field>

      <Field label="Public Host Discovery">
        <select
          style={styles.input}
          value={config.cloud_provider}
          onChange={(e) => handleChange("cloud_provider", e.target.value)}
        >
          <option value="gcp">GCP (auto-detect)</option>
          <option value="aws">AWS (auto-detect)</option>
          <option value="custom">Custom</option>
        </select>
      </Field>

      {config.cloud_provider === "custom" && (
        <Field label="Public Host">
          <input
            style={styles.input}
            placeholder="e.g. mygameserver.example.com"
            value={config.public_host}
            onChange={(e) => handleChange("public_host", e.target.value)}
          />
        </Field>
      )}

      <button style={styles.saveButton} onClick={handleSave}>
        Save Configuration
      </button>
      <p style={styles.note}>Changes take effect after restarting Huginn.</p>
    </div>
  );
}

function Field({ label, children }) {
  return (
    <div style={styles.field}>
      <label style={styles.label}>{label}</label>
      {children}
    </div>
  );
}

const styles = {
  container: { padding: "24px", maxWidth: "700px", color: "#fff" },
  heading: { marginBottom: "20px" },
  row: { display: "flex", gap: "16px" },
  field: { marginBottom: "24px", flex: 1 },
  label: {
    display: "block",
    fontSize: "12px",
    color: "#9ca3af",
    marginBottom: "4px",
    textTransform: "uppercase",
    letterSpacing: "0.03em",
  },
  input: {
    width: "100%",
    padding: "8px 10px",
    background: "#161b22",
    border: "1px solid #30363d",
    borderRadius: "6px",
    color: "#fff",
    boxSizing: "border-box",
  },
  saveButton: {
    background: "#238636",
    color: "#fff",
    border: "none",
    padding: "10px 20px",
    borderRadius: "6px",
    cursor: "pointer",
    marginTop: "8px",
  },
  note: { fontSize: "13px", color: "#6b7280", marginTop: "8px" },
};
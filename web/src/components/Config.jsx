import { useState, useEffect } from "react";
import { authFetch } from "../auth";
export default function Config() {
  const [config, setConfig] = useState(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(null);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    authFetch("/api/config")
      .then(async (res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
      })
      .then(setConfig)
      .catch((err) => setLoadError(err.message))
      .finally(() => setLoading(false));
  }, []);

  const handleChange = (field, value) => {
    setConfig((prev) => ({ ...prev, [field]: value }));
    setSaved(false);
    setSaveError(null);
  };

  const handleSave = async () => {
    setSaving(true);
    setSaveError(null);
    setSaved(false);
    try {
      const res = await authFetch("/api/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text.trim() || `HTTP ${res.status}`);
      }
      setSaved(true);
    } catch (err) {
      setSaveError(err.message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return <div style={styles.container}>Loading configuration…</div>;
  }
  if (loadError) {
    return (
      <div style={styles.container}>
        <div style={styles.error}>Could not load config: {loadError}</div>
      </div>
    );
  }

  const errors = saveError ? saveError.split("; ") : [];

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
        <Field label="Max Players" hint="Requires a restart to take effect">
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
            onChange={(e) =>
              handleChange("heartbeat_timeout_seconds", Number(e.target.value))
            }
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

<Field label="GCP Project ID">
  <input
    style={styles.input}
    value={config.gcp_project || ""}
    onChange={(e) => handleChange("gcp_project", e.target.value)}
  />
</Field>
<Field label="GCP Zone">
  <input
    style={styles.input}
    value={config.gcp_zone || ""}
    onChange={(e) => handleChange("gcp_zone", e.target.value)}
  />
</Field>
	  <Field label="Host Auto-Scaling" hint="Off by default — enabling this lets Huginn automatically create new GCP hosts when existing capacity is exhausted, which can incur real cloud costs.">
  <label style={{ display: "flex", alignItems: "center", gap: "8px", color: "#fff" }}>
    <input
      type="checkbox"
      checked={!!config.host_auto_scaling_enabled}
      onChange={(e) => handleChange("host_auto_scaling_enabled", e.target.checked)}
    />
    Enable automatic host scaling
  </label>
</Field>
{config.host_auto_scaling_enabled && (
  <Field label="Scale-Down Idle Threshold (minutes)" hint="A host with zero running instances is removed after being idle this long.">
    <input
      type="number"
      style={styles.input}
      value={config.host_scale_down_idle_minutes}
      onChange={(e) => handleChange("host_scale_down_idle_minutes", Number(e.target.value))}
    />
  </Field>
)}
      {errors.length > 0 && (
        <div style={styles.error}>
          {errors.length === 1 ? (
            errors[0]
          ) : (
            <ul style={styles.errorList}>
              {errors.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          )}
        </div>
      )}

      {saved && (
        <div style={styles.success}>
        Saved — applied immediately. (Image, max players, and listen addresses still need a restart.)
	</div>
      )}

      <button
        style={{
          ...styles.saveButton,
          ...(saving ? styles.buttonDisabled : {}),
        }}
        disabled={saving}
        onClick={handleSave}
      >
        {saving ? "Saving…" : "Save Configuration"}
      </button>
    </div>
  );
}

function Field({ label, children , hint }) {
  return (
    <div style={styles.field}>
      <label style={styles.label}>{label}</label>
      {hint && <div style={styles.hint}>{hint}</div>}
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
    colorScheme: "dark",
    fontFamily: "inherit",
    fontSize: "13px",
  },
  saveButton: {
    background: "#238636",
    color: "#fff",
    border: "none",
    padding: "8px 16px",
    borderRadius: "6px",
    cursor: "pointer",
    fontSize: "13px",
    fontWeight: 500,
    fontFamily: "inherit",
  },
	hint: {
  fontSize: "11px",
  color: "#6b7280",
  marginTop: "4px",
},
  buttonDisabled: { opacity: 0.5, cursor: "not-allowed" },
  error: {
    color: "#fca5a5",
    background: "rgba(220, 38, 38, 0.12)",
    border: "1px solid rgba(220, 38, 38, 0.35)",
    padding: "10px 14px",
    borderRadius: "6px",
    marginBottom: "16px",
    fontSize: "13px",
  },
  errorList: { margin: 0, paddingLeft: "18px" },
  success: {
    color: "#86efac",
    background: "rgba(34, 197, 94, 0.12)",
    border: "1px solid rgba(34, 197, 94, 0.35)",
    padding: "10px 14px",
    borderRadius: "6px",
    marginBottom: "16px",
    fontSize: "13px",
  },
};

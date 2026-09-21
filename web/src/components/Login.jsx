import { useState } from "react";
import { setToken } from "../auth";

export default function Login({ onLoggedIn }) {
  const [token, setTokenInput] = useState("");
  const [error, setError] = useState(null);
  const [checking, setChecking] = useState(false);
  const [showToken, setShowToken] = useState(false);

    const handleSubmit = async (e) => {
    e.preventDefault();
    setChecking(true);
    setError(null);
    try {
      const res = await fetch("/api/fleet", {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        throw new Error("Invalid token");
      }
      setToken(token);
      onLoggedIn();
    } catch (err) {
      setError(err.message);
    } finally {
      setChecking(false);
    }
  };
  return (
    <div style={styles.container}>
      <form onSubmit={handleSubmit} style={styles.form}>
        <h2 style={styles.heading}>Huginn</h2>
        <p style={styles.label}>Enter your access token</p>
        <div style={styles.inputRow}>
          <input
            type={showToken ? "text" : "password"}
            value={token}
            onChange={(e) => setTokenInput(e.target.value)}
            style={styles.input}
            autoFocus
          />
          <button
            type="button"
            onClick={() => setShowToken((s) => !s)}
            style={styles.toggleButton}
            tabIndex={-1}
          >
            {showToken ? "Hide" : "Show"}
          </button>
        </div>
        {error && <div style={styles.error}>{error}</div>}
        <button type="submit" disabled={checking} style={styles.button}>
          {checking ? "Checking…" : "Continue"}
        </button>
      </form>
    </div>
  );
}
 
const styles = {
  container: { display: "flex", alignItems: "center", justifyContent: "center", minHeight: "100vh", background: "#0d1117" },
  form: { background: "#161b22", padding: "32px", borderRadius: "8px", width: "320px" },
  heading: { color: "#fff", marginTop: 0 },
  label: { color: "#9ca3af", fontSize: "13px", marginBottom: "8px" },
  input: { width: "100%", padding: "10px", background: "#0d1117", border: "1px solid #30363d", borderRadius: "6px", color: "#fff", marginBottom: "12px", boxSizing: "border-box" },
  toggleButton: { padding: "0 12px", background: "#374151", border: "none", borderRadius: "6px", color: "#e5e7eb", fontSize: "12px", cursor: "pointer" },
  button: { width: "100%", padding: "10px", background: "#4ade80", border: "none", borderRadius: "6px", color: "#0d1117", fontWeight: "bold", cursor: "pointer" },
  error: { color: "#f87171", fontSize: "13px", marginBottom: "12px" },
};

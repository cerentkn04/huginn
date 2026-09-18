import { useState, useEffect } from "react";
import Sidebar from "./components/Sidebar";
import Fleet from "./components/Fleet";
import Hosts from "./components/Hosts";
import Config from "./components/Config";

export default function App() {
  const [activeTab, setActiveTab] = useState("fleet");
  const [instances, setInstances] = useState([]);
  const [status, setStatus] = useState("connecting");
  const [connected, setConnected] = useState(false);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    const es = new EventSource("/api/fleet/stream");

    es.onopen = () => setConnected(true);

    es.onmessage = (event) => {
      try {
        setInstances(JSON.parse(event.data));
        setLoaded(true);
        setStatus("live");
      } catch (err) {
        console.error("huginn: bad snapshot", err);
      }
    };

    es.onerror = () => {
      setStatus(es.readyState === EventSource.CLOSED ? "offline" : "connecting");
    };

    return () => es.close();
  }, []);

  return (
    <div style={styles.layout}>
      <Sidebar activeTab={activeTab} onTabChange={setActiveTab} status={status} />
      <div style={styles.content}>
        {activeTab === "fleet" ? (
          <Fleet instances={instances} loaded={loaded} />
        ) : activeTab === "hosts" ? (
          <Hosts />
        ) : (
          <Config />
        )}
      </div>
    </div>
  );
}

const styles = {
  layout: { display: "flex", minHeight: "100vh" },
  content: { flex: 1, minWidth: 0 },
};

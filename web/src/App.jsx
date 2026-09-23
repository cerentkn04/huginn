import { useState, useEffect } from "react";
import Sidebar from "./components/Sidebar";
import Fleet from "./components/Fleet";
import Hosts from "./components/Hosts";
import Config from "./components/Config";

import Login from "./components/Login";
import { getToken, authEventSource } from "./auth";

export default function App() {
 const [loggedIn, setLoggedIn] = useState(!!getToken());
  const [activeTab, setActiveTab] = useState("fleet");
  const [hostFilter, setHostFilter] = useState(null);
  const [instances, setInstances] = useState([]);
  const [status, setStatus] = useState("connecting");
  const [connected, setConnected] = useState(false);
  const [loaded, setLoaded] = useState(false);
  const [peakToday, setPeakToday] = useState(0);

  useEffect(() => {
    if (!loggedIn) return;
    const es = authEventSource("/api/fleet/stream");
    es.onopen = () => setConnected(true);
    es.addEventListener("peak", (event) => {
      const n = Number(event.data);
      if (!Number.isNaN(n)) setPeakToday(n);
    });
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

  const goToHost = (hostID) => {
    setHostFilter(hostID);
    setActiveTab("fleet");
  };

  const handleTabChange = (tab) => {
    setActiveTab(tab);
    if (tab !== "fleet") setHostFilter(null);
  };
  if (!loggedIn) {
    return <Login onLoggedIn={() => setLoggedIn(true)} />;
  }

  return (
    <div style={styles.layout}>
      <Sidebar activeTab={activeTab} onTabChange={handleTabChange} status={status} />
      <div style={styles.content}>
        {activeTab === "fleet" ? (
          <Fleet
            instances={instances}
            loaded={loaded}
            peakToday={peakToday}
            hostFilter={hostFilter}
            onClearFilter={() => setHostFilter(null)}
          />
        ) : activeTab === "hosts" ? (
          <Hosts onSelectHost={goToHost} instances={instances} />
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

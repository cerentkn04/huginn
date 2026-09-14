import { useState, useEffect } from "react";
import Navbar from "./components/Navbar";
import Fleet from "./components/Fleet";
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
  <div>
      <Navbar activeTab={activeTab} onTabChange={setActiveTab} status={status} />
      {activeTab === "fleet" ? (
        <Fleet instances={instances} loaded={loaded} />
      ) : (
        <Config />
      )}
    </div>
  );
}
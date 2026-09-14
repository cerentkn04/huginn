import { useState, useEffect } from "react";
import Navbar from "./components/Navbar";
import Fleet from "./components/Fleet";
import Config from "./components/Config";

export default function App() {
  const [activeTab, setActiveTab] = useState("fleet");
  const [instances, setInstances] = useState([]);
  const [connected, setConnected] = useState(false);

  useEffect(() => {
    const es = new EventSource("/api/fleet/stream");

    es.onopen = () => setConnected(true);

    es.onmessage = (event) => {
      setInstances(JSON.parse(event.data));
      setConnected(true);
    };

    es.onerror = () => setConnected(false);

    return () => es.close();
  }, []);

  return (
    <div>
      <Navbar activeTab={activeTab} onTabChange={setActiveTab} connected={connected} />
      {activeTab === "fleet" ? <Fleet instances={instances} /> : <Config />}
    </div>
  );
}
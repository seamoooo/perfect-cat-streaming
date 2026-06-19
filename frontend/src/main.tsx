import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./styles.css";
import { initBrowserAgent } from "./lib/observability/browser";

// Bring up New Relic Browser Agent before React renders so the very first
// page-load metric is captured. No-op when env vars are missing.
initBrowserAgent();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

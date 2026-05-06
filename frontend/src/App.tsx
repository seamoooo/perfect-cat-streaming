import { useMemo } from "react";
import { BrowserRouter, Route, Routes } from "react-router-dom";
import { GalleryPage } from "./routes/GalleryPage";
import { VideoDetailPage } from "./routes/VideoDetailPage";
import { ConsoleKanpachiSink, NewRelicKanpachiSink, type KanpachiSink } from "./lib/telemetry";

function chooseSink(): KanpachiSink {
  // Use New Relic when the agent is detected on window; otherwise console.
  if (typeof window !== "undefined" && window.newrelic) {
    return new NewRelicKanpachiSink();
  }
  return new ConsoleKanpachiSink();
}

function genSessionId(): string {
  const arr = new Uint8Array(8);
  if (typeof crypto !== "undefined" && crypto.getRandomValues) {
    crypto.getRandomValues(arr);
  } else {
    for (let i = 0; i < arr.length; i++) arr[i] = Math.floor(Math.random() * 256);
  }
  return Array.from(arr, (b) => b.toString(16).padStart(2, "0")).join("");
}

export default function App() {
  const sink = useMemo(chooseSink, []);
  const sessionId = useMemo(genSessionId, []);
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<GalleryPage />} />
        <Route
          path="/clips/:id"
          element={<VideoDetailPage sink={sink} sessionId={sessionId} />}
        />
      </Routes>
    </BrowserRouter>
  );
}

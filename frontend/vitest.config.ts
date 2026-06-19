import { defineConfig } from "vitest/config";

// Vitest runs in Node and shares Vite's transformer, but Playwright lives in
// e2e/ and must not be picked up here.
export default defineConfig({
  test: {
    exclude: ["node_modules", "dist", "e2e"],
  },
});

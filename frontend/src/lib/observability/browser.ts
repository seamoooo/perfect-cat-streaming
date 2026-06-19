// New Relic Browser Agent loader.
//
// Initializes the SPA-flavoured agent at app boot. All values come from Vite
// build-time env vars so the script is configured before React renders.
// Missing env vars => no agent is instantiated and the app runs unchanged.
//
// The agent installs `window.newrelic` which the existing
// NewRelicKanpachiSink.addPageAction() then routes player QoE events into.

import { BrowserAgent } from "@newrelic/browser-agent/loaders/browser-agent";

interface BrowserEnv {
  licenseKey: string;
  applicationID: string;
  accountID: string;
  trustKey: string;
  agentID: string;
}

function readEnv(): BrowserEnv | null {
  const e = import.meta.env;
  const required = {
    licenseKey: (e.VITE_NEW_RELIC_LICENSE_KEY as string | undefined) ?? "",
    applicationID: (e.VITE_NEW_RELIC_APP_ID as string | undefined) ?? "",
    accountID: (e.VITE_NEW_RELIC_ACCOUNT_ID as string | undefined) ?? "",
    trustKey: (e.VITE_NEW_RELIC_TRUST_KEY as string | undefined) ?? "",
    agentID: (e.VITE_NEW_RELIC_AGENT_ID as string | undefined) ?? "",
  };
  for (const k of Object.keys(required) as (keyof BrowserEnv)[]) {
    if (!required[k]) return null;
  }
  return required;
}

let agent: BrowserAgent | null = null;

export function initBrowserAgent(): void {
  if (agent) return; // idempotent
  const env = readEnv();
  if (!env) {
    console.info(
      "[newrelic] browser agent disabled (set VITE_NEW_RELIC_LICENSE_KEY / APP_ID / ACCOUNT_ID / TRUST_KEY / AGENT_ID to enable)",
    );
    return;
  }

  // Options track the snippet NR generates for the "Install with NPM" flow.
  // The published TS types want string IDs; the runtime accepts both.
  const options = {
    info: {
      applicationID: env.applicationID,
      beacon: "bam.nr-data.net",
      errorBeacon: "bam.nr-data.net",
      licenseKey: env.licenseKey,
      sa: 1,
    },
    init: {
      ajax: { deny_list: ["bam.nr-data.net"] },
      browser_consent_mode: { enabled: false },
      distributed_tracing: { enabled: true },
      performance: {
        capture_detail: false,
        capture_marks: false,
        capture_measures: true,
      },
      privacy: { cookies_enabled: true },
      // Session Replay — sampling rates come from the NR UI; the agent only
      // needs the feature toggled on. Pro / Pro+SPA agent required.
      session_replay: {
        enabled: true,
        // Mirror the values configured server-side at NR UI > Browser settings
        // > Session replay; if they drift, server-side wins.
        sampling_rate: 10.0,
        error_sampling_rate: 100.0,
        // Default-privacy: mask all text + inputs. Flip to false if you set
        // explicit allow/deny selectors in NR UI.
        mask_all_inputs: true,
        block_selector: "",
        mask_text_selector: "*",
        // Pull cross-origin CSS so replays render correctly (matches the
        // "Fetch cross-origin CSS" toggle in NR UI).
        fix_stylesheets: true,
        // Lazy-load the replay library after page-load — leaves "Record initial
        // page load" OFF, matching the UI default. Flip to true if you want
        // first-frame capture (at a small startup cost).
        preload: false,
      },
      // Session Trace — paired SPA/timeline view alongside replay.
      session_trace: { enabled: true },
    },
    loader_config: {
      accountID: env.accountID,
      agentID: env.agentID,
      applicationID: env.applicationID,
      licenseKey: env.licenseKey,
      trustKey: env.trustKey,
    },
  };

  agent = new BrowserAgent(options);
  console.info(`[newrelic] browser agent initialised app=${env.applicationID}`);
}

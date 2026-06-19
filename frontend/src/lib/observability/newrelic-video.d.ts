// NR's @newrelic/video-* packages ship no TypeScript declarations.
// We use only the constructor + dispose + setAdsTracker surface; declaring
// them as opaque modules is enough to satisfy strict mode without bringing
// in any non-existent @types/* shim.

declare module "@newrelic/video-html5" {
  const Html5Tracker: new (player: HTMLVideoElement, options?: unknown) => {
    dispose?: () => void;
    setAdsTracker?: (t: unknown) => void;
  };
  export default Html5Tracker;
}

declare module "@newrelic/video-html5/src/ads/ima.js" {
  const Html5ImaAdsTracker: new (imaAttachment: unknown, options?: unknown) => {
    dispose?: () => void;
  };
  export default Html5ImaAdsTracker;
}

declare module "@newrelic/video-core" {
  const nrvideo: {
    Core: {
      addTracker: (tracker: unknown, options?: unknown) => void;
      removeTracker: (tracker: unknown) => void;
    };
    [key: string]: unknown;
  };
  export default nrvideo;
}

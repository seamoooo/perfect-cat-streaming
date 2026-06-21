import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
  /** optional override of the fallback UI */
  fallback?: ReactNode;
}

interface State {
  error: Error | null;
}

/**
 * Catches render-time errors in its subtree, reports them to the New Relic
 * Browser agent, and shows a fallback instead of white-screening the SPA.
 *
 * Used (among other things) by the developer "frontend" chaos demo: a render
 * error is raised deliberately so operators can see the JS error surface in New
 * Relic Browser exactly like a real crash.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    // Route through the same browser agent the rest of the app uses. The agent
    // also captures this via componentStack, but an explicit noticeError adds
    // our marker attribute so demo errors are filterable in NR.
    window.newrelic?.noticeError?.(error, {
      "chaos.injected": "frontend_render_error",
      componentStack: info.componentStack ?? "",
    });
    console.error("[ErrorBoundary]", error, info.componentStack);
  }

  reset = () => this.setState({ error: null });

  render() {
    if (this.state.error) {
      if (this.props.fallback) return this.props.fallback;
      return (
        <div role="alert" className="error-boundary">
          <div style={{ fontSize: 40, marginBottom: 8 }}>🙀</div>
          <p style={{ margin: "0 0 4px", fontWeight: 700, fontSize: 18 }}>
            画面の描画でエラーが発生しました
          </p>
          <p style={{ margin: "0 0 16px", fontSize: 13, opacity: 0.75 }}>
            ブラウザ起因のエラー（New Relic デモ用カオス）
          </p>
          <button
            type="button"
            className="btn btn-primary"
            onClick={this.reset}
          >
            ↻ 再読み込み
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

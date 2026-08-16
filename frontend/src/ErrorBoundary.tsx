import { Component, type PropsWithChildren, type ReactNode } from "react";

type ErrorBoundaryState = { hasError: boolean };

export default class ErrorBoundary extends Component<PropsWithChildren, ErrorBoundaryState> {
  state: ErrorBoundaryState = { hasError: false };

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  render(): ReactNode {
    if (!this.state.hasError) {
      return this.props.children;
    }

    return (
      <main className="shell">
        <section className="panel" role="alert">
          <div className="panel-heading"><div><p className="eyebrow">ERROR</p><h1>画面の表示に失敗しました</h1></div></div>
          <p className="lead">一時的な問題が発生しました。アプリを再読み込みしてください。</p>
          <button className="primary" onClick={() => window.location.reload()}>再読み込み</button>
        </section>
      </main>
    );
  }
}

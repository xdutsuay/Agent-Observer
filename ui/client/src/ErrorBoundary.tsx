import React from 'react';

export class ErrorBoundary extends React.Component<{children: React.ReactNode}, {error: any}> {
  state = { error: null };
  static getDerivedStateFromError(error: any) { return { error }; }
  render() {
    if (this.state.error) {
      return (
        <div style={{ color: 'red', padding: 20, whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
          <h2>React Crashed</h2>
          {String(this.state.error.stack || this.state.error)}
        </div>
      );
    }
    return this.props.children;
  }
}

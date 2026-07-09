import { useEffect, useState } from 'react'
import { getAppInfo, type AppInfo } from '../shared/api/appInfo'

type LoadState =
  | { status: 'loading' }
  | { status: 'ready'; appInfo: AppInfo }
  | { status: 'error'; message: string }

function App() {
  const [state, setState] = useState<LoadState>({ status: 'loading' })

  useEffect(() => {
    let active = true

    getAppInfo()
      .then((appInfo) => {
        if (active) {
          setState({ status: 'ready', appInfo })
        }
      })
      .catch((error: unknown) => {
        if (active) {
          setState({
            status: 'error',
            message: error instanceof Error ? error.message : String(error),
          })
        }
      })

    return () => {
      active = false
    }
  }, [])

  return (
    <main className="app-shell">
      <section className="status-panel">
        <div className="status-panel__header">
          <span className="status-panel__eyebrow">Wails v3</span>
          <h1>Cove</h1>
        </div>

        {state.status === 'loading' && (
          <p className="status-panel__message">Loading application metadata...</p>
        )}

        {state.status === 'error' && (
          <p className="status-panel__message status-panel__message--error">
            {state.message}
          </p>
        )}

        {state.status === 'ready' && (
          <dl className="metadata-grid">
            <div>
              <dt>Name</dt>
              <dd>{state.appInfo.name}</dd>
            </div>
            <div>
              <dt>Version</dt>
              <dd>{state.appInfo.version}</dd>
            </div>
            <div>
              <dt>Platform</dt>
              <dd>{state.appInfo.platform}</dd>
            </div>
            <div>
              <dt>Architecture</dt>
              <dd>{state.appInfo.arch}</dd>
            </div>
          </dl>
        )}
      </section>
    </main>
  )
}

export default App

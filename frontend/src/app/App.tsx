import { useEffect, useState } from 'react'
import './App.css'
import { AuthScreen } from '../features/auth/AuthScreen'
import { clearSession, restoreSession } from '../features/auth/api'
import type { StoredSession } from '../features/auth/types'
import { getAppInfo, type AppInfo } from '../shared/api/appInfo'

const coveIcon = '/cove-mark.svg'

type LoadState =
  | { status: 'loading' }
  | { status: 'ready'; appInfo: AppInfo }
  | { status: 'error'; message: string }

type AuthState =
  | { status: 'restoring' }
  | { status: 'anonymous' }
  | { status: 'authenticated'; session: StoredSession }

type AuthenticatedAppProps = {
  session: StoredSession
  onLogout: () => void
}

function AuthenticatedApp({ session, onLogout }: AuthenticatedAppProps) {
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

  const displayName = session.user.nickname || session.user.username

  return (
    <main className="app-shell app-shell--authenticated">
      <section className="home-panel">
        <header className="home-header">
          <div className="brand-lockup">
            <img className="brand-lockup__icon" src={coveIcon} alt="" />
            <span>Cove</span>
          </div>
          <button className="secondary-button" type="button" onClick={onLogout}>
            退出登录
          </button>
        </header>

        <div className="welcome-block">
          <p className="welcome-block__label">已登录</p>
          <h1>你好，{displayName}</h1>
          <p>你的 Cove 已准备就绪。</p>
        </div>

        {state.status === 'loading' && (
          <div className="metadata-skeleton" aria-label="正在加载应用信息">
            <span />
            <span />
            <span />
            <span />
          </div>
        )}

        {state.status === 'error' && (
          <p className="status-message status-message--error" role="alert">
            {state.message}
          </p>
        )}

        {state.status === 'ready' && (
          <dl className="metadata-grid">
            <div>
              <dt>应用</dt>
              <dd>{state.appInfo.name}</dd>
            </div>
            <div>
              <dt>版本</dt>
              <dd>{state.appInfo.version}</dd>
            </div>
            <div>
              <dt>平台</dt>
              <dd>{state.appInfo.platform}</dd>
            </div>
            <div>
              <dt>架构</dt>
              <dd>{state.appInfo.arch}</dd>
            </div>
          </dl>
        )}
      </section>
    </main>
  )
}

function App() {
  const [authState, setAuthState] = useState<AuthState>({ status: 'restoring' })

  useEffect(() => {
    let active = true
    restoreSession().then((session) => {
      if (!active) {
        return
      }
      setAuthState(session ? { status: 'authenticated', session } : { status: 'anonymous' })
    })
    return () => {
      active = false
    }
  }, [])

  function handleLogout() {
    clearSession()
    setAuthState({ status: 'anonymous' })
  }

  if (authState.status === 'restoring') {
    return (
      <main className="launch-screen" aria-label="正在恢复登录状态">
        <img src={coveIcon} alt="" />
        <strong>Cove</strong>
        <span className="launch-screen__progress" />
      </main>
    )
  }

  if (authState.status === 'anonymous') {
    return (
      <AuthScreen
        onAuthenticated={(session) => setAuthState({ status: 'authenticated', session })}
      />
    )
  }

  return <AuthenticatedApp session={authState.session} onLogout={handleLogout} />
}

export default App

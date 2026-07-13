import { useCallback, useEffect, useState } from 'react'
import './App.css'
import { AuthScreen } from '../features/auth/AuthScreen'
import { clearSession, restoreSession } from '../features/auth/api'
import type { StoredSession } from '../features/auth/types'
import { ChatScreen } from '../features/chat/ChatScreen'
import { ProfileScreen } from '../features/profile/ProfileScreen'

const coveIcon = '/cove-mark.svg'

type AuthState =
  | { status: 'restoring' }
  | { status: 'anonymous' }
  | { status: 'authenticated'; session: StoredSession }

type AuthenticatedAppProps = {
  session: StoredSession
  onLogout: () => void
  onSessionChange: (session: StoredSession) => void
}

export function AuthenticatedApp({ session, onLogout, onSessionChange }: AuthenticatedAppProps) {
  const [profileOpen, setProfileOpen] = useState(false)

  return (
    <div className="authenticated-app">
      <div className="authenticated-app__chat" aria-hidden={profileOpen} inert={profileOpen ? true : undefined}>
        <ChatScreen session={session} onLogout={onLogout} onOpenProfile={() => setProfileOpen(true)} />
      </div>
      {profileOpen && (
        <ProfileScreen
          session={session}
          onBack={() => setProfileOpen(false)}
          onLogout={onLogout}
          onSessionChange={onSessionChange}
        />
      )}
    </div>
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

  const handleLogout = useCallback(() => {
    clearSession()
    setAuthState({ status: 'anonymous' })
  }, [])

  const handleSessionChange = useCallback((session: StoredSession) => {
    setAuthState({ status: 'authenticated', session })
  }, [])

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

  return (
    <AuthenticatedApp
      session={authState.session}
      onLogout={handleLogout}
      onSessionChange={handleSessionChange}
    />
  )
}

export default App

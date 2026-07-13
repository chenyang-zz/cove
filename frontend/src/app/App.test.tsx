// @vitest-environment jsdom

import { cleanup, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import type { StoredSession } from '../features/auth/types'

let chatMounts = 0
let chatUnmounts = 0

vi.mock('../features/chat/ChatScreen', async () => {
  const React = await import('react')
  return {
    ChatScreen: ({ onOpenProfile }: { onOpenProfile: () => void }) => {
      React.useEffect(() => {
        chatMounts += 1
        return () => {
          chatUnmounts += 1
        }
      }, [])
      return <button type="button" onClick={onOpenProfile}>打开个人信息</button>
    },
  }
})

vi.mock('../features/profile/ProfileScreen', () => ({
  ProfileScreen: ({ onBack }: { onBack: () => void }) => (
    <section aria-label="个人信息覆盖层"><button type="button" onClick={onBack}>返回</button></section>
  ),
}))

import { AuthenticatedApp } from './App'

const session: StoredSession = {
  accessToken: 'access',
  refreshToken: 'refresh',
  user: { id: 'user-1', username: 'linhai', nickname: '林海', email: null, avatar: null },
}

afterEach(() => {
  cleanup()
  chatMounts = 0
  chatUnmounts = 0
})

describe('AuthenticatedApp', () => {
  it('keeps chat mounted while the profile overlay opens and closes', async () => {
    const user = userEvent.setup()
    render(
      <AuthenticatedApp session={session} onLogout={vi.fn()} onSessionChange={vi.fn()} />,
    )
    expect(chatMounts).toBe(1)

    await user.click(screen.getByRole('button', { name: '打开个人信息' }))
    expect(screen.getByLabelText('个人信息覆盖层')).toBeTruthy()
    expect(chatMounts).toBe(1)
    expect(chatUnmounts).toBe(0)

    await user.click(screen.getByRole('button', { name: '返回' }))
    expect(screen.queryByLabelText('个人信息覆盖层')).toBeNull()
    expect(chatMounts).toBe(1)
    expect(chatUnmounts).toBe(0)
  })
})

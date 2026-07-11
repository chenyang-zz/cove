// @vitest-environment jsdom

import { act, cleanup, fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { StoredSession } from '../auth/types'
import type { ChatStreamEvent } from './types'

const mocks = vi.hoisted(() => ({
  listConversations: vi.fn(),
  listMessages: vi.fn(),
  streamChat: vi.fn(),
}))

vi.mock('./api', () => ({
  listConversations: mocks.listConversations,
  listMessages: mocks.listMessages,
  streamChat: mocks.streamChat,
}))

import { ChatScreen } from './ChatScreen'

const session: StoredSession = {
  accessToken: 'access',
  refreshToken: 'refresh',
  user: {
    id: 'user-1',
    username: 'linhai',
    nickname: '林海',
    email: 'linhai@example.com',
    avatar: null,
  },
}

beforeEach(() => {
  vi.restoreAllMocks()
  sessionStorage.clear()
  mocks.listConversations.mockReset().mockResolvedValue({ list: [] })
  mocks.listMessages.mockReset().mockResolvedValue({ list: [] })
  mocks.streamChat.mockReset()
  Element.prototype.scrollIntoView = vi.fn()
  Element.prototype.scrollTo = vi.fn()
  window.scrollTo = vi.fn()
})

afterEach(() => {
  cleanup()
})

describe('ChatScreen', () => {
  it('tracks the visual viewport so the keyboard cannot scroll the whole app away', async () => {
    const listeners = new Map<string, EventListener>()
    const viewport = {
      height: 844,
      width: 390,
      offsetTop: 0,
      addEventListener: vi.fn((type: string, listener: EventListener) => listeners.set(type, listener)),
      removeEventListener: vi.fn((type: string) => listeners.delete(type)),
    }
    vi.stubGlobal('visualViewport', viewport)

    const { container, unmount } = render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const app = container.querySelector<HTMLElement>('.chat-app')
    expect(container.querySelector('.message-scroll')?.classList.contains('message-scroll--empty')).toBe(true)
    expect(app?.dataset.keyboardOpen).toBe('false')
    expect(app?.style.getPropertyValue('--chat-keyboard-height')).toBe('0px')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('0px')

    viewport.height = 516
    viewport.offsetTop = 286
    act(() => listeners.get('resize')?.(new Event('resize')))
    expect(app?.dataset.keyboardOpen).toBe('true')
    expect(app?.style.getPropertyValue('--chat-keyboard-height')).toBe('328px')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('148px')

    viewport.height = 844
    act(() => listeners.get('resize')?.(new Event('resize')))
    expect(app?.dataset.keyboardOpen).toBe('false')
    expect(app?.style.getPropertyValue('--chat-content-shift')).toBe('0px')
    await waitFor(() => {
      expect(Element.prototype.scrollTo).toHaveBeenCalledWith({ top: 0 })
    })

    unmount()
    expect(document.documentElement.classList.contains('chat-document')).toBe(false)
    expect(viewport.removeEventListener).toHaveBeenCalledWith('resize', expect.any(Function))
    vi.unstubAllGlobals()
  })

  it('shows the personalized empty state and toggles the mobile drawer', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)

    expect(await screen.findByRole('heading', { name: '你好，林海' })).toBeTruthy()
    const drawer = screen.getByRole('complementary')
    expect(drawer.classList.contains('chat-drawer--open')).toBe(false)

    await user.click(screen.getByRole('button', { name: '打开会话列表' }))
    expect(drawer.classList.contains('chat-drawer--open')).toBe(true)
    await user.click(screen.getAllByRole('button', { name: '关闭会话列表' })[1])
    expect(drawer.classList.contains('chat-drawer--open')).toBe(false)
  })

  it('closes the account menu when the user clicks outside or presses Escape', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })

    const trigger = screen.getByRole('button', { name: '打开账户菜单' })
    await user.click(trigger)
    expect(screen.getByRole('menu')).toBeTruthy()

    await user.click(screen.getByRole('heading', { name: '你好，林海' }))
    expect(screen.queryByRole('menu')).toBeNull()

    await user.click(trigger)
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('focuses the composer without allowing WKWebView to scroll the page', async () => {
    const user = userEvent.setup()
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const composer = await screen.findByRole('textbox', { name: '发送给 Cove 的消息' })
    const focus = vi.spyOn(composer, 'focus')

    await user.pointer({ target: composer, keys: '[MouseLeft]' })

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('protects form-edge taps when the textarea remains focused after the keyboard closes', async () => {
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    const textarea = await screen.findByRole('textbox', { name: '发送给 Cove 的消息' })
    const form = textarea.closest('form')
    expect(form).toBeTruthy()

    textarea.focus()
    const focus = vi.spyOn(textarea, 'focus')
    fireEvent.pointerDown(form as HTMLFormElement)

    expect(focus).toHaveBeenCalledWith({ preventScroll: true })
  })

  it('loads the latest conversation and its message history', async () => {
    mocks.listConversations.mockResolvedValue({
      list: [
        {
          id: 'conversation-1',
          title: '周末安排',
          is_group: false,
          member_persona_ids: [],
          enable_tools: false,
          created_at: '2026-07-10T08:00:00Z',
          updated_at: '2026-07-11T08:00:00Z',
        },
      ],
    })
    mocks.listMessages.mockResolvedValue({
      list: [
        {
          id: 'message-1',
          role: 'assistant',
          content: '我们可以先安排上午。',
          meta_data: null,
          images: [],
          sender_persona_id: null,
          sender_name: null,
          feedback: null,
          created_at: '2026-07-11T08:01:00Z',
        },
      ],
    })

    render(<ChatScreen session={session} onLogout={vi.fn()} />)

    expect(await screen.findByText('我们可以先安排上午。')).toBeTruthy()
    expect(document.querySelector('.message-scroll')?.classList.contains('message-scroll--empty')).toBe(false)
    expect(mocks.listMessages).toHaveBeenCalledWith('conversation-1')
    expect(screen.getAllByText('周末安排')).toHaveLength(2)
  })

  it('creates a conversation from meta and renders streamed markdown tokens', async () => {
    const user = userEvent.setup()
    mocks.streamChat.mockImplementation(
      async (
        _input: unknown,
        _signal: AbortSignal,
        onEvent: (event: ChatStreamEvent) => void,
      ) => {
        onEvent({ type: 'meta', conversation_id: 'conversation-2', title: '学习计划' })
        onEvent({ type: 'token', text: '**先确定目标**' })
        onEvent({ type: 'done', text: 'message-2' })
      },
    )
    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })

    const composer = screen.getByRole('textbox', { name: '发送给 Cove 的消息' })
    await user.type(composer, '帮我制定学习计划')
    await user.click(screen.getByRole('button', { name: '发送消息' }))

    expect(await screen.findByText('先确定目标')).toHaveProperty('tagName', 'STRONG')
    expect(mocks.streamChat).toHaveBeenCalledWith(
      { message: '帮我制定学习计划' },
      expect.any(AbortSignal),
      expect.any(Function),
    )
    expect(screen.getAllByText('学习计划').length).toBeGreaterThan(0)
    expect(mocks.listMessages).not.toHaveBeenCalled()
  })

  it('blocks duplicate sends while a stream is pending and supports retry after failure', async () => {
    const user = userEvent.setup()
    let emit: ((event: ChatStreamEvent) => void) | undefined
    let finish: (() => void) | undefined
    mocks.streamChat
      .mockImplementationOnce(
        (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          emit = onEvent
          return new Promise<void>((resolve) => {
            finish = resolve
          })
        },
      )
      .mockImplementationOnce(
        async (_input: unknown, _signal: AbortSignal, onEvent: (event: ChatStreamEvent) => void) => {
          onEvent({ type: 'done', text: 'message-retry' })
        },
      )

    render(<ChatScreen session={session} onLogout={vi.fn()} />)
    await screen.findByRole('heading', { name: '你好，林海' })
    const composer = screen.getByRole('textbox', { name: '发送给 Cove 的消息' })
    await user.type(composer, '请再试一次')
    await user.click(screen.getByRole('button', { name: '发送消息' }))

    expect((composer as HTMLTextAreaElement).disabled).toBe(true)
    expect(mocks.streamChat).toHaveBeenCalledTimes(1)
    act(() => {
      emit?.({ type: 'error', content: '服务暂时不可用' })
      finish?.()
    })
    const retry = await screen.findByRole('button', { name: '重新发送' })
    await user.click(retry)

    await waitFor(() => expect(mocks.streamChat).toHaveBeenCalledTimes(2))
  })
})

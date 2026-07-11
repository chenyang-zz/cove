import {
  ArrowClockwise,
  ArrowUp,
  Books,
  DotsThree,
  GlobeHemisphereWest,
  List,
  Paperclip,
  Plus,
  SignOut,
  WarningCircle,
  X,
} from '@phosphor-icons/react'
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type FormEvent,
  type KeyboardEvent,
} from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import type { StoredSession } from '../auth/types'
import { listConversations, listMessages, streamChat } from './api'
import type {
  ChatMessage,
  ChatStreamEvent,
  Conversation,
  ResourceState,
  StreamState,
  ToolActivity,
} from './types'
import './ChatScreen.css'

const coveIcon = '/cove-mark.svg'

type ChatScreenProps = {
  session: StoredSession
  onLogout: () => void
}

function localMessage(role: 'user' | 'assistant', content: string): ChatMessage {
  return {
    id: `local-${role}-${Date.now()}-${Math.random().toString(16).slice(2)}`,
    role,
    content,
    meta_data: null,
    images: [],
    sender_persona_id: null,
    sender_name: null,
    feedback: null,
    created_at: new Date().toISOString(),
    pending: true,
    tools: [],
  }
}

function upsertConversation(items: Conversation[], event: { conversation_id: string; title: string }) {
  const existing = items.find((item) => item.id === event.conversation_id)
  const timestamp = new Date().toISOString()
  const next: Conversation = existing
    ? { ...existing, title: event.title, updated_at: timestamp }
    : {
        id: event.conversation_id,
        title: event.title,
        is_group: false,
        member_persona_ids: [],
        enable_tools: false,
        created_at: timestamp,
        updated_at: timestamp,
      }
  return [next, ...items.filter((item) => item.id !== next.id)]
}

function updateTool(tools: ToolActivity[], event: Extract<ChatStreamEvent, { type: string }>) {
  if (event.type !== 'tool_call' && event.type !== 'tool_result') {
    return tools
  }
  const toolEvent = event as {
    type: 'tool_call' | 'tool_result'
    tool_call_id: string
    tool: string
    error?: string
  }
  const status =
    toolEvent.type === 'tool_call' ? 'running' : toolEvent.error ? 'error' : 'complete'
  const activity: ToolActivity = {
    id: toolEvent.tool_call_id,
    tool: toolEvent.tool,
    status,
  }
  return [...tools.filter((item) => item.id !== activity.id), activity]
}

export function ChatScreen({ session, onLogout }: ChatScreenProps) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [conversationState, setConversationState] = useState<ResourceState>('loading')
  const [conversationError, setConversationError] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [messageState, setMessageState] = useState<ResourceState>('idle')
  const [messageError, setMessageError] = useState('')
  const [streamState, setStreamState] = useState<StreamState>({ status: 'idle' })
  const [draft, setDraft] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [accountOpen, setAccountOpen] = useState(false)
  const abortRef = useRef<AbortController | null>(null)
  const skipHistoryForRef = useRef<string | null>(null)
  const viewportRootRef = useRef<HTMLElement | null>(null)
  const accountMenuRef = useRef<HTMLDivElement | null>(null)
  const messageScrollRef = useRef<HTMLDivElement | null>(null)
  const textareaRef = useRef<HTMLTextAreaElement | null>(null)
  const hasMessagesRef = useRef(false)
  const keyboardHeightRef = useRef(Number(sessionStorage.getItem('cove-keyboard-height')) || 0)
  const keyboardPreparationTimerRef = useRef<number | null>(null)

  const displayName = session.user.nickname || session.user.username
  const activeConversation = conversations.find((item) => item.id === selectedId)
  const isEmptyConversation = messageState === 'ready' && messages.length === 0

  const loadConversations = useCallback(async (selectFirst = false) => {
    setConversationState('loading')
    setConversationError('')
    try {
      const response = await listConversations()
      const sorted = [...response.list].sort(
        (a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at),
      )
      setConversations(sorted)
      setConversationState('ready')
      if (selectFirst && sorted.length > 0) {
        setSelectedId((current) => current ?? sorted[0].id)
      }
    } catch (error: unknown) {
      setConversationState('error')
      setConversationError(error instanceof Error ? error.message : '会话加载失败。')
    }
  }, [])

  const loadHistory = useCallback(async (conversationId: string) => {
    setMessageState('loading')
    setMessageError('')
    try {
      const response = await listMessages(conversationId)
      setMessages(response.list)
      setMessageState('ready')
    } catch (error: unknown) {
      setMessageState('error')
      setMessageError(error instanceof Error ? error.message : '消息加载失败。')
    }
  }, [])

  useEffect(() => {
    void loadConversations(true)
  }, [loadConversations])

  useEffect(() => {
    if (!selectedId) {
      setMessages([])
      setMessageState('ready')
      return
    }
    if (skipHistoryForRef.current === selectedId) {
      skipHistoryForRef.current = null
      setMessageState('ready')
      return
    }
    void loadHistory(selectedId)
  }, [loadHistory, selectedId])

  useEffect(() => {
    hasMessagesRef.current = messages.length > 0
    const messageScroll = messageScrollRef.current
    messageScroll?.scrollTo({ top: messageScroll.scrollHeight, behavior: 'smooth' })
  }, [messages])

  useEffect(() => {
    return () => {
      abortRef.current?.abort()
      if (keyboardPreparationTimerRef.current !== null) {
        window.clearTimeout(keyboardPreparationTimerRef.current)
      }
    }
  }, [])

  useEffect(() => {
    if (!accountOpen) {
      return
    }

    function closeAccountMenu(event: PointerEvent) {
      if (!accountMenuRef.current?.contains(event.target as Node)) {
        setAccountOpen(false)
      }
    }

    function closeAccountMenuWithEscape(event: globalThis.KeyboardEvent) {
      if (event.key === 'Escape') {
        setAccountOpen(false)
      }
    }

    document.addEventListener('pointerdown', closeAccountMenu)
    document.addEventListener('keydown', closeAccountMenuWithEscape)
    return () => {
      document.removeEventListener('pointerdown', closeAccountMenu)
      document.removeEventListener('keydown', closeAccountMenuWithEscape)
    }
  }, [accountOpen])

  useLayoutEffect(() => {
    document.documentElement.classList.add('chat-document')
    window.scrollTo(0, 0)
    return () => document.documentElement.classList.remove('chat-document')
  }, [])

  useLayoutEffect(() => {
    const root = viewportRootRef.current
    const viewport = window.visualViewport
    if (!root || !viewport) {
      return
    }
    const activeRoot = root
    const activeViewport = viewport
    let layoutHeight = Math.max(window.innerHeight, activeViewport.height)
    let layoutWidth = activeViewport.width

    function syncVisualViewport() {
      const widthChanged = Math.abs(activeViewport.width - layoutWidth) > 1
      if (widthChanged) {
        layoutHeight = activeViewport.height
        layoutWidth = activeViewport.width
      }

      layoutHeight = Math.max(layoutHeight, window.innerHeight, activeViewport.height)
      const keyboardHeight = Math.max(0, layoutHeight - activeViewport.height)
      const contentShift = Math.round(Math.min(160, keyboardHeight * 0.45))
      const keyboardOpen = keyboardHeight > 20
      if (!keyboardOpen && activeViewport.height > layoutHeight) {
        layoutHeight = activeViewport.height
      }

      activeRoot.style.setProperty('--chat-keyboard-height', `${keyboardHeight}px`)
      activeRoot.style.setProperty('--chat-content-shift', `${contentShift}px`)
      activeRoot.dataset.keyboardOpen = String(keyboardOpen)
      if (keyboardOpen) {
        keyboardHeightRef.current = keyboardHeight
        sessionStorage.setItem('cove-keyboard-height', String(keyboardHeight))
        if (keyboardPreparationTimerRef.current !== null) {
          window.clearTimeout(keyboardPreparationTimerRef.current)
          keyboardPreparationTimerRef.current = null
        }
        window.requestAnimationFrame(() => {
          const messageScroll = messageScrollRef.current
          messageScroll?.scrollTo({ top: messageScroll.scrollHeight })
        })
      } else if (!hasMessagesRef.current) {
        window.requestAnimationFrame(() => {
          messageScrollRef.current?.scrollTo({ top: 0 })
        })
      }
    }

    syncVisualViewport()
    activeViewport.addEventListener('resize', syncVisualViewport)
    return () => {
      activeViewport.removeEventListener('resize', syncVisualViewport)
    }
  }, [])

  function focusComposerWithoutScroll(textarea: HTMLTextAreaElement) {
    const root = viewportRootRef.current
    const viewport = window.visualViewport
    if (root && viewport && viewport.width < 900) {
      const anticipatedHeight =
        keyboardHeightRef.current || Math.min(360, Math.max(260, window.innerHeight * 0.38))
      const anticipatedContentShift = Math.round(Math.min(160, anticipatedHeight * 0.45))
      const heightBeforeFocus = viewport.height
      root.style.setProperty('--chat-keyboard-height', `${anticipatedHeight}px`)
      root.dataset.keyboardOpen = 'true'
      void root.offsetHeight

      if (keyboardPreparationTimerRef.current !== null) {
        window.clearTimeout(keyboardPreparationTimerRef.current)
      }
      keyboardPreparationTimerRef.current = window.setTimeout(() => {
        if (viewport.height >= heightBeforeFocus - 20) {
          root.style.setProperty('--chat-keyboard-height', '0px')
          root.style.setProperty('--chat-content-shift', '0px')
          root.dataset.keyboardOpen = 'false'
        }
        keyboardPreparationTimerRef.current = null
      }, 650)
      window.requestAnimationFrame(() => {
        root.style.setProperty('--chat-content-shift', `${anticipatedContentShift}px`)
      })
    }
    textarea.focus({ preventScroll: true })
  }

  function handleComposerSurfacePress(event: {
    target: EventTarget | null
    preventDefault: () => void
  }) {
    const root = viewportRootRef.current
    const textarea = textareaRef.current
    const target = event.target
    if (
      !root ||
      !textarea ||
      !(target instanceof Element) ||
      target.closest('button') ||
      root.dataset.keyboardOpen === 'true'
    ) {
      return
    }

    event.preventDefault()
    focusComposerWithoutScroll(textarea)
  }

  function startNewConversation() {
    abortRef.current?.abort()
    abortRef.current = null
    setSelectedId(null)
    setMessages([])
    setMessageState('ready')
    setMessageError('')
    setStreamState({ status: 'idle' })
    setDrawerOpen(false)
    window.requestAnimationFrame(() => {
      const textarea = textareaRef.current
      if (textarea) {
        focusComposerWithoutScroll(textarea)
      }
    })
  }

  function selectConversation(conversationId: string) {
    if (conversationId === selectedId) {
      setDrawerOpen(false)
      return
    }
    abortRef.current?.abort()
    abortRef.current = null
    setStreamState({ status: 'idle' })
    setSelectedId(conversationId)
    setDrawerOpen(false)
  }

  function updateAssistant(id: string, updater: (message: ChatMessage) => ChatMessage) {
    setMessages((current) => current.map((message) => (message.id === id ? updater(message) : message)))
  }

  async function sendMessage(prompt = draft) {
    const text = prompt.trim()
    if (!text || streamState.status === 'streaming') {
      return
    }

    const conversationIdAtSend = selectedId
    const userMessage = localMessage('user', text)
    const assistantMessage = localMessage('assistant', '')
    const controller = new AbortController()
    abortRef.current = controller
    setDraft('')
    setMessageError('')
    setMessageState('ready')
    setStreamState({ status: 'streaming' })
    setMessages((current) => [...current, userMessage, assistantMessage])
    let reachedTerminalEvent = false

    try {
      await streamChat(
        {
          ...(conversationIdAtSend ? { conversation_id: conversationIdAtSend } : {}),
          message: text,
        },
        controller.signal,
        (event) => {
          if (event.type === 'meta') {
            const meta = event as { type: 'meta'; conversation_id: string; title: string }
            if (meta.conversation_id !== conversationIdAtSend) {
              skipHistoryForRef.current = meta.conversation_id
              setSelectedId(meta.conversation_id)
            }
            setConversations((current) => upsertConversation(current, meta))
            return
          }
          if (event.type === 'token') {
            const token = event as { type: 'token'; text: string }
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              content: message.content + token.text,
            }))
            return
          }
          if (event.type === 'tool_call' || event.type === 'tool_result') {
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              tools: updateTool(message.tools ?? [], event),
            }))
            return
          }
          if (event.type === 'done') {
            reachedTerminalEvent = true
            const done = event as { type: 'done'; text: string }
            updateAssistant(assistantMessage.id, (message) => ({
              ...message,
              id: done.text || message.id,
              pending: false,
            }))
            setMessages((current) =>
              current.map((message) =>
                message.id === userMessage.id ? { ...message, pending: false } : message,
              ),
            )
            setStreamState({ status: 'idle' })
            return
          }
          if (event.type === 'error') {
            reachedTerminalEvent = true
            const streamError = event as { type: 'error'; content: string }
            updateAssistant(assistantMessage.id, (message) => ({ ...message, pending: false }))
            setStreamState({ status: 'error', message: streamError.content, prompt: text })
            return
          }
          if (import.meta.env.DEV) {
            console.debug('Ignored chat stream event', event)
          }
        },
      )
      if (!reachedTerminalEvent && !controller.signal.aborted) {
        updateAssistant(assistantMessage.id, (item) => ({ ...item, pending: false }))
        setStreamState({
          status: 'error',
          message: '消息流提前结束，请重新发送。',
          prompt: text,
        })
      }
    } catch (error: unknown) {
      if (!controller.signal.aborted) {
        const message = error instanceof Error ? error.message : '回复中断，请稍后重试。'
        updateAssistant(assistantMessage.id, (item) => ({ ...item, pending: false }))
        setStreamState({ status: 'error', message, prompt: text })
      }
    } finally {
      if (abortRef.current === controller) {
        abortRef.current = null
      }
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void sendMessage()
  }

  function handleComposerKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault()
      void sendMessage()
    }
  }

  function handleDraftChange(value: string) {
    setDraft(value)
    const textarea = textareaRef.current
    if (textarea) {
      textarea.style.height = 'auto'
      textarea.style.height = `${Math.min(textarea.scrollHeight, 132)}px`
    }
  }

  return (
    <main className="chat-app" ref={viewportRootRef}>
      <button
        className={drawerOpen ? 'chat-drawer-scrim chat-drawer-scrim--visible' : 'chat-drawer-scrim'}
        type="button"
        aria-label="关闭会话列表"
        tabIndex={drawerOpen ? 0 : -1}
        onClick={() => setDrawerOpen(false)}
      />

      <aside className={drawerOpen ? 'chat-drawer chat-drawer--open' : 'chat-drawer'}>
        <header className="chat-drawer__header">
          <div className="brand-lockup">
            <img className="brand-lockup__icon" src={coveIcon} alt="" />
            <span>Cove</span>
          </div>
          <button className="icon-button chat-drawer__close" type="button" aria-label="关闭会话列表" onClick={() => setDrawerOpen(false)}>
            <X size={20} weight="bold" />
          </button>
        </header>

        <button className="new-chat-button" type="button" onClick={startNewConversation}>
          <Plus size={18} weight="bold" />
          <span>新对话</span>
        </button>

        <div className="conversation-list" aria-label="历史会话">
          <p className="conversation-list__label">最近对话</p>
          {conversationState === 'loading' && (
            <div className="conversation-skeleton" aria-label="正在加载会话">
              <span />
              <span />
              <span />
            </div>
          )}
          {conversationState === 'error' && (
            <div className="drawer-error" role="alert">
              <p>{conversationError}</p>
              <button type="button" onClick={() => void loadConversations(false)}>
                <ArrowClockwise size={16} /> 重试
              </button>
            </div>
          )}
          {conversationState === 'ready' && conversations.length === 0 && (
            <p className="conversation-list__empty">发送第一条消息后，会话会保存在这里。</p>
          )}
          {conversations.map((conversation) => (
            <button
              className={conversation.id === selectedId ? 'conversation-row conversation-row--active' : 'conversation-row'}
              type="button"
              key={conversation.id}
              onClick={() => selectConversation(conversation.id)}
            >
              <span>{conversation.title || '新对话'}</span>
              <time dateTime={conversation.updated_at}>
                {new Intl.DateTimeFormat('zh-CN', { month: 'numeric', day: 'numeric' }).format(
                  new Date(conversation.updated_at),
                )}
              </time>
            </button>
          ))}
        </div>

        <div className="drawer-account">
          <span className="drawer-account__avatar">{displayName.slice(0, 1).toUpperCase()}</span>
          <span className="drawer-account__name">{displayName}</span>
        </div>
      </aside>

      <section className="chat-workspace">
        <header className="chat-header">
          <button className="icon-button chat-header__menu" type="button" aria-label="打开会话列表" onClick={() => { setAccountOpen(false); setDrawerOpen(true) }}>
            <List size={22} />
          </button>
          <div className="chat-header__title">
            <strong>{activeConversation?.title || '新对话'}</strong>
            <span>{streamState.status === 'streaming' ? 'Cove 正在回复' : 'Cove AI'}</span>
          </div>
          <div className="account-menu" ref={accountMenuRef}>
            <button className="icon-button" type="button" aria-label="打开账户菜单" aria-expanded={accountOpen} onClick={() => setAccountOpen((open) => !open)}>
              <DotsThree size={24} weight="bold" />
            </button>
            {accountOpen && (
              <div className="account-menu__popover" role="menu">
                <div>
                  <strong>{displayName}</strong>
                  <span>{session.user.email || `@${session.user.username}`}</span>
                </div>
                <button type="button" onClick={onLogout}>
                  <SignOut size={18} />
                  退出登录
                </button>
              </div>
            )}
          </div>
        </header>

        <div
          className={isEmptyConversation ? 'message-scroll message-scroll--empty' : 'message-scroll'}
          ref={messageScrollRef}
          aria-busy={messageState === 'loading'}
        >
          <div className="message-column" role="log" aria-live="polite" aria-relevant="additions text">
            {messageState === 'loading' && (
              <div className="message-skeleton" aria-label="正在加载消息">
                <span />
                <span />
                <span />
              </div>
            )}
            {messageState === 'error' && (
              <div className="message-error" role="alert">
                <WarningCircle size={24} />
                <p>{messageError}</p>
                {selectedId && (
                  <button type="button" onClick={() => void loadHistory(selectedId)}>
                    重新加载
                  </button>
                )}
              </div>
            )}
            {messageState === 'ready' && messages.length === 0 && (
              <div className="chat-empty">
                <img src={coveIcon} alt="" />
                <h1>你好，{displayName}</h1>
                <p>把正在思考的事告诉我，我们一起理清。</p>
                <div className="prompt-suggestions">
                  <button type="button" onClick={() => handleDraftChange('帮我把今天最重要的三件事理清楚')}>
                    梳理今天的重点
                  </button>
                  <button type="button" onClick={() => handleDraftChange('帮我制定一个可执行的学习计划')}>
                    制定学习计划
                  </button>
                </div>
              </div>
            )}
            {messages.map((message) => (
              <article className={`message message--${message.role}`} key={message.id}>
                {message.role === 'assistant' && (
                  <img className="message__avatar" src={coveIcon} alt="Cove" />
                )}
                <div className="message__body">
                  {message.tools && message.tools.length > 0 && (
                    <div className="tool-activity">
                      {message.tools.map((tool) => (
                        <span className={`tool-activity__item tool-activity__item--${tool.status}`} key={tool.id}>
                          {tool.status === 'running' ? '正在使用' : tool.status === 'error' ? '工具失败' : '已使用'} {tool.tool}
                        </span>
                      ))}
                    </div>
                  )}
                  {message.role === 'assistant' ? (
                    message.content ? (
                      <ReactMarkdown
                        remarkPlugins={[remarkGfm]}
                        skipHtml
                        components={{
                          a: ({ href, children }) => (
                            <a href={href} target="_blank" rel="noreferrer noopener">
                              {children}
                            </a>
                          ),
                        }}
                      >
                        {message.content}
                      </ReactMarkdown>
                    ) : (
                      message.pending && <span className="thinking-indicator" aria-label="Cove 正在思考"><i /><i /><i /></span>
                    )
                  ) : (
                    <p>{message.content}</p>
                  )}
                  {message.role === 'assistant' && message.pending && message.content && (
                    <span className="stream-cursor" aria-hidden="true" />
                  )}
                </div>
              </article>
            ))}
            {streamState.status === 'error' && (
              <div className="stream-error" role="alert">
                <WarningCircle size={19} />
                <span>{streamState.message}</span>
                <button type="button" onClick={() => void sendMessage(streamState.prompt)}>
                  重新发送
                </button>
              </div>
            )}
            <div />
          </div>
        </div>

        <footer className="composer-area">
          <form
            className="composer"
            onSubmit={handleSubmit}
            onTouchStartCapture={handleComposerSurfacePress}
            onPointerDownCapture={handleComposerSurfacePress}
          >
            <textarea
              ref={textareaRef}
              rows={1}
              value={draft}
              placeholder="问问 Cove..."
              aria-label="发送给 Cove 的消息"
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="sentences"
              spellCheck={false}
              enterKeyHint="send"
              disabled={streamState.status === 'streaming'}
              onTouchStart={(event) => {
                if (document.activeElement !== event.currentTarget) {
                  event.preventDefault()
                  focusComposerWithoutScroll(event.currentTarget)
                }
              }}
              onPointerDown={(event) => {
                if (document.activeElement !== event.currentTarget) {
                  event.preventDefault()
                  focusComposerWithoutScroll(event.currentTarget)
                }
              }}
              onFocus={() => setAccountOpen(false)}
              onChange={(event) => handleDraftChange(event.target.value)}
              onKeyDown={handleComposerKeyDown}
            />
            <div className="composer__toolbar">
              <div>
                <button className="composer-tool" type="button" disabled aria-label="添加附件，暂未开放" title="附件功能即将开放">
                  <Paperclip size={18} />
                </button>
                <button className="composer-tool" type="button" disabled aria-label="知识库，暂未开放" title="知识库功能即将开放">
                  <Books size={18} />
                </button>
                <button className="composer-tool" type="button" disabled aria-label="联网搜索，暂未开放" title="联网搜索功能即将开放">
                  <GlobeHemisphereWest size={18} />
                </button>
              </div>
              <button className="send-button" type="submit" aria-label="发送消息" disabled={!draft.trim() || streamState.status === 'streaming'}>
                <ArrowUp size={19} weight="bold" />
              </button>
            </div>
          </form>
          <p>Cove 可能会出错，请核对重要信息。</p>
        </footer>
      </section>
    </main>
  )
}

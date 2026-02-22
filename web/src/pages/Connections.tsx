// web/src/pages/Connections.tsx
import { useEffect, useState } from 'react'
import { Loader2, Plus, Trash2, Link2, X } from 'lucide-react'
import { Header } from '@/shared/ui/Header'
import { listConnections, createConnection, deleteConnection } from '@/shared/api'
import type { Connection, ConnectionCreate, ConnectionType } from '@/shared/types'

const typeLabels: Record<ConnectionType, string> = {
  telegram: 'Telegram',
  slack: 'Slack',
  http: 'HTTP',
  smtp: 'SMTP',
}

const typeBadgeClass: Record<ConnectionType, string> = {
  telegram: 'bg-info/10 text-info',
  slack:    'bg-success/10 text-success',
  http:     'bg-warning/10 text-warning',
  smtp:     'bg-muted text-muted-foreground',
}

const emptyForm = (): ConnectionCreate => ({ name: '', type: 'telegram' })

export default function Connections() {
  const [connections, setConnections] = useState<Connection[]>([])
  const [loading, setLoading] = useState(true)
  const [open, setOpen] = useState(false)
  const [form, setForm] = useState<ConnectionCreate>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const reload = async () => {
    try {
      const list = await listConnections()
      setConnections(list ?? [])
    } catch {
      setConnections([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { reload() }, [])

  const handleDelete = async (id: string) => {
    if (!confirm('이 Connection을 삭제할까요?')) return
    await deleteConnection(id)
    reload()
  }

  const handleSubmit = async () => {
    if (!form.name.trim()) { setError('이름을 입력하세요.'); return }
    setSaving(true)
    setError('')
    try {
      await createConnection(form)
      setOpen(false)
      setForm(emptyForm())
      reload()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '저장에 실패했습니다.')
    } finally {
      setSaving(false)
    }
  }

  const set = (key: keyof ConnectionCreate, value: string | number) =>
    setForm((f) => ({ ...f, [key]: value }))

  const setExtra = (key: string, value: string) =>
    setForm((f) => ({ ...f, extras: { ...f.extras, [key]: value } }))

  return (
    <div className="h-screen flex flex-col bg-background text-foreground">
      <Header />

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto px-6 py-6 space-y-6">
          <div className="flex items-center justify-between">
            <h1 className="text-xl font-semibold">Connections</h1>
            <button
              onClick={() => { setOpen(true); setError(''); setForm(emptyForm()) }}
              className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-lg bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              New Connection
            </button>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-20">
              <Loader2 className="animate-spin text-muted-foreground" size={32} />
            </div>
          ) : connections.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-20 gap-3 text-muted-foreground">
              <Link2 size={40} strokeWidth={1.2} />
              <p className="text-sm">등록된 Connection이 없습니다.</p>
              <button
                onClick={() => { setOpen(true); setError(''); setForm(emptyForm()) }}
                className="text-xs text-primary hover:underline"
              >
                첫 번째 Connection 만들기
              </button>
            </div>
          ) : (
            <div className="space-y-2">
              {connections.map((conn) => (
                <div
                  key={conn.id}
                  className="flex items-center justify-between px-4 py-3 rounded-lg border border-border bg-card"
                >
                  <div className="flex items-center gap-3">
                    <span className={`text-[10px] font-medium px-2 py-0.5 rounded-full ${typeBadgeClass[conn.type]}`}>
                      {typeLabels[conn.type]}
                    </span>
                    <div>
                      <p className="text-sm font-medium">{conn.name}</p>
                      <p className="text-xs text-muted-foreground font-mono">{conn.id}</p>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(conn.id)}
                    className="p-1.5 rounded hover:bg-muted transition-colors text-muted-foreground hover:text-destructive"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Create dialog */}
      {open && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-card border border-border rounded-xl shadow-xl w-full max-w-md mx-4 p-6 space-y-4">
            <div className="flex items-center justify-between">
              <h2 className="text-base font-semibold">New Connection</h2>
              <button onClick={() => setOpen(false)} className="p-1 rounded hover:bg-muted transition-colors">
                <X className="h-4 w-4" />
              </button>
            </div>

            {/* Name */}
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">이름</label>
              <input
                type="text"
                value={form.name}
                onChange={(e) => set('name', e.target.value)}
                placeholder="내 텔레그램 봇"
                className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
              />
            </div>

            {/* Type */}
            <div className="space-y-1">
              <label className="text-xs text-muted-foreground">타입</label>
              <select
                value={form.type}
                onChange={(e) => setForm((f) => ({ ...emptyForm(), name: f.name, type: e.target.value as ConnectionType }))}
                className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
              >
                {(Object.keys(typeLabels) as ConnectionType[]).map((t) => (
                  <option key={t} value={t}>{typeLabels[t]}</option>
                ))}
              </select>
            </div>

            {/* Type-specific fields */}
            {form.type === 'telegram' && (
              <>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Bot Token</label>
                  <input
                    type="password"
                    value={form.token ?? ''}
                    onChange={(e) => set('token', e.target.value)}
                    placeholder="7123456789:AAF-xxxxx"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Chat ID</label>
                  <input
                    type="text"
                    value={form.extras?.chat_id ?? ''}
                    onChange={(e) => setExtra('chat_id', e.target.value)}
                    placeholder="123456789"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                  <p className="text-[10px] text-muted-foreground">
                    봇에게 /start 후 getUpdates로 확인
                  </p>
                </div>
              </>
            )}

            {form.type === 'slack' && (
              <>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Webhook URL</label>
                  <input
                    type="password"
                    value={form.extras?.webhook_url ?? ''}
                    onChange={(e) => setExtra('webhook_url', e.target.value)}
                    placeholder="https://hooks.slack.com/services/..."
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                  <p className="text-[10px] text-muted-foreground">
                    Slack 앱 → Incoming Webhooks에서 발급
                  </p>
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Channel (선택)</label>
                  <input
                    type="text"
                    value={form.extras?.channel ?? ''}
                    onChange={(e) => setExtra('channel', e.target.value)}
                    placeholder="#general"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </>
            )}

            {form.type === 'smtp' && (
              <>
                <div className="grid grid-cols-3 gap-2">
                  <div className="col-span-2 space-y-1">
                    <label className="text-xs text-muted-foreground">Host</label>
                    <input
                      type="text"
                      value={form.host ?? ''}
                      onChange={(e) => set('host', e.target.value)}
                      placeholder="smtp.gmail.com"
                      className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                  <div className="space-y-1">
                    <label className="text-xs text-muted-foreground">Port</label>
                    <input
                      type="number"
                      value={form.port ?? 587}
                      onChange={(e) => set('port', Number(e.target.value))}
                      className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                    />
                  </div>
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Email</label>
                  <input
                    type="text"
                    value={form.login ?? ''}
                    onChange={(e) => set('login', e.target.value)}
                    placeholder="you@gmail.com"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Password / App Password</label>
                  <input
                    type="password"
                    value={form.password ?? ''}
                    onChange={(e) => set('password', e.target.value)}
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">수신 이메일</label>
                  <input
                    type="text"
                    value={form.extras?.to ?? ''}
                    onChange={(e) => setExtra('to', e.target.value)}
                    placeholder="recipient@example.com"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary"
                  />
                </div>
              </>
            )}

            {form.type === 'http' && (
              <>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Webhook URL</label>
                  <input
                    type="text"
                    value={form.host ?? ''}
                    onChange={(e) => set('host', e.target.value)}
                    placeholder="https://your-server.com/webhook"
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                </div>
                <div className="space-y-1">
                  <label className="text-xs text-muted-foreground">Bearer Token (선택)</label>
                  <input
                    type="password"
                    value={form.token ?? ''}
                    onChange={(e) => set('token', e.target.value)}
                    className="w-full text-sm border border-border rounded-md px-3 py-1.5 bg-background outline-none focus:ring-1 focus:ring-primary font-mono"
                  />
                </div>
              </>
            )}

            {error && <p className="text-xs text-destructive">{error}</p>}

            <div className="flex justify-end gap-2 pt-2">
              <button
                onClick={() => setOpen(false)}
                className="px-3 py-1.5 text-sm rounded-md border border-border hover:bg-muted transition-colors"
              >
                취소
              </button>
              <button
                onClick={handleSubmit}
                disabled={saving}
                className="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium rounded-md bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
              >
                {saving && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
                저장
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

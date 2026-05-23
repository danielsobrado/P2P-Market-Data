import { useState } from 'react'
import {
  Code,
  Play,
  Square,
  Trash2,
  Upload,
  Eye,
  RefreshCw,
} from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import { formatBytes, formatRelativeTime, scriptStatusKind } from '@/lib/terminal/formatters'
import { EmptyState, Panel, StatusBadge } from '../TerminalComponents'

export function ScriptsView({ data }: { data: TerminalData }) {
  const {
    scripts,
    refreshScripts,
    runScript,
    stopScript,
    deleteScript,
    getScriptContent,
    uploadScript,
  } = data
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [content, setContent] = useState<string | null>(null)
  const [uploadOpen, setUploadOpen] = useState(false)
  const [busy, setBusy] = useState<string | null>(null)

  const viewCode = async (id: string) => {
    setSelectedId(id)
    const text = await getScriptContent(id)
    setContent(text)
  }

  const act = async (id: string, fn: () => Promise<void>) => {
    setBusy(id)
    try {
      await fn()
    } finally {
      setBusy(null)
    }
  }

  return (
    <div style={{ padding: 8, flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Panel
        title="Script Registry"
        tag="PY"
        sub={`${scripts.length} scripts`}
        flush
        actions={
          <>
            <button className="btn sm" onClick={() => refreshScripts()}>
              <RefreshCw size={11} /> Refresh
            </button>
            <button className="btn sm primary" onClick={() => setUploadOpen(true)}>
              <Upload size={11} /> Upload
            </button>
          </>
        }
        style={{ flex: 1, minHeight: 280 }}
      >
        {scripts.length > 0 ? (
          <table className="dense-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Author</th>
                <th>Version</th>
                <th className="num">Size</th>
                <th>Updated</th>
                <th>Status</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {scripts.map((s) => (
                <tr key={s.id} className={selectedId === s.id ? 'selected' : ''}>
                  <td className="bright">{s.name}</td>
                  <td className="dim">{s.author || '—'}</td>
                  <td className="dim">{s.version || '—'}</td>
                  <td className="num dim">{formatBytes(s.size)}</td>
                  <td className="dim">{formatRelativeTime(s.updated)}</td>
                  <td>
                    <StatusBadge kind={scriptStatusKind(s.status)}>{s.status}</StatusBadge>
                  </td>
                  <td style={{ textAlign: 'right' }}>
                    <div style={{ display: 'inline-flex', gap: 4 }}>
                      <button
                        className="btn sm icon"
                        title="View code"
                        onClick={() => viewCode(s.id)}
                        disabled={busy === s.id}
                      >
                        <Eye size={11} />
                      </button>
                      {s.status === 'running' ? (
                        <button
                          className="btn sm icon"
                          title="Stop"
                          onClick={() => act(s.id, () => stopScript(s.id))}
                          disabled={busy === s.id}
                        >
                          <Square size={11} />
                        </button>
                      ) : (
                        <button
                          className="btn sm icon"
                          title="Run"
                          onClick={() => act(s.id, () => runScript(s.id))}
                          disabled={busy === s.id}
                        >
                          <Play size={11} />
                        </button>
                      )}
                      <button
                        className="btn sm icon danger"
                        title="Delete"
                        onClick={() => act(s.id, () => deleteScript(s.id))}
                        disabled={busy === s.id}
                      >
                        <Trash2 size={11} />
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState
            icon={<Code size={28} />}
            title="No scripts installed"
            hint="Upload a Python script to get started"
          />
        )}
      </Panel>

      {content !== null && (
        <Panel title="Script Source" tag={selectedId?.slice(0, 8) ?? ''} style={{ maxHeight: 240 }}>
          <pre className="code-view">{content}</pre>
        </Panel>
      )}

      {uploadOpen && (
        <ScriptUploadModal
          onClose={() => setUploadOpen(false)}
          onUpload={async (payload) => {
            await uploadScript(payload)
            setUploadOpen(false)
          }}
        />
      )}
    </div>
  )
}

function ScriptUploadModal({
  onClose,
  onUpload,
}: {
  onClose: () => void
  onUpload: (data: { name: string; content: string; description?: string; author?: string; version?: string }) => Promise<void>
}) {
  const [name, setName] = useState('script.py')
  const [content, setContent] = useState('# P2P market data script\n')
  const [description, setDescription] = useState('')
  const [author, setAuthor] = useState('')
  const [version, setVersion] = useState('1.0.0')
  const [loading, setLoading] = useState(false)

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'grid',
        placeItems: 'center',
        zIndex: 100,
      }}
      onClick={onClose}
    >
      <div className="panel" style={{ width: 520 }} onClick={(e) => e.stopPropagation()}>
        <div className="panel-head">
          <span className="panel-title">Upload Script</span>
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="field">
            <label className="lbl">Filename</label>
            <input className="input" value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="field">
            <label className="lbl">Description</label>
            <input className="input" value={description} onChange={(e) => setDescription(e.target.value)} />
          </div>
          <div className="field">
            <label className="lbl">Content</label>
            <textarea
              className="input"
              style={{ height: 160, padding: 8, resize: 'vertical' }}
              value={content}
              onChange={(e) => setContent(e.target.value)}
            />
          </div>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
            <button className="btn ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              className="btn primary"
              disabled={loading}
              onClick={async () => {
                setLoading(true)
                try {
                  await onUpload({ name, content, description, author, version })
                } finally {
                  setLoading(false)
                }
              }}
            >
              Upload
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

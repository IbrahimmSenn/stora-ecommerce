import { useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { api, ApiError, type AuditEntry } from '../lib/api'

export function AdminAuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .adminListAudit()
      .then((res) => {
        setEntries(res.entries)
        setTotal(res.total)
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : 'Could not load the audit log.'))
      .finally(() => setLoading(false))
  }, [])

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="Accountability"
        title="Audit log."
        caption="Every staff mutation, with who did it and the result."
      />

      {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}

      <div className="flex items-baseline justify-between mb-6">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Recent actions</span>
        <span className="uc-tight text-[0.7rem] text-ink-faint tnum">{total} total</span>
      </div>

      {loading ? (
        <p className="text-sm text-ink-soft">Loading.</p>
      ) : entries.length === 0 ? (
        <p className="text-sm text-ink-faint">No actions recorded yet.</p>
      ) : (
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b border-rule-strong text-left">
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">When</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Actor</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Action</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Target</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 text-right">Status</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-rule">
            {entries.map((e) => (
              <tr key={e.id}>
                <td className="py-2.5 pr-4 text-ink-faint tnum whitespace-nowrap">
                  {new Date(e.occurred_at).toLocaleString()}
                </td>
                <td className="py-2.5 pr-4 text-ink-soft break-all">
                  {e.actor_email ?? '—'}
                  {e.actor_role && (
                    <span className="uc-tight text-[0.65rem] text-ink-faint ml-2">{e.actor_role}</span>
                  )}
                </td>
                <td className="py-2.5 pr-4 tnum text-ink">{e.action}</td>
                <td className="py-2.5 pr-4 text-ink-soft break-all">{e.target}</td>
                <td className="py-2.5 text-right tnum text-ink">{e.status_code}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Page>
  )
}

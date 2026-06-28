import { useCallback, useEffect, useState } from 'react'
import { Page } from '../components/Page'
import { Masthead } from '../components/Masthead'
import { api, ApiError, type AdminUser, type UserRole } from '../lib/api'

const ROLES: UserRole[] = ['admin', 'support', 'sales', 'customer']

export function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUser[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [savingId, setSavingId] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)

  const refresh = useCallback(() => {
    setLoading(true)
    setError(null)
    api
      .adminListUsers()
      .then((res) => {
        setUsers(res.users)
        setTotal(res.total)
      })
      .catch((e) => setError(e instanceof ApiError ? e.message : 'Could not load users.'))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    refresh()
  }, [refresh])

  async function changeRole(id: string, role: UserRole) {
    setSavingId(id)
    setNotice(null)
    try {
      await api.adminSetUserRole(id, role)
      setUsers((prev) => prev.map((u) => (u.id === id ? { ...u, role } : u)))
      setNotice('Role updated.')
    } catch (e) {
      setNotice(e instanceof ApiError ? e.message : 'Could not update role.')
      refresh()
    } finally {
      setSavingId(null)
    }
  }

  return (
    <Page width="max-w-5xl">
      <Masthead
        eyebrow="People"
        title="Users."
        caption="Every account and the role it holds. Least privilege — grant only what's needed."
      />

      {error && <p className="text-sm text-accent mb-6" role="alert">{error}</p>}
      {notice && <p className="text-sm text-ink-soft mb-6" role="status">{notice}</p>}

      <div className="flex items-baseline justify-between mb-6">
        <span className="uc-tight text-[0.7rem] text-ink-faint">Accounts</span>
        <span className="uc-tight text-[0.7rem] text-ink-faint tnum">{total} total</span>
      </div>

      {loading ? (
        <p className="text-sm text-ink-soft">Loading.</p>
      ) : (
        <table className="w-full text-sm border-collapse">
          <thead>
            <tr className="border-b border-rule-strong text-left">
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Email</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Joined</th>
              <th className="uc-tight text-[0.7rem] text-ink-faint font-normal py-2 pr-4">Role</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-rule">
            {users.map((u) => (
              <tr key={u.id}>
                <td className="py-3 pr-4 text-ink break-all">{u.email}</td>
                <td className="py-3 pr-4 text-ink-faint tnum">
                  {new Date(u.created_at).toLocaleDateString()}
                </td>
                <td className="py-3 pr-4">
                  <select
                    value={u.role}
                    disabled={savingId === u.id}
                    onChange={(e) => changeRole(u.id, e.target.value as UserRole)}
                    className="bg-raised border-0 border-b border-rule-strong focus:border-ink px-0 py-1 text-ink transition-colors cursor-pointer disabled:opacity-50"
                    style={{ borderRadius: 0 }}
                  >
                    {ROLES.map((r) => (
                      <option key={r} value={r}>{r}</option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </Page>
  )
}

import { useRef, useState } from 'react'
import { Button } from '../components/Button'
import { api, ApiError, type BulkUploadResult } from '../lib/api'

const JSON_EXAMPLE = `[
  { "name": "Example A", "price": 1999, "stock_quantity": 10 },
  { "name": "Example B", "price": 4500, "stock_quantity": 3, "description": "..." }
]`

/* BulkUploadPanel — JSON paste or CSV file upload for products. Prices are in
 * cents (matching the API). Per-row failures come back in the result and are
 * shown without discarding the rows that succeeded. */
export function BulkUploadPanel({ onUploaded }: { onUploaded: () => void }) {
  const [mode, setMode] = useState<'json' | 'csv'>('json')
  const [jsonText, setJsonText] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [result, setResult] = useState<BulkUploadResult | null>(null)
  const fileRef = useRef<HTMLInputElement>(null)

  function report(res: BulkUploadResult) {
    setResult(res)
    if (res.created > 0) onUploaded()
  }

  async function submitJSON() {
    setError(null)
    setResult(null)
    let parsed: unknown
    try {
      parsed = JSON.parse(jsonText)
    } catch {
      setError('That is not valid JSON.')
      return
    }
    if (!Array.isArray(parsed)) {
      setError('JSON must be an array of product objects.')
      return
    }
    setBusy(true)
    try {
      report(await api.adminBulkUploadJSON(parsed))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Upload failed.')
    } finally {
      setBusy(false)
    }
  }

  async function submitCSV() {
    setError(null)
    setResult(null)
    const file = fileRef.current?.files?.[0]
    if (!file) {
      setError('Choose a CSV file first.')
      return
    }
    setBusy(true)
    try {
      const text = await file.text()
      report(await api.adminBulkUploadCSV(text))
    } catch (e) {
      setError(e instanceof ApiError ? e.message : 'Upload failed.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="max-w-3xl mb-16">
      <h2 className="uc-tight text-[0.7rem] text-ink-faint mb-6">
        Bulk upload
      </h2>

      <div className="flex gap-2 mb-5">
        {(['json', 'csv'] as const).map((m) => (
          <button
            key={m}
            type="button"
            onClick={() => {
              setMode(m)
              setResult(null)
              setError(null)
            }}
            className={`text-xs px-3 py-1.5 border transition-colors cursor-pointer uppercase tracking-wide ${
              mode === m
                ? 'border-accent text-accent'
                : 'border-rule text-ink-soft hover:border-ink hover:text-ink'
            }`}
          >
            {m}
          </button>
        ))}
      </div>

      {mode === 'json' ? (
        <div className="flex flex-col gap-3">
          <label className="block">
            <span className="block uc-tight text-[0.7rem] text-ink-faint mb-2">
              Products JSON (array; price in cents)
            </span>
            <textarea
              value={jsonText}
              onChange={(e) => setJsonText(e.target.value)}
              rows={8}
              placeholder={JSON_EXAMPLE}
              className="w-full bg-raised border border-rule focus:border-ink px-3 py-2 text-sm text-ink font-mono placeholder:text-ink-faint outline-none resize-y"
            />
          </label>
          <Button type="button" disabled={busy} onClick={submitJSON}>
            {busy ? 'Uploading.' : 'Upload JSON'}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <p className="text-xs text-ink-faint">
            CSV header row with columns:{' '}
            <span className="text-ink-soft">
              name, description, price, stock_quantity, category_id, brand_id, weight_g, dimensions_cm
            </span>
            . Price is in cents.
          </p>
          <input
            ref={fileRef}
            type="file"
            accept=".csv,text/csv"
            className="text-sm text-ink-soft file:mr-4 file:border file:border-rule file:bg-transparent file:px-3 file:py-1.5 file:text-ink file:cursor-pointer"
          />
          <Button type="button" disabled={busy} onClick={submitCSV}>
            {busy ? 'Uploading.' : 'Upload CSV'}
          </Button>
        </div>
      )}

      {error && <p className="text-sm text-accent mt-4" role="alert">{error}</p>}

      {result && (
        <div className="mt-5 text-sm" role="status">
          <p className="text-ink">
            <span className="tnum text-positive">{result.created}</span> created,{' '}
            <span className="tnum text-accent">{result.failed}</span> failed.
          </p>
          {result.errors.length > 0 && (
            <ul className="mt-3 flex flex-col gap-1 text-xs text-ink-soft">
              {result.errors.map((er) => (
                <li key={er.index}>
                  <span className="tnum text-ink-faint">row {er.index + 1}</span>
                  {er.name ? ` (${er.name})` : ''}: {er.error}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </section>
  )
}

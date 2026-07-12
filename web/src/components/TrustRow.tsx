/* TrustRow.tsx — compact service-promise strip (delivery, returns, security,
 * support). Used in the footer; tone switches for dark bands.
 */
import { Truck, RotateCcw, ShieldCheck, Headset } from './icons'

const items = [
  { icon: Truck, label: 'Fast delivery', detail: 'Free over $50' },
  { icon: RotateCcw, label: '30-day returns', detail: 'No questions asked' },
  { icon: ShieldCheck, label: 'Secure checkout', detail: 'Encrypted end to end' },
  { icon: Headset, label: 'Support', detail: 'Real humans, every day' },
]

export function TrustRow({ onDark = false }: { onDark?: boolean }) {
  const labelCls = onDark ? 'text-on-primary' : 'text-ink'
  const detailCls = onDark ? 'text-on-primary/70' : 'text-ink-soft'
  const iconCls = onDark ? 'text-on-primary/80' : 'text-primary'

  return (
    <ul className="grid grid-cols-2 lg:grid-cols-4 gap-x-6 gap-y-4">
      {items.map(({ icon: Icon, label, detail }) => (
        <li key={label} className="flex items-center gap-3">
          <Icon size={24} strokeWidth={1.5} aria-hidden className={`shrink-0 ${iconCls}`} />
          <div className="min-w-0">
            <p className={`text-sm font-semibold leading-tight ${labelCls}`}>{label}</p>
            <p className={`text-xs leading-tight mt-0.5 ${detailCls}`}>{detail}</p>
          </div>
        </li>
      ))}
    </ul>
  )
}

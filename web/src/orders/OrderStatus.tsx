// Shared status helpers for the orders surface. Keep this colocated with
// the orders pages so the styling stays consistent.

const STATUS_LABELS: Record<string, string> = {
  pending_payment: 'Pending payment',
  paid: 'Paid',
  payment_failed: 'Payment failed',
  processing: 'Processing',
  shipped: 'Shipped',
  delivered: 'Delivered',
  cancelled: 'Cancelled',
  refunded: 'Refunded',
}

export function formatStatus(status: string): string {
  return STATUS_LABELS[status] ?? status
}

// Brief: one honest accent. Accent reserved for `paid` and `delivered` —
// the positive terminal states. Everything else stays in the neutral scale,
// distinguished by italic/weight cues, not hue.
const STATUS_TONE: Record<string, string> = {
  pending_payment: 'text-ink-soft',
  paid: 'text-accent',
  payment_failed: 'text-ink-soft italic',
  processing: 'text-ink-soft',
  shipped: 'text-ink',
  delivered: 'text-accent',
  cancelled: 'text-ink-faint italic',
  refunded: 'text-ink-faint italic',
}

export function StatusBadge({ status }: { status: string }) {
  const tone = STATUS_TONE[status] ?? 'text-ink-soft'
  return (
    <span className={`uc-tight text-[0.7rem] ${tone}`}>
      {formatStatus(status)}
    </span>
  )
}

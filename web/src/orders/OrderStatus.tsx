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

const STATUS_TONE: Record<string, string> = {
  pending_payment: 'border-amber-300 text-amber-800 bg-amber-50',
  paid: 'border-emerald-300 text-emerald-800 bg-emerald-50',
  payment_failed: 'border-red-300 text-red-800 bg-red-50',
  processing: 'border-sky-300 text-sky-800 bg-sky-50',
  shipped: 'border-indigo-300 text-indigo-800 bg-indigo-50',
  delivered: 'border-emerald-300 text-emerald-800 bg-emerald-50',
  cancelled: 'border-gray-300 text-gray-700 bg-gray-50',
  refunded: 'border-gray-300 text-gray-700 bg-gray-50',
}

export function StatusBadge({ status }: { status: string }) {
  const tone = STATUS_TONE[status] ?? 'border-gray-300 text-gray-700 bg-gray-50'
  return (
    <span className={`inline-block px-2 py-0.5 text-xs border ${tone}`}>
      {formatStatus(status)}
    </span>
  )
}

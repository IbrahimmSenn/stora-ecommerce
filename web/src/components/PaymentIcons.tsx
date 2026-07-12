/* PaymentIcons.tsx — simple text-in-rect marks for accepted payment methods.
 * Deliberately generic (no trademark artwork); purely informational.
 */

const methods = ['VISA', 'Mastercard', 'PayPal', 'Apple Pay']

export function PaymentIcons() {
  return (
    <ul className="flex flex-wrap items-center gap-2" aria-label="Accepted payment methods">
      {methods.map((m) => (
        <li key={m}>
          <span
            role="img"
            aria-label={m}
            className="inline-flex h-7 items-center rounded border border-on-primary/25 px-2 text-[0.65rem] font-bold tracking-wide text-on-primary/85 select-none"
          >
            {m}
          </span>
        </li>
      ))}
    </ul>
  )
}

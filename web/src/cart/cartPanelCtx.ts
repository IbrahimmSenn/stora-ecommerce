import { createContext } from 'react'

export type AddedItem = {
  productId: string
  productName: string
  unitPriceCents: number
  quantity: number
  imageUrl?: string | null
}

export type CartPanelState = {
  isOpen: boolean
  /** The line just added, used for the highlight wash. Null when opened as a
   *  plain cart preview (the nav cart trigger) rather than from an add. */
  added: AddedItem | null
  /** Open the panel with the just-added line. Stores the trigger element so
   *  focus can be returned on close. */
  openWith: (item: AddedItem, trigger?: HTMLElement | null) => void
  /** Open the panel as a quick cart preview (no just-added line). */
  open: (trigger?: HTMLElement | null) => void
  close: () => void
}

export const CartPanelCtx = createContext<CartPanelState | null>(null)

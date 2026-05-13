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
  /** The line just added, used for the highlight wash. Null when opened
   *  from a non-add context (e.g. an "Open cart" trigger we don't ship yet). */
  added: AddedItem | null
  /** Open the panel with the just-added line. Stores the trigger element so
   *  focus can be returned on close. */
  openWith: (item: AddedItem, trigger?: HTMLElement | null) => void
  close: () => void
}

export const CartPanelCtx = createContext<CartPanelState | null>(null)

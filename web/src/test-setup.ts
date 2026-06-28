import '@testing-library/jest-dom/vitest'
import { vi } from 'vitest'

// jsdom doesn't implement matchMedia; stub it so components that read
// prefers-reduced-motion (lib/motion) can render under test.
if (!window.matchMedia) {
  window.matchMedia = vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }))
}

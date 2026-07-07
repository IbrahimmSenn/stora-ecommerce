import { Component, type ReactNode } from 'react'
import { Page } from './Page'
import { Masthead } from './Masthead'

type Props = { children: ReactNode }
type State = { hasError: boolean }

// Class component: error boundaries have no hook equivalent.
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false }

  static getDerivedStateFromError(): State {
    return { hasError: true }
  }

  componentDidCatch(error: unknown) {
    console.error('unhandled render error:', error)
  }

  render() {
    if (!this.state.hasError) return this.props.children
    return (
      <Page width="max-w-3xl">
        <Masthead eyebrow="Error" title="Something went wrong" />
        <p className="text-ink-soft leading-relaxed mb-8 max-w-prose">
          An unexpected error occurred while displaying this page. Your cart
          and account are unaffected. Try again, or reload the page if the
          problem persists.
        </p>
        <div className="flex flex-wrap gap-x-8 gap-y-2 text-sm">
          <button
            type="button"
            onClick={() => this.setState({ hasError: false })}
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors cursor-pointer"
          >
            Try again
          </button>
          <button
            type="button"
            onClick={() => window.location.assign('/')}
            className="text-ink underline underline-offset-4 decoration-rule-strong hover:decoration-accent hover:text-accent transition-colors cursor-pointer"
          >
            Back to the shop
          </button>
        </div>
      </Page>
    )
  }
}

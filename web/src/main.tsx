import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import '@fontsource/lato/400.css'
import '@fontsource/lato/700.css'
import '@fontsource/lato/900.css'
import '@fontsource-variable/hanken-grotesk/index.css'
import './index.css'
import App from './App.tsx'
import { AuthProvider } from './auth/AuthContext'
import { CartProvider } from './cart/CartContext'
import { CartPanelProvider } from './cart/CartPanelProvider'
import { SidePanelProvider } from './components/SidePanelProvider'
import { ToastProvider } from './components/ToastProvider'
import { ThemeProvider } from './lib/theme'
import { initVitals } from './lib/vitals'

initVitals()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <CartProvider>
            <CartPanelProvider>
              <SidePanelProvider>
                <ToastProvider>
                  <App />
                </ToastProvider>
              </SidePanelProvider>
            </CartPanelProvider>
          </CartProvider>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
)

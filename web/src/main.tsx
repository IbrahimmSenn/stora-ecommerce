import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import '@fontsource-variable/bricolage-grotesque/index.css'
import '@fontsource-variable/hanken-grotesk/index.css'
import './index.css'
import App from './App.tsx'
import { AuthProvider } from './auth/AuthContext'
import { CartProvider } from './cart/CartContext'
import { CartPanelProvider } from './cart/CartPanelProvider'
import { ThemeProvider } from './lib/theme'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ThemeProvider>
      <BrowserRouter>
        <AuthProvider>
          <CartProvider>
            <CartPanelProvider>
              <App />
            </CartPanelProvider>
          </CartProvider>
        </AuthProvider>
      </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
)

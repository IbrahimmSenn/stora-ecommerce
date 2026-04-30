import { Route, Routes } from 'react-router-dom'
import { Nav } from './components/Nav'
import { ProductsPage } from './products/ProductsPage'
import { CartPage } from './cart/CartPage'
import { CheckoutPage } from './checkout/CheckoutPage'
import { LoginPage } from './auth/LoginPage'

function App() {
  return (
    <>
      <Nav />
      <Routes>
        <Route path="/" element={<ProductsPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/checkout" element={<CheckoutPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<p className="p-8">Not found.</p>} />
      </Routes>
    </>
  )
}

export default App

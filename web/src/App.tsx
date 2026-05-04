import { Route, Routes } from 'react-router-dom'
import { Nav } from './components/Nav'
import { ProductsPage } from './products/ProductsPage'
import { CartPage } from './cart/CartPage'
import { CheckoutPage } from './checkout/CheckoutPage'
import { PaymentPage } from './payment/PaymentPage'
import { OrderDetailPage } from './orders/OrderDetailPage'
import { OrderHistoryPage } from './orders/OrderHistoryPage'
import { LoginPage } from './auth/LoginPage'

function App() {
  return (
    <>
      <Nav />
      <Routes>
        <Route path="/" element={<ProductsPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/checkout" element={<CheckoutPage />} />
        <Route path="/orders" element={<OrderHistoryPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
        <Route path="/orders/:id/confirmation" element={<OrderDetailPage />} />
        <Route path="/orders/:id/pay" element={<PaymentPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="*" element={<p className="p-8">Not found.</p>} />
      </Routes>
    </>
  )
}

export default App

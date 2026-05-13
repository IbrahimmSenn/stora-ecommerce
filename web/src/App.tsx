import { Navigate, Route, Routes } from 'react-router-dom'
import { Nav } from './components/Nav'
import { ProductsPage } from './products/ProductsPage'
import { ProductDetailPage } from './products/ProductDetailPage'
import { CartPage } from './cart/CartPage'
import { CheckoutPage } from './checkout/CheckoutPage'
import { PaymentPage } from './payment/PaymentPage'
import { OrderDetailPage } from './orders/OrderDetailPage'
import { OrderHistoryPage } from './orders/OrderHistoryPage'
import { LoginPage } from './auth/LoginPage'
import { RegisterPage } from './auth/RegisterPage'
import { ForgotPasswordPage } from './auth/ForgotPasswordPage'
import { ResetPasswordPage } from './auth/ResetPasswordPage'
import { OAuthCallbackPage } from './auth/OAuthCallbackPage'
import { AccountPage } from './account/AccountPage'
import { TwoFactorSetupPage } from './account/TwoFactorSetupPage'
import { TwoFactorDisablePage } from './account/TwoFactorDisablePage'
import { TokenTesterPage } from './dev/TokenTesterPage'
import { AdminLayout } from './admin/AdminLayout'
import { AdminProductsPage } from './admin/AdminProductsPage'
import { AdminCategoriesPage } from './admin/AdminCategoriesPage'
import { AdminBrandsPage } from './admin/AdminBrandsPage'

function App() {
  return (
    <>
      <Nav />
      <Routes>
        <Route path="/" element={<ProductsPage />} />
        <Route path="/shop/:slug" element={<ProductsPage />} />
        <Route path="/product/:id" element={<ProductDetailPage />} />
        <Route path="/cart" element={<CartPage />} />
        <Route path="/checkout" element={<CheckoutPage />} />
        <Route path="/orders" element={<OrderHistoryPage />} />
        <Route path="/orders/:id" element={<OrderDetailPage />} />
        <Route path="/orders/:id/confirmation" element={<OrderDetailPage />} />
        <Route path="/orders/:id/pay" element={<PaymentPage />} />

        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/auth/oauth/callback" element={<OAuthCallbackPage />} />

        <Route path="/account" element={<AccountPage />} />
        <Route path="/account/2fa/setup" element={<TwoFactorSetupPage />} />
        <Route path="/account/2fa/disable" element={<TwoFactorDisablePage />} />

        <Route path="/dev/tokens" element={<TokenTesterPage />} />

        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<Navigate to="products" replace />} />
          <Route path="products" element={<AdminProductsPage />} />
          <Route path="categories" element={<AdminCategoriesPage />} />
          <Route path="brands" element={<AdminBrandsPage />} />
        </Route>

        <Route path="*" element={<p className="p-8">Not found.</p>} />
      </Routes>
    </>
  )
}

export default App

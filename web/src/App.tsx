import { Navigate, Route, Routes } from 'react-router-dom'
import { Nav } from './components/Nav'
import { Footer } from './components/Footer'
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
import { AdminDeliveryPage } from './admin/AdminDeliveryPage'
import { AdminOrdersPage } from './admin/AdminOrdersPage'
import { AdminUsersPage } from './admin/AdminUsersPage'
import { AdminReviewsPage } from './admin/AdminReviewsPage'
import { AdminAuditPage } from './admin/AdminAuditPage'
import { NotFoundPage } from './components/NotFoundPage'
import { AboutPage } from './pages/AboutPage'
import { ContactPage } from './pages/ContactPage'

function App() {
  return (
    <>
      <a
        href="#main"
        className="sr-only focus:not-sr-only focus:fixed focus:top-3 focus:left-3 focus:z-[100] focus:bg-accent focus:text-on-accent focus:px-4 focus:py-2 focus:no-underline"
      >
        Skip to content
      </a>
      <Nav />
      <main id="main" tabIndex={-1} className="outline-none">
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

        <Route path="/about" element={<AboutPage />} />
        <Route path="/contact" element={<ContactPage />} />

        <Route path="/account" element={<AccountPage />} />
        <Route path="/account/2fa/setup" element={<TwoFactorSetupPage />} />
        <Route path="/account/2fa/disable" element={<TwoFactorDisablePage />} />

        {import.meta.env.DEV && (
          <Route path="/dev/tokens" element={<TokenTesterPage />} />
        )}

        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<Navigate to="products" replace />} />
          <Route path="products" element={<AdminProductsPage />} />
          <Route path="categories" element={<AdminCategoriesPage />} />
          <Route path="brands" element={<AdminBrandsPage />} />
          <Route path="delivery" element={<AdminDeliveryPage />} />
          <Route path="orders" element={<AdminOrdersPage />} />
          <Route path="users" element={<AdminUsersPage />} />
          <Route path="reviews" element={<AdminReviewsPage />} />
          <Route path="audit" element={<AdminAuditPage />} />
        </Route>

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
      </main>
      <Footer />
    </>
  )
}

export default App

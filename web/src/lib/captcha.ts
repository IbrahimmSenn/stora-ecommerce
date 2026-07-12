/* captcha.ts — fetches the public reCAPTCHA site key from the server and
 * lazily loads grecaptcha. If the key is empty (dev mode with SKIP_CAPTCHA),
 * returns an empty token and the backend will skip verification.
 */

type GrecaptchaApi = {
  ready: (cb: () => void) => void
  execute: (siteKey: string, opts: { action: string }) => Promise<string>
}

declare global {
  interface Window {
    grecaptcha?: GrecaptchaApi
  }
}

let siteKeyPromise: Promise<string> | null = null
let scriptPromise: Promise<void> | null = null

async function fetchSiteKey(): Promise<string> {
  if (!siteKeyPromise) {
    siteKeyPromise = fetch('/api/v1/config/recaptcha')
      .then((r) => r.json() as Promise<{ site_key: string }>)
      .then((d) => d.site_key ?? '')
      .catch(() => '')
  }
  return siteKeyPromise
}

function loadScript(siteKey: string): Promise<void> {
  if (scriptPromise) return scriptPromise
  scriptPromise = new Promise((resolve, reject) => {
    const s = document.createElement('script')
    s.src = `https://www.google.com/recaptcha/api.js?render=${siteKey}`
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error('recaptcha script failed to load'))
    document.head.appendChild(s)
  })
  return scriptPromise
}

// True when the server has a reCAPTCHA site key configured. Used to decide
// whether to show the reCAPTCHA branding line (required by Google's terms).
export async function captchaEnabled(): Promise<boolean> {
  return (await fetchSiteKey()) !== ''
}

export async function getCaptchaToken(action: string): Promise<string> {
  const siteKey = await fetchSiteKey()
  if (!siteKey) return '' // dev mode — backend's SKIP_CAPTCHA accepts empty
  await loadScript(siteKey)
  const api = window.grecaptcha
  if (!api) return ''
  await new Promise<void>((resolve) => api.ready(resolve))
  return api.execute(siteKey, { action })
}

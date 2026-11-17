// Only same-app paths may be redirected to after login ("/x", never "//evil.com" or "https://...").
function safeRedirect(target: string | undefined): string {
  return target && target.startsWith('/') && !target.startsWith('//') && target !== '/login'
    ? target
    : '/tracking'
}

export { safeRedirect }

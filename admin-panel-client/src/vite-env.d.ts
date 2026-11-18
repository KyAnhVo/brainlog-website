interface ViteTypeOptions {
  strictImportMetaEnv: unknown
}

interface ImportMetaEnv {
  readonly VITE_API_BASE_URL: string
  // Dev only: "false" turns MSW off and uses the Vite proxy to the Go server.
  readonly VITE_USE_MOCKS?: string
}

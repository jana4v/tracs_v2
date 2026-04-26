import { ModuleRegistry, provideGlobalGridOptions } from 'ag-grid-community'
import { AllEnterpriseModule, LicenseManager } from 'ag-grid-enterprise'

// Register all AG Grid Enterprise modules
ModuleRegistry.registerModules([AllEnterpriseModule])

// Use legacy theme (ag-theme-alpine-dark) with updated cell selection
provideGlobalGridOptions({ 
  theme: 'legacy',
  cellSelection: true,
})

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig()
  const licenseKey = config.public.agGridLicenseKey as string
  if (licenseKey) {
    LicenseManager.setLicenseKey(licenseKey)
  }
})

import { useMemo, type ReactNode } from 'react'
import { ThemeProvider } from '@mui/material/styles'
import ScopedCssBaseline from '@mui/material/ScopedCssBaseline'
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider'
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns'
import { SnackbarProvider } from 'notistack'
import { buildTheme } from '@/lib/theme'
import { ThemeModeContext } from '@/lib/ThemeModeContext'
import { FaviconProvider } from '@/lib/FaviconContext'
import { PaletteProvider } from '@/lib/PaletteContext'
import { DateRangeProvider } from '@/lib/dateRange'
import { useThemeMode } from '@/lib/themeMode'

/** Hosts a legacy (MUI) page inside the new shell with every provider it expects. */
export function LegacyPage({ children }: { children: ReactNode }) {
  const { mode, toggle } = useThemeMode()
  const theme = useMemo(() => buildTheme(mode), [mode])
  return (
    <ThemeModeContext.Provider value={{ mode, toggleMode: toggle }}>
      <ThemeProvider theme={theme}>
        <LocalizationProvider dateAdapter={AdapterDateFns}>
          <ScopedCssBaseline>
            <SnackbarProvider maxSnack={5} anchorOrigin={{ vertical: 'bottom', horizontal: 'left' }} autoHideDuration={5000}>
              <FaviconProvider>
              <PaletteProvider>
                <DateRangeProvider>{children}</DateRangeProvider>
              </PaletteProvider>
              </FaviconProvider>
            </SnackbarProvider>
          </ScopedCssBaseline>
        </LocalizationProvider>
      </ThemeProvider>
    </ThemeModeContext.Provider>
  )
}

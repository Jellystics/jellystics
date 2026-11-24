import { jellyfinSetting, pickLanguage, resolveTheme } from './prefs'

describe('pickLanguage', () => {
  it('takes the first supported candidate', () => expect(pickLanguage([null, 'de-DE', 'fr-CA', 'es'])).toBe('fr'))
  it('reads Jellyfin values like es-mx / en-us', () => { expect(pickLanguage(['es-mx'])).toBe('es'); expect(pickLanguage(['en_US'])).toBe('en') })
  it('falls back to English', () => expect(pickLanguage(['de', '', undefined])).toBe('en'))
})

describe('resolveTheme', () => {
  it('follows the Jellyfin theme when set', () => {
    expect(resolveTheme('light')).toBe('light')
    expect(resolveTheme('appletv')).toBe('light')
    expect(resolveTheme('catppuccin-latte')).toBe('light')
    expect(resolveTheme('purplehaze')).toBe('dark')
    expect(resolveTheme('dark')).toBe('dark')
  })
  it('is dark by default, whatever the browser prefers', () => {
    expect(resolveTheme(null)).toBe('dark')
    expect(resolveTheme(undefined)).toBe('dark')
    expect(resolveTheme('')).toBe('dark')
  })
})

describe('jellyfinSetting', () => {
  it('reads the per-user key', () => {
    const s = { getItem: (k: string) => (k === 'u1-appTheme' ? 'light' : null) }
    expect(jellyfinSetting(s, 'u1', 'appTheme')).toBe('light')
    expect(jellyfinSetting(s, null, 'appTheme')).toBeNull()
    expect(jellyfinSetting({ getItem: () => { throw new Error('x') } }, 'u1', 'appTheme')).toBeNull()
  })
})

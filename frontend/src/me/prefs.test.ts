import { jellyfinSetting, pickLanguage, resolveTheme } from './prefs'

describe('pickLanguage', () => {
  it('takes the first supported candidate', () => expect(pickLanguage([null, 'de-DE', 'fr-CA', 'es'])).toBe('fr'))
  it('reads Jellyfin values like es-mx / en-us', () => { expect(pickLanguage(['es-mx'])).toBe('es'); expect(pickLanguage(['en_US'])).toBe('en') })
  it('falls back to English', () => expect(pickLanguage(['de', '', undefined])).toBe('en'))
})

describe('resolveTheme', () => {
  it('follows the Jellyfin theme when set', () => {
    expect(resolveTheme('light', false)).toBe('light')
    expect(resolveTheme('appletv', false)).toBe('light')
    expect(resolveTheme('catppuccin-latte', false)).toBe('light')
    expect(resolveTheme('purplehaze', true)).toBe('dark')
    expect(resolveTheme('dark', true)).toBe('dark')
  })
  it('is dark by default, light when the browser prefers it', () => {
    expect(resolveTheme(null, false)).toBe('dark')
    expect(resolveTheme(null, true)).toBe('light')
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

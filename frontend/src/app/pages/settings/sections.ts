export const SETTINGS_SECTIONS = ['config', 'tasks', 'backup', 'logs', 'webhooks', 'apikeys', 'about'] as const
export type SettingsSection = (typeof SETTINGS_SECTIONS)[number]

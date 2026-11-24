-- 018: plugin-reported « Mes statistiques » error (null | unreachable |
-- secretMismatch), served by the plugin status.
ALTER TABLE plugin_sync_state ADD COLUMN IF NOT EXISTS me_error text;

import { Box, Typography } from '@mui/material'
import { DataHistogram24Regular } from '@fluentui/react-icons'
import { useTranslation } from 'react-i18next'

interface EmptyStateProps {
  message?: string
  icon?: React.ReactNode
  height?: number | string
}

export default function EmptyState({ message, icon, height = '100%' }: EmptyStateProps) {
  const { t } = useTranslation()

  return (
    <Box
      sx={{
        height,
        minHeight: 120,
        width: '100%',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 1,
        color: 'text.disabled',
      }}
    >
      {icon ?? <DataHistogram24Regular style={{ fontSize: 32, opacity: 0.5 }} />}
      <Typography variant="body2" sx={{ color: 'text.secondary' }}>
        {message ?? t('common.noData')}
      </Typography>
    </Box>
  )
}

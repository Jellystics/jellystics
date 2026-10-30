import { Box, Button, Paper, Typography } from '@mui/material'
import { ErrorCircle24Regular } from '@fluentui/react-icons'
import { useRouteError, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'

export default function RouteErrorBoundary() {
  const error = useRouteError()
  const navigate = useNavigate()
  const { t } = useTranslation()

  const detail = error instanceof Error ? error.message : String(error ?? '')

  return (
    <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
      <Paper
        sx={{
          maxWidth: 560,
          width: '100%',
          p: 4,
          textAlign: 'center',
          boxShadow: 'none',
          border: '1px solid',
          borderColor: 'divider',
        }}
      >
        <ErrorCircle24Regular style={{ fontSize: 40, opacity: 0.6 }} />
        <Typography variant="h6" sx={{ mt: 1.5, fontWeight: 700 }}>
          {t('error.somethingWentWrong')}
        </Typography>
        {detail && (
          <Typography
            variant="body2"
            sx={{
              mt: 1.5,
              px: 2,
              py: 1.5,
              borderRadius: 1,
              bgcolor: 'action.hover',
              fontFamily: 'monospace',
              fontSize: 12,
              textAlign: 'left',
              wordBreak: 'break-word',
            }}
          >
            {detail}
          </Typography>
        )}
        <Box sx={{ mt: 3, display: 'flex', gap: 1.5, justifyContent: 'center' }}>
          <Button variant="contained" onClick={() => window.location.reload()}>
            {t('common.reload')}
          </Button>
          <Button variant="outlined" onClick={() => navigate('/')}>
            {t('error.backToDashboard')}
          </Button>
        </Box>
      </Paper>
    </Box>
  )
}

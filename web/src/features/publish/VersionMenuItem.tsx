import AdjustIcon from '@mui/icons-material/Adjust'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked'
import { MenuItem, Stack, Typography } from '@mui/material'
import type { Translator } from '../../i18n'
import { getVersionLabel, type VersionOption } from './workflowOptions'

export function renderVersionMenuItem(option: VersionOption, t: Translator, disabled = false, current = false) {
  const status = option.published ? 'published' : 'draft'
  const StatusIcon = option.published ? CheckCircleOutlineIcon : RadioButtonUncheckedIcon

  return (
    <MenuItem key={option.value} value={option.value} disabled={disabled} aria-label={getVersionLabel(option, t)} sx={{ color: disabled ? 'text.disabled' : 'text.primary' }}>
      <Stack direction="row" spacing={1} alignItems="center" sx={{ width: '100%' }}>
        <Typography component="span" sx={{ flex: 1 }}>
          {option.version > 0 ? `v${option.version}` : option.label}
        </Typography>
        <Stack direction="row" spacing={1} alignItems="center" sx={{ color: 'text.secondary' }}>
          {current && (
            <Stack direction="row" spacing={0.5} alignItems="center">
              <AdjustIcon data-testid="version-status-current" fontSize="small" />
              <Typography component="span" variant="body2">
                {t('operations.versionCurrent')}
              </Typography>
            </Stack>
          )}
          <Stack direction="row" spacing={0.5} alignItems="center">
            <StatusIcon data-testid={`version-status-${status}`} fontSize="small" />
            <Typography component="span" variant="body2">
              {t(option.published ? 'operations.versionPublished' : 'operations.versionDraft')}
            </Typography>
          </Stack>
        </Stack>
      </Stack>
    </MenuItem>
  )
}

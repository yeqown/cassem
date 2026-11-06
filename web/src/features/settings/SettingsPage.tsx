import { useState, type ChangeEvent } from 'react'
import { Box, FormControl, FormControlLabel, InputLabel, MenuItem, Paper, Select, Stack, Switch, Typography } from '@mui/material'
import type { SelectChangeEvent } from '@mui/material/Select'
import { ContentViewer } from '../elements/ContentViewer'
import { codeThemeOptions, readSettings, uiThemeOptions, updateSettings, type CodeTheme, type UITheme } from '../../lib/settings'
import { useI18n } from '../../i18n'

const codePreview = `{
  "enabled": true,
  "region": "ap-east-1",
  "retries": 3,
  "features": ["watch", "dispatch"]
}`

export function SettingsPage() {
  const { t } = useI18n()
  const [settings, setSettings] = useState(readSettings)

  function handleCodeThemeChange(event: SelectChangeEvent<CodeTheme>) {
    const codeTheme = event.target.value as CodeTheme
    setSettings((current) => ({ ...current, codeTheme }))
    updateSettings({ codeTheme })
  }

  function handleUIThemeChange(event: SelectChangeEvent<UITheme>) {
    const uiTheme = event.target.value as UITheme
    setSettings((current) => ({ ...current, uiTheme }))
    updateSettings({ uiTheme })
  }

  function handleLineWrappingChange(event: ChangeEvent<HTMLInputElement>) {
    const editorLineWrapping = event.target.checked
    setSettings((current) => ({ ...current, editorLineWrapping }))
    updateSettings({ editorLineWrapping })
  }

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">{t('common.settings')}</Typography>
        <Typography color="text.secondary">{t('common.localPreferences')}</Typography>
      </Box>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">{t('common.appearance')}</Typography>
          <FormControl fullWidth>
            <InputLabel id="ui-theme-label">{t('common.uiTheme')}</InputLabel>
            <Select labelId="ui-theme-label" label={t('common.uiTheme')} value={settings.uiTheme} onChange={handleUIThemeChange}>
              {uiThemeOptions.map((option) => (
                <MenuItem key={option.value} value={option.value}>{t(`common.theme.${option.value}`)}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </Stack>
      </Paper>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">{t('common.editor')}</Typography>
          <Stack data-testid="editor-settings-layout" data-preview-position="right" direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="stretch">
            <Stack data-testid="editor-settings-panel" spacing={2} flex={1} minWidth={0}>
              <FormControl fullWidth>
                <InputLabel id="code-theme-label">{t('common.codeTheme')}</InputLabel>
                <Select labelId="code-theme-label" label={t('common.codeTheme')} value={settings.codeTheme} onChange={handleCodeThemeChange}>
                  {codeThemeOptions.map((option) => (
                    <MenuItem key={option.value} value={option.value}>{option.label}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControlLabel
                control={<Switch checked={settings.editorLineWrapping} onChange={handleLineWrappingChange} />}
                label={t('common.lineWrapping')}
              />
            </Stack>
            <Stack data-testid="code-theme-preview-panel" flex={1.2} minWidth={0}>
              <ContentViewer
                ariaLabel={t('common.codePreview')}
                codeTheme={settings.codeTheme}
                contentType={1}
                lineWrapping={settings.editorLineWrapping}
                minRows={7}
                value={codePreview}
              />
            </Stack>
          </Stack>
        </Stack>
      </Paper>
    </Stack>
  )
}

import { useState, type ChangeEvent } from 'react'
import { Box, FormControl, FormControlLabel, InputLabel, MenuItem, Paper, Select, Stack, Switch, Typography } from '@mui/material'
import type { SelectChangeEvent } from '@mui/material/Select'
import { ContentViewer } from '../elements/ContentViewer'
import { codeThemeOptions, readSettings, uiThemeOptions, updateSettings, type CodeTheme, type UITheme } from '../../lib/settings'

const codePreview = `{
  "enabled": true,
  "region": "ap-east-1",
  "retries": 3,
  "features": ["watch", "dispatch"]
}`

export function SettingsPage() {
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
        <Typography variant="h4" component="h1">Settings</Typography>
        <Typography color="text.secondary">Local preferences stored in this browser.</Typography>
      </Box>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">Appearance</Typography>
          <FormControl fullWidth>
            <InputLabel id="ui-theme-label">UI theme</InputLabel>
            <Select labelId="ui-theme-label" label="UI theme" value={settings.uiTheme} onChange={handleUIThemeChange}>
              {uiThemeOptions.map((option) => (
                <MenuItem key={option.value} value={option.value}>{option.label}</MenuItem>
              ))}
            </Select>
          </FormControl>
        </Stack>
      </Paper>

      <Paper variant="outlined" sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">Editor</Typography>
          <Stack data-testid="editor-settings-layout" data-preview-position="right" direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems="stretch">
            <Stack data-testid="editor-settings-panel" spacing={2} flex={1} minWidth={0}>
              <FormControl fullWidth>
                <InputLabel id="code-theme-label">Code theme</InputLabel>
                <Select labelId="code-theme-label" label="Code theme" value={settings.codeTheme} onChange={handleCodeThemeChange}>
                  {codeThemeOptions.map((option) => (
                    <MenuItem key={option.value} value={option.value}>{option.label}</MenuItem>
                  ))}
                </Select>
              </FormControl>
              <FormControlLabel
                control={<Switch checked={settings.editorLineWrapping} onChange={handleLineWrappingChange} />}
                label="Editor line wrapping"
              />
            </Stack>
            <Stack data-testid="code-theme-preview-panel" flex={1.2} minWidth={0}>
              <ContentViewer
                ariaLabel="Code theme preview"
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

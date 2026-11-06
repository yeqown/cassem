import { useId, useState } from 'react'
import TranslateIcon from '@mui/icons-material/Translate'
import CheckIcon from '@mui/icons-material/Check'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { Box, Button, ListItemIcon, ListItemText, Menu, MenuItem } from '@mui/material'
import { languageOptions, useI18n } from '../i18n'

export function LanguageSwitcher() {
  const { locale, setLocale, t } = useI18n()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const id = useId()
  const current = languageOptions.find((option) => option.value === locale)!

  return (
    <>
      <Button
        id={`${id}-button`}
        color="inherit"
        aria-label={t('common.language')}
        aria-haspopup="menu"
        aria-controls={anchor ? `${id}-menu` : undefined}
        aria-expanded={Boolean(anchor)}
        onClick={(event) => setAnchor(event.currentTarget)}
        sx={{ minWidth: 44, minHeight: 44, px: 1.25, mr: 1, gap: 0.75, textTransform: 'none', flexShrink: 0 }}
      >
        <TranslateIcon fontSize="small" />
        <Box component="span" sx={{ display: { xs: 'none', sm: 'inline' } }}>{current.label}</Box>
        <ExpandMoreIcon fontSize="small" sx={{ display: { xs: 'none', sm: 'inline-flex' } }} />
      </Button>
      <Menu
        id={`${id}-menu`}
        anchorEl={anchor}
        open={Boolean(anchor)}
        onClose={() => setAnchor(null)}
        slotProps={{ list: { 'aria-labelledby': `${id}-button` } }}
      >
        {languageOptions.map((option) => (
          <MenuItem
            key={option.value}
            lang={option.value}
            role="menuitemradio"
            aria-checked={locale === option.value}
            selected={locale === option.value}
            onClick={() => { setLocale(option.value); setAnchor(null) }}
            sx={{ minWidth: 160, minHeight: 44 }}
          >
            <ListItemIcon>{locale === option.value && <CheckIcon fontSize="small" />}</ListItemIcon>
            <ListItemText>{option.label}</ListItemText>
          </MenuItem>
        ))}
      </Menu>
    </>
  )
}

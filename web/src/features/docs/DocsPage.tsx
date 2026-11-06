import { useEffect, type ReactNode } from 'react'
import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import AutoStoriesIcon from '@mui/icons-material/AutoStories'
import CheckCircleOutlineIcon from '@mui/icons-material/CheckCircleOutline'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Chip,
  Divider,
  List,
  ListItemButton,
  ListItemText,
  Paper,
  Stack,
  Typography,
} from '@mui/material'
import { Link as RouterLink, useLocation } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { useToast } from '../../components/ToastProvider'
import { useI18n } from '../../i18n'
import {
  configurationExample,
  goExample,
  httpFetchExample,
  httpResponseExample,
  installExample,
  localModuleExample,
  tokenCreateExample,
  tokenListExample,
  tokenLoginExample,
  tokenRevokeExample,
} from './examples'

const sections = ['concepts', 'quickstart', 'consume', 'updates', 'faq'] as const
const concepts = [
  { key: 'app', example: 'order-service' },
  { key: 'env', example: 'dev' },
  { key: 'element', example: 'app-config' },
  { key: 'content', example: 'JSON' },
  { key: 'version' },
  { key: 'instance', example: 'order-service-local-01' },
] as const
const services = [
  { name: 'cassemadm', key: 'adm' },
  { name: 'cassemkv', key: 'kv' },
  { name: 'cassemagent', key: 'agent' },
] as const
const creationSteps = ['app', 'env', 'element', 'publish'] as const
const updateSteps = ['edit', 'verify', 'rollback'] as const
const questions = ['unpublished', 'read', 'draft', 'watch', 'access'] as const

function CodeExample({ title, code }: { title: string; code: string }) {
  const { t } = useI18n()
  const { showToast } = useToast()

  async function copy() {
    try {
      await navigator.clipboard.writeText(code)
      showToast(t('docs.copied'), 'success')
    } catch {
      showToast(t('docs.copyFailed'), 'error')
    }
  }

  return (
    <Paper variant="outlined" sx={{ minWidth: 0, overflow: 'hidden' }}>
      <Stack direction={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ sm: 'center' }} sx={{ px: 2, py: 0.5, bgcolor: 'action.hover' }}>
        <Typography variant="body2" fontWeight={600} sx={{ py: 1 }}>{title}</Typography>
        <Button onClick={() => void copy()} startIcon={<ContentCopyIcon />} aria-label={t('docs.copyLabel', { title })} sx={{ minHeight: 44, alignSelf: { xs: 'flex-start', sm: 'auto' }, flexShrink: 0 }}>
          {t('docs.copy')}
        </Button>
      </Stack>
      <Divider />
      <Box
        component="pre"
        tabIndex={0}
        aria-label={t('docs.codeLabel', { title })}
        sx={{ m: 0, p: 2, overflowX: 'auto', fontSize: 13, lineHeight: 1.75, tabSize: 4, '&:focus-visible': { outline: '2px solid', outlineColor: 'primary.main', outlineOffset: -2 } }}
      >
        <code>{code}</code>
      </Box>
    </Paper>
  )
}

function DocSection({ id, number, title, children }: { id: string; number: number; title: string; children: ReactNode }) {
  return (
    <Paper component="section" variant="outlined" aria-labelledby={id} sx={{ p: { xs: 2, sm: 3.5 }, minWidth: 0 }}>
      <Stack spacing={3}>
        <Stack direction="row" spacing={1.5} alignItems="baseline">
          <Typography aria-hidden="true" color="primary.main" fontWeight={700}>{String(number).padStart(2, '0')}</Typography>
          <Typography id={id} tabIndex={-1} variant="h5" component="h2" sx={{ fontWeight: 700, scrollMarginTop: 96 }}>{title}</Typography>
        </Stack>
        {children}
      </Stack>
    </Paper>
  )
}

export function DocsPage() {
  const { t } = useI18n()
  const { hash, key: locationKey } = useLocation()

  useEffect(() => {
    if (!hash) return
    const heading = document.getElementById(hash.slice(1))
    if (!heading) return
    heading.scrollIntoView({ block: 'start' })
    heading.focus({ preventScroll: true })
  }, [hash, locationKey])

  return (
    <Stack spacing={3} sx={{ maxWidth: 1280, mx: 'auto', minWidth: 0, '& p': { lineHeight: 1.8 }, overflowWrap: 'anywhere' }}>
      <AppBreadcrumbs items={[{ label: t('common.guide') }]} />
      <Paper variant="outlined" sx={{ p: { xs: 2.5, sm: 4 }, borderTop: 4, borderTopColor: 'primary.main' }}>
        <Stack spacing={2}>
          <Stack direction="row" spacing={1} alignItems="center" sx={{ color: 'primary.main' }}>
            <AutoStoriesIcon />
            <Typography variant="overline" fontWeight={700}>{t('common.guide')}</Typography>
          </Stack>
          <Typography variant="h4" component="h1" fontWeight={700}>{t('docs.title')}</Typography>
          <Typography color="text.secondary">{t('docs.description')}</Typography>
          <Stack direction="row" spacing={1.5} useFlexGap flexWrap="wrap">
            <Button component={RouterLink} to="/docs#quickstart" variant="contained" endIcon={<ArrowForwardIcon />} sx={{ minHeight: 44, px: 2 }}>{t('docs.start')}</Button>
            <Button component={RouterLink} to="/apps" variant="outlined" sx={{ minHeight: 44, px: 2 }}>{t('docs.openApps')}</Button>
          </Stack>
        </Stack>
      </Paper>

      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: 'minmax(0, 1fr)', lg: 'minmax(0, 1fr) 220px' }, gap: 3, alignItems: 'start' }}>
        <Paper component="nav" variant="outlined" aria-label={t('docs.contents')} sx={{ p: 2, gridColumn: { lg: 2 }, gridRow: { lg: 1 }, position: { lg: 'sticky' }, top: 88 }}>
          <Typography variant="subtitle2" sx={{ px: 1.5, pb: 1 }}>{t('docs.contents')}</Typography>
          <List disablePadding>
            {sections.map((section, index) => (
              <ListItemButton
                key={section}
                component={RouterLink}
                to={`/docs#${section}`}
                selected={hash === `#${section}`}
                aria-current={hash === `#${section}` ? 'location' : undefined}
                sx={{ minHeight: 44, px: 1.5, gap: 1.5 }}
              >
                <Typography variant="caption" color="text.secondary" aria-hidden="true" sx={{ flexShrink: 0, whiteSpace: 'nowrap', fontVariantNumeric: 'tabular-nums' }}>{String(index + 1).padStart(2, '0')}</Typography>
                <ListItemText primary={t(`docs.${section}`)} primaryTypographyProps={{ variant: 'body2' }} />
              </ListItemButton>
            ))}
          </List>
        </Paper>

        <Stack spacing={3} sx={{ minWidth: 0, gridColumn: 1, gridRow: { lg: 1 } }}>
          <DocSection id="concepts" number={1} title={t('docs.concepts')}>
            <Typography>{t('docs.concepts.intro')}</Typography>
            <Box component="dl" sx={{ m: 0 }}>
              {concepts.map((concept) => (
                <Box key={concept.key} sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', sm: '180px minmax(0, 1fr)' }, gap: 1, py: 2, borderBottom: 1, borderColor: 'divider', '&:first-of-type': { pt: 0 } }}>
                  <Typography component="dt" fontWeight={600}>{t(`docs.concepts.${concept.key}`)}</Typography>
                  <Box component="dd" sx={{ m: 0 }}>
                    <Typography variant="body2" color="text.secondary">{t(`docs.concepts.${concept.key}Description`)}</Typography>
                    <Typography variant="body2" sx={{ mt: 0.75 }}>
                      {t('docs.concepts.example')}{': '}
                      {'example' in concept ? <Box component="code">{concept.example}</Box> : t('docs.concepts.versionExample')}
                    </Typography>
                  </Box>
                </Box>
              ))}
            </Box>
            <Typography variant="h6" component="h3">{t('docs.concepts.services')}</Typography>
            <Stack spacing={2}>
              {services.map((service) => (
                <Box key={service.key}>
                  <Typography component="h4" variant="subtitle1" fontWeight={700}>{service.name}</Typography>
                  <Typography variant="body2" color="text.secondary">{t(`docs.concepts.${service.key}`)}</Typography>
                </Box>
              ))}
            </Stack>
            <Box sx={{ bgcolor: 'action.hover', p: 2, borderLeft: 3, borderColor: 'primary.main' }}>
              {(['managePath', 'readPath', 'pushPath'] as const).map((path) => <Typography key={path} variant="body2" sx={{ py: 0.5 }}>{t(`docs.concepts.${path}`)}</Typography>)}
            </Box>
          </DocSection>

          <DocSection id="quickstart" number={2} title={t('docs.quickstart')}>
            <Typography>{t('docs.quickstart.intro')}</Typography>
            <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap">
              <Chip label="App: order-service" variant="outlined" />
              <Chip label="Env: dev" variant="outlined" />
              <Chip label="Key: app-config" variant="outlined" />
            </Stack>
            <Alert severity="info">
              <AlertTitle>{t('docs.quickstart.before')}</AlertTitle>
              {t('docs.quickstart.prerequisites')}
            </Alert>
            <Stack component="ol" spacing={3.5} sx={{ m: 0, pl: 3, '& > li::marker': { color: 'primary.main', fontWeight: 700 } }}>
              {creationSteps.map((step) => (
                <Box component="li" key={step} sx={{ pl: 0.5, minWidth: 0 }}>
                  <Stack spacing={1.5}>
                    <Typography variant="h6" component="h3">{t(`docs.quickstart.${step}Title`)}</Typography>
                    <Typography>{t(`docs.quickstart.${step}Body`)}</Typography>
                    {step === 'element' && <CodeExample title={t('docs.quickstart.payload')} code={configurationExample} />}
                    <Stack direction="row" spacing={1} alignItems="flex-start">
                      <CheckCircleOutlineIcon color="success" fontSize="small" sx={{ mt: 0.25, flexShrink: 0 }} />
                      <Typography variant="body2" color="text.secondary">{t(`docs.quickstart.${step}Result`)}</Typography>
                    </Stack>
                  </Stack>
                </Box>
              ))}
            </Stack>
            <Alert severity="info">{t('docs.quickstart.publishNote')}</Alert>
            <Button component={RouterLink} to="/apps" variant="outlined" endIcon={<ArrowForwardIcon />} sx={{ alignSelf: 'flex-start', minHeight: 44 }}>{t('docs.openApps')}</Button>
          </DocSection>

          <DocSection id="consume" number={3} title={t('docs.consume')}>
            <Typography>{t('docs.consume.intro')}</Typography>
            <Alert severity="info">
              <AlertTitle>{t('docs.consume.addressTitle')}</AlertTitle>
              {t('docs.consume.addressBody')}
            </Alert>
            <Typography variant="h6" component="h3">{t('docs.consume.httpTitle')}</Typography>
            <Typography>{t('docs.consume.httpBody')}</Typography>
            <CodeExample title={t('docs.consume.httpCommand')} code={httpFetchExample} />
            <CodeExample title={t('docs.consume.httpResponse')} code={httpResponseExample} />
            <Typography>{t('docs.consume.httpResult')}</Typography>
            <Alert severity="info">{t('docs.consume.httpErrors')}</Alert>
            <Typography variant="h6" component="h4">{t('docs.consume.tokenTitle')}</Typography>
            <Typography>{t('docs.consume.tokenPermissions')}</Typography>
            <Typography>{t('docs.consume.tokenLogin')}</Typography>
            <CodeExample title={t('docs.consume.tokenLoginCommand')} code={tokenLoginExample} />
            <Typography>{t('docs.consume.tokenCreate')}</Typography>
            <CodeExample title={t('docs.consume.tokenCreateCommand')} code={tokenCreateExample} />
            <Alert severity="info">
              <AlertTitle>{t('docs.consume.tokenOnceTitle')}</AlertTitle>
              {t('docs.consume.tokenOnceBody')}
            </Alert>
            <Typography variant="subtitle1" component="h5" fontWeight={700}>{t('docs.consume.tokenManageTitle')}</Typography>
            <Typography>{t('docs.consume.tokenList')}</Typography>
            <CodeExample title={t('docs.consume.tokenListCommand')} code={tokenListExample} />
            <Typography>{t('docs.consume.tokenRevoke')}</Typography>
            <CodeExample title={t('docs.consume.tokenRevokeCommand')} code={tokenRevokeExample} />
            <Typography variant="body2" color="text.secondary">{t('docs.consume.tokenLifecycle')}</Typography>
            <Divider />
            <Typography variant="h6" component="h3">{t('docs.consume.sdkTitle')}</Typography>
            <Typography>{t('docs.consume.sdkSetup')}</Typography>
            <CodeExample title={t('docs.consume.install')} code={installExample} />
            <Typography>{t('docs.consume.localModule')}</Typography>
            <CodeExample title={t('docs.consume.localCommand')} code={localModuleExample} />
            <Typography>{t('docs.consume.parameters')}</Typography>
            <CodeExample title={t('docs.consume.code')} code={goExample} />
            <Typography>{t('docs.consume.apply')}</Typography>
            <Alert severity="info">{t('docs.consume.watchNote')}</Alert>
          </DocSection>

          <DocSection id="updates" number={4} title={t('docs.updates')}>
            <Typography>{t('docs.updates.intro')}</Typography>
            {updateSteps.map((step) => (
              <Box key={step}>
                <Typography variant="h6" component="h3" gutterBottom>{t(`docs.updates.${step}Title`)}</Typography>
                <Typography>{t(`docs.updates.${step}Body`)}</Typography>
              </Box>
            ))}
            <Button component={RouterLink} to="/cluster/instances" variant="outlined" endIcon={<ArrowForwardIcon />} sx={{ alignSelf: 'flex-start', minHeight: 44 }}>{t('common.instances')}</Button>
            <Alert severity="info">
              <AlertTitle>{t('docs.updates.grayTitle')}</AlertTitle>
              {t('docs.updates.grayBody')}
            </Alert>
          </DocSection>

          <DocSection id="faq" number={5} title={t('docs.faq')}>
            {questions.map((question) => (
              <Box key={question}>
                <Typography variant="h6" component="h3" gutterBottom>{t(`docs.faq.${question}Title`)}</Typography>
                <Typography color="text.secondary">{t(`docs.faq.${question}Body`)}</Typography>
              </Box>
            ))}
          </DocSection>
        </Stack>
      </Box>
    </Stack>
  )
}

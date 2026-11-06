import { useCallback, useEffect, useRef, useState } from 'react'
import {
  Autocomplete,
  Box,
  Chip,
  FormControlLabel,
  Paper,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useNavigate, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DiffViewer } from '../../components/DiffViewer'
import { useToast } from '../../components/ToastProvider'
import { WizardLayout } from '../../components/WizardLayout'
import type { Element } from '../../domain/types'
import { useI18n, type Translator } from '../../i18n'
import { ApiError, apiRequest, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'
import { renderVersionMenuItem } from './VersionMenuItem'
import { getVersionLabel, requestAgentIdOptions, requestComparisonElement, requestInstanceIdOptions, requestVersionOptions, type VersionOption } from './workflowOptions'

const stepKeys = ['operations.version', 'operations.strategy', 'operations.targets', 'operations.reviewDiff', 'operations.impactConfirmation', 'operations.result'] as const

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function buildDetailPath(appId: string, env: string, key: string) {
  return `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`
}

function getStrategyLabel(publishMode: number, t: Translator) {
  return t(publishMode === 1 ? 'operations.publish.gray' : 'operations.publish.full')
}

function isPublishVersionDisabled(option: VersionOption, usingVersion: number | null) {
  if (usingVersion === null) return option.published
  return option.version <= usingVersion
}

const impactGridSx = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
  gap: 3,
}

const fullRowSx = { gridColumn: '1 / -1' }

type PublishWizardFlowProps = {
  appId: string
  env: string
  elementKey: string
}

function PublishWizardFlow({ appId, env, elementKey }: PublishWizardFlowProps) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [activeStep, setActiveStep] = useState(0)
  const [version, setVersion] = useState('')
  const [versionOptions, setVersionOptions] = useState<VersionOption[]>([])
  const [agentOptions, setAgentOptions] = useState<string[]>([])
  const [instanceOptions, setInstanceOptions] = useState<string[]>([])
  const [usingVersion, setUsingVersion] = useState<number | null>(null)
  const [agentIds, setAgentIds] = useState<string[]>([])
  const [instanceIds, setInstanceIds] = useState<string[]>([])
  const [publishMode, setPublishMode] = useState(2)
  const [diffPair, setDiffPair] = useState<{ base: Element; compare: Element } | null>(null)
  const [diffLoaded, setDiffLoaded] = useState(false)
  const [diffLoading, setDiffLoading] = useState(false)
  const [currentVersion, setCurrentVersion] = useState<number | null>(null)
  const [optionsLoading, setOptionsLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
  const detailPath = buildDetailPath(appId, env, elementKey)
  const missingParams = !appId || !env || !elementKey
  const selectedVersion = versionOptions.find((option) => option.value === version)
  const parsedVersion = selectedVersion?.version ?? null
  const versionIsValid = Boolean(selectedVersion && !isPublishVersionDisabled(selectedVersion, usingVersion))
  const requiresExplicitTargets = publishMode === 1
  const hasExplicitTargets = agentIds.length > 0 || instanceIds.length > 0
  const grayTargetsAreValid = !requiresExplicitTargets || hasExplicitTargets
  const targetsValidationMessage = t('operations.publish.targetsRequired')

  const loadOptions = useCallback(async () => {
    if (missingParams) return

    setOptionsLoading(true)

    try {
      const [versionsData, nextAgentOptions, nextInstanceOptions] = await Promise.all([
        requestVersionOptions(appId, env, elementKey),
        requestAgentIdOptions(),
        requestInstanceIdOptions(appId, env, elementKey),
      ])

      if (!mountedRef.current) return

      setVersionOptions(versionsData.options)
      setUsingVersion(versionsData.usingVersion)
      setAgentOptions(nextAgentOptions)
      setInstanceOptions(nextInstanceOptions)
      setVersion('')
      setAgentIds([])
      setInstanceIds([])
    } catch (err) {
      if (!mountedRef.current) return
      setVersionOptions([])
      setUsingVersion(null)
      setAgentOptions([])
      setInstanceOptions([])
      showToast(getErrorMessage(err, t('operations.publish.loadFailed')), 'error')
    } finally {
      if (mountedRef.current) setOptionsLoading(false)
    }
  }, [appId, elementKey, env, missingParams, showToast, t])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ appId, elementKey, env })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadOptions()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [appId, elementKey, env, loadOptions])

  async function handleLoadDiffAndAdvance() {
    if (missingParams || !versionIsValid || diffLoading || submitting) return

    const targetVersion = selectedVersion
    const liveVersion = usingVersion !== null ? versionOptions.find((option) => option.version === usingVersion) : null

    setDiffLoading(true)
    setDiffLoaded(false)
    setDiffPair(null)
    setCurrentVersion(null)

    if (!targetVersion) {
      setDiffLoading(false)
      showToast(t('operations.publish.selectVersion'), 'error')
      return
    }

    if (liveVersion) {
      setCurrentVersion(usingVersion)
      setDiffPair({ base: liveVersion.element, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(3)
      setDiffLoading(false)
      return
    }

    if (usingVersion === null && !targetVersion.published) {
      setDiffPair({ base: { metadata: targetVersion.element.metadata, raw: '', version: 0, published: true }, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(3)
      setDiffLoading(false)
      return
    }

    if (usingVersion === null) {
      setDiffLoading(false)
      showToast(t('operations.publish.noComparison'), 'error')
      return
    }

    try {
      const currentVersionData = await requestComparisonElement(appId, env, elementKey, usingVersion)
      if (!mountedRef.current) return

      if (currentVersionData.version !== usingVersion) {
        showToast(t('operations.publish.noComparison'), 'error')
        return
      }

      setCurrentVersion(usingVersion)
      setDiffPair({ base: currentVersionData, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(3)
    } catch (err) {
      if (!mountedRef.current) return
      showToast(getErrorMessage(err, t('operations.publish.noComparison')), 'error')
    } finally {
      if (mountedRef.current) setDiffLoading(false)
    }
  }

  async function handlePublish() {
    if (missingParams || !versionIsValid || !grayTargetsAreValid || submitting) return

    const payload = {
      version: parsedVersion,
      publishMode,
      ...(publishMode === 1 ? { agentId: agentIds, instanceId: instanceIds } : {}),
    }

    setSubmitting(true)

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(elementKey)}/publish`,
        jsonBody(payload),
      )

      if (!mountedRef.current) return

      showToast(t('operations.publish.queued', { version: parsedVersion ?? '-', strategy: getStrategyLabel(publishMode, t).toLowerCase() }), 'success')
      setActiveStep(5)
    } catch (err) {
      if (!mountedRef.current) return
      showToast(getErrorMessage(err, t('operations.publish.failed')), 'error')
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleNext() {
    if (submitting || diffLoading) return

    if (missingParams) {
      showToast(t('operations.missingParams'), 'error')
      return
    }

    if (activeStep === 0) {
      if (!versionIsValid) {
        showToast(t('operations.publish.selectVersion'), 'error')
        return
      }

      setActiveStep(1)
      return
    }

    if (activeStep === 1) {
      setActiveStep(2)
      return
    }

    if (activeStep === 2) {
      if (!grayTargetsAreValid) {
        showToast(targetsValidationMessage, 'error')
        return
      }

      await handleLoadDiffAndAdvance()
      return
    }

    if (activeStep === 3) {
      setActiveStep(4)
      return
    }

    if (activeStep === 4) {
      await handlePublish()
      return
    }

    navigate(detailPath)
  }

  function handleBack() {
    if (submitting) return
    setActiveStep((currentStep) => Math.max(currentStep - 1, 0))
  }

  const nextLabel = activeStep === 4 ? t('operations.publish') : activeStep === 5 ? t('operations.done') : t('operations.next')
  const nextDisabled =
    missingParams ||
    submitting ||
    diffLoading ||
    (activeStep === 0 && (optionsLoading || !versionIsValid)) ||
    (activeStep === 2 && !grayTargetsAreValid) ||
    (activeStep === 3 && !diffLoaded)
  const backDisabled = activeStep === 0 || activeStep === 5 || submitting || diffLoading

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs
        items={[
          { label: t('operations.apps'), to: '/apps' },
          { label: appId || t('operations.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs` },
          { label: env || t('operations.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
          { label: elementKey || t('operations.unknown'), to: detailPath },
          { label: t('operations.publish') },
        ]}
      />
      <WizardLayout
        title={t('operations.publish.title')}
        steps={stepKeys.map((key) => t(key))}
        activeStep={activeStep}
        onBack={handleBack}
        onNext={() => void handleNext()}
        onCancel={() => navigate(detailPath)}
        nextLabel={nextLabel}
        backDisabled={backDisabled}
        nextDisabled={nextDisabled}
      >
        <Stack spacing={3}>
          {activeStep === 0 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.publish.chooseVersion')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.publish.versionDescription')}
              </Typography>
              <TextField
                select
                label={t('operations.version')}
                value={version}
                onChange={(event) => {
                  setVersion(event.target.value)
                  setDiffLoaded(false)
                  setDiffPair(null)
                  setCurrentVersion(null)
                }}
                required
                fullWidth
                disabled={submitting || diffLoading || missingParams || optionsLoading}
                helperText={optionsLoading ? t('operations.loadingVersions') : versionOptions.some((option) => !isPublishVersionDisabled(option, usingVersion)) ? t('operations.publish.selectNewerVersion') : t('operations.publish.noNewerVersion')}
              >
                {versionOptions.length > 0
                  ? versionOptions.map((option) => renderVersionMenuItem(option, t, isPublishVersionDisabled(option, usingVersion), option.version === usingVersion))
                  : renderVersionMenuItem({ value: '', label: t('operations.publish.noVersionCandidates'), version: 0, published: true, element: { metadata: { key: elementKey } } }, t, true)}
              </TextField>
            </Stack>
          )}

          {activeStep === 1 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.publish.chooseStrategy')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.publish.strategyDescription')}
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <RadioGroup
                  value={String(publishMode)}
                  onChange={(event) => {
                    const nextPublishMode = Number(event.target.value)
                    setPublishMode(nextPublishMode)
                    if (nextPublishMode === 2) {
                      setAgentIds([])
                      setInstanceIds([])
                    }
                  }}
                >
                  <FormControlLabel
                    value="2"
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography fontWeight={600}>{t('operations.publish.full')}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {t('operations.publish.fullDescription')}
                        </Typography>
                      </Box>
                    }
                  />
                  <FormControlLabel
                    value="1"
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography fontWeight={600}>{t('operations.publish.gray')}</Typography>
                        <Typography variant="body2" color="text.secondary">
                          {t('operations.publish.grayDescription')}
                        </Typography>
                      </Box>
                    }
                  />
                </RadioGroup>
              </Paper>
            </Stack>
          )}

          {activeStep === 2 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.publish.defineTargets')}
              </Typography>
              {requiresExplicitTargets ? (
                <>
                  <Typography color="text.secondary">
                    {t('operations.publish.targetsDescription')}
                  </Typography>
                  {!hasExplicitTargets && <Typography color="error.main">{targetsValidationMessage}</Typography>}
                  <Autocomplete
                    multiple
                    options={agentOptions}
                    value={agentIds}
                    onChange={(_, value) => {
                      setAgentIds(value)
                    }}
                    disabled={submitting || diffLoading || missingParams}
                    renderInput={(params) => (
                      <TextField {...params} label={t('operations.publish.agentIds')} helperText={agentOptions.length > 0 ? t('operations.publish.selectAgents') : t('operations.publish.noAgentCandidates')} />
                    )}
                  />
                  <Autocomplete
                    multiple
                    options={instanceOptions}
                    value={instanceIds}
                    onChange={(_, value) => {
                      setInstanceIds(value)
                    }}
                    disabled={submitting || diffLoading || missingParams}
                    renderInput={(params) => (
                      <TextField {...params} label={t('operations.publish.instanceIds')} helperText={instanceOptions.length > 0 ? t('operations.publish.selectInstances') : t('operations.publish.noInstanceCandidates')} />
                    )}
                  />
                </>
              ) : (
                <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                  <Typography color="text.secondary">
                    {t('operations.publish.fullTargetsDescription')}
                  </Typography>
                </Paper>
              )}
            </Stack>
          )}

          {activeStep === 3 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.reviewDiff')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.publish.diffDescription')}
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <Stack spacing={2}>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="caption" color="text.secondary">
                        {t('operations.currentVersion')}
                      </Typography>
                      <Typography fontWeight={600}>{currentVersion ?? '-'}</Typography>
                    </Box>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="caption" color="text.secondary">
                        {t('operations.publishVersion')}
                      </Typography>
                      <Typography fontWeight={600}>{selectedVersion ? getVersionLabel(selectedVersion, t) : '-'}</Typography>
                    </Box>
                  </Stack>
                  {diffLoaded && diffPair ? (
                    <DiffViewer
                      oldValue={decodeRaw(diffPair.base.raw)}
                      newValue={decodeRaw(diffPair.compare.raw)}
                      baseLabel={t('operations.currentVersionLabel', { version: currentVersion ?? '-' })}
                      compareLabel={selectedVersion ? getVersionLabel(selectedVersion, t) : t('operations.publishVersion')}
                      contentType={diffPair.compare.metadata?.contentType}
                    />
                  ) : diffLoading ? (
                    <Typography color="text.secondary">{t('operations.loadingDiff')}</Typography>
                  ) : (
                    <Typography color="text.secondary">{t('operations.noDiff')}</Typography>
                  )}
                </Stack>
              </Paper>
            </Stack>
          )}

          {activeStep === 4 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.impactConfirmation')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.publish.impactDescription')}
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <Box data-testid="publish-impact-grid" sx={impactGridSx}>
                  <Box data-testid="publish-impact-app" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.app')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {appId || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-env" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.env')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {env || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-key" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.key')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {elementKey || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-version" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.version')}
                    </Typography>
                    <Typography fontWeight={600}>{selectedVersion ? getVersionLabel(selectedVersion, t) : '-'}</Typography>
                  </Box>
                  <Box data-testid="publish-impact-strategy" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.strategy')}
                    </Typography>
                    <Typography fontWeight={600}>{getStrategyLabel(publishMode, t)}</Typography>
                  </Box>
                  <Box data-testid="publish-impact-agent-ids" sx={fullRowSx}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.publish.agentIds')}
                    </Typography>
                    <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                      {agentIds.length > 0 ? agentIds.map((agentId) => <Chip key={agentId} label={agentId} variant="outlined" />) : <Typography color="text.secondary">{t('operations.publish.noAgentTargets')}</Typography>}
                    </Stack>
                  </Box>
                  <Box data-testid="publish-impact-instance-ids" sx={fullRowSx}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.publish.instanceIds')}
                    </Typography>
                    <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                      {instanceIds.length > 0 ? instanceIds.map((instanceId) => <Chip key={instanceId} label={instanceId} variant="outlined" />) : <Typography color="text.secondary">{t('operations.publish.noInstanceTargets')}</Typography>}
                    </Stack>
                  </Box>
                </Box>
              </Paper>
            </Stack>
          )}

          {activeStep === 5 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.result')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.publish.resultDescription')}
              </Typography>
            </Stack>
          )}
        </Stack>
      </WizardLayout>
    </Stack>
  )
}

export function PublishWizardPage() {
  const { appId = '', env = '', key = '' } = useParams()
  return <PublishWizardFlow key={`${appId}/${env}/${key}`} appId={appId} env={env} elementKey={key} />
}

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
import { ApiError, apiRequest, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'
import { renderVersionMenuItem } from './VersionMenuItem'
import { requestAgentIdOptions, requestComparisonElement, requestInstanceIdOptions, requestVersionOptions, type VersionOption } from './workflowOptions'

const steps = ['Version', 'Strategy', 'Targets', 'Review diff', 'Impact confirmation', 'Result']

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function buildDetailPath(appId: string, env: string, key: string) {
  return `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`
}

function getStrategyLabel(publishMode: number) {
  return publishMode === 1 ? 'Gray publish' : 'Full publish'
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
  const targetsValidationMessage = 'Gray publish requires at least one agent or instance target.'

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
      showToast(getErrorMessage(err, 'failed to load workflow options'), 'error')
    } finally {
      if (mountedRef.current) setOptionsLoading(false)
    }
  }, [appId, elementKey, env, missingParams, showToast])

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
      showToast('select a version before continuing', 'error')
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
      showToast('no comparison data is available to review this publish.', 'error')
      return
    }

    try {
      const currentVersionData = await requestComparisonElement(appId, env, elementKey, usingVersion)
      if (!mountedRef.current) return

      if (currentVersionData.version !== usingVersion) {
        showToast('no comparison data is available to review this publish.', 'error')
        return
      }

      setCurrentVersion(usingVersion)
      setDiffPair({ base: currentVersionData, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(3)
    } catch (err) {
      if (!mountedRef.current) return
      showToast(getErrorMessage(err, 'no comparison data is available to review this publish.'), 'error')
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

      showToast(`Version ${parsedVersion} was queued for ${getStrategyLabel(publishMode).toLowerCase()}.`, 'success')
      setActiveStep(5)
    } catch (err) {
      if (!mountedRef.current) return
      showToast(getErrorMessage(err, 'failed to publish element'), 'error')
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleNext() {
    if (submitting || diffLoading) return

    if (missingParams) {
      showToast('missing app, environment, or key', 'error')
      return
    }

    if (activeStep === 0) {
      if (!versionIsValid) {
        showToast('select a version before continuing', 'error')
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

  const nextLabel = activeStep === 4 ? 'Publish' : activeStep === 5 ? 'Done' : 'Next'
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
          { label: 'Apps', to: '/apps' },
          { label: appId || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs` },
          { label: env || 'unknown', to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
          { label: elementKey || 'unknown', to: detailPath },
          { label: 'Publish' },
        ]}
      />
      <WizardLayout
        title="Publish element"
        steps={steps}
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
                Choose the version to publish
              </Typography>
              <Typography color="text.secondary">
                Pick the draft or historical version you want to ship into the selected environment.
              </Typography>
              <TextField
                select
                label="Version"
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
                helperText={optionsLoading ? 'Loading versions.' : versionOptions.some((option) => !isPublishVersionDisabled(option, usingVersion)) ? 'Select a newer version to publish.' : 'No newer version available.'}
              >
                {versionOptions.length > 0
                  ? versionOptions.map((option) => renderVersionMenuItem(option, isPublishVersionDisabled(option, usingVersion), option.version === usingVersion))
                  : renderVersionMenuItem({ value: '', label: 'No version candidates', version: 0, published: true, element: { metadata: { key: elementKey } } }, true)}
              </TextField>
            </Stack>
          )}

          {activeStep === 1 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                Choose a rollout strategy
              </Typography>
              <Typography color="text.secondary">
                Use a full publish for broad rollout, or gray publish for a narrower targeted release.
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
                        <Typography fontWeight={600}>Full publish</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Publish to the standard audience with no explicit targeting requirements.
                        </Typography>
                      </Box>
                    }
                  />
                  <FormControlLabel
                    value="1"
                    control={<Radio />}
                    label={
                      <Box>
                        <Typography fontWeight={600}>Gray publish</Typography>
                        <Typography variant="body2" color="text.secondary">
                          Limit rollout with explicit agent or instance targeting.
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
                Define rollout targets
              </Typography>
              {requiresExplicitTargets ? (
                <>
                  <Typography color="text.secondary">
                    Select agent or instance targets for this element. Gray publish requires at least one target before you can continue.
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
                      <TextField {...params} label="Agent IDs" helperText={agentOptions.length > 0 ? 'Select one or more agent targets.' : 'No agent candidates found.'} />
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
                      <TextField {...params} label="Instance IDs" helperText={instanceOptions.length > 0 ? 'Select one or more instance targets.' : 'No instance candidates found for this element.'} />
                    )}
                  />
                </>
              ) : (
                <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                  <Typography color="text.secondary">
                    Full publish does not use explicit agent or instance targets. Continue to review the rollout payload.
                  </Typography>
                </Paper>
              )}
            </Stack>
          )}

          {activeStep === 3 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                Review diff
              </Typography>
              <Typography color="text.secondary">
                Inspect the change between the current live version and the selected publish version before proceeding.
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <Stack spacing={2}>
                  <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="caption" color="text.secondary">
                        Current version
                      </Typography>
                      <Typography fontWeight={600}>{currentVersion ?? '-'}</Typography>
                    </Box>
                    <Box sx={{ flex: 1 }}>
                      <Typography variant="caption" color="text.secondary">
                        Publish version
                      </Typography>
                      <Typography fontWeight={600}>{selectedVersion?.label || '-'}</Typography>
                    </Box>
                  </Stack>
                  {diffLoaded && diffPair ? (
                    <DiffViewer
                      oldValue={decodeRaw(diffPair.base.raw)}
                      newValue={decodeRaw(diffPair.compare.raw)}
                      baseLabel={`Current v${currentVersion ?? '-'}`}
                      compareLabel={selectedVersion?.label || 'Publish version'}
                      contentType={diffPair.compare.metadata?.contentType}
                    />
                  ) : diffLoading ? (
                    <Typography color="text.secondary">Loading diff…</Typography>
                  ) : (
                    <Typography color="text.secondary">No diff loaded yet.</Typography>
                  )}
                </Stack>
              </Paper>
            </Stack>
          )}

          {activeStep === 4 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                Impact confirmation
              </Typography>
              <Typography color="text.secondary">
                Review the publish payload before dispatching it to the control plane.
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <Box data-testid="publish-impact-grid" sx={impactGridSx}>
                  <Box data-testid="publish-impact-app" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      App
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {appId || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-env" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      Env
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {env || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-key" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      Key
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {elementKey || '-'}
                    </Typography>
                  </Box>
                  <Box data-testid="publish-impact-version" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      Version
                    </Typography>
                    <Typography fontWeight={600}>{selectedVersion?.label || '-'}</Typography>
                  </Box>
                  <Box data-testid="publish-impact-strategy" sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      Strategy
                    </Typography>
                    <Typography fontWeight={600}>{getStrategyLabel(publishMode)}</Typography>
                  </Box>
                  <Box data-testid="publish-impact-agent-ids" sx={fullRowSx}>
                    <Typography variant="caption" color="text.secondary">
                      Agent IDs
                    </Typography>
                    <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                      {agentIds.length > 0 ? agentIds.map((agentId) => <Chip key={agentId} label={agentId} variant="outlined" />) : <Typography color="text.secondary">No explicit agent targets.</Typography>}
                    </Stack>
                  </Box>
                  <Box data-testid="publish-impact-instance-ids" sx={fullRowSx}>
                    <Typography variant="caption" color="text.secondary">
                      Instance IDs
                    </Typography>
                    <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                      {instanceIds.length > 0 ? instanceIds.map((instanceId) => <Chip key={instanceId} label={instanceId} variant="outlined" />) : <Typography color="text.secondary">No explicit instance targets.</Typography>}
                    </Stack>
                  </Box>
                </Box>
              </Paper>
            </Stack>
          )}

          {activeStep === 5 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                Result
              </Typography>
              <Typography color="text.secondary">
                Select Done to return to the element detail page and continue reviewing versions or operations.
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

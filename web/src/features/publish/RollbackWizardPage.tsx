import { useCallback, useEffect, useRef, useState } from 'react'
import { Box, Paper, Stack, TextField, Typography } from '@mui/material'
import { useNavigate, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DiffViewer } from '../../components/DiffViewer'
import { useToast } from '../../components/ToastProvider'
import { WizardLayout } from '../../components/WizardLayout'
import type { Element } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, jsonBody } from '../../lib/api'
import { decodeRaw } from '../../lib/raw'
import { renderVersionMenuItem } from './VersionMenuItem'
import { getVersionLabel, requestComparisonElement, requestVersionOptions, type VersionOption } from './workflowOptions'

const stepKeys = ['operations.version', 'operations.reviewDiff', 'operations.impactConfirmation', 'operations.result'] as const

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function buildDetailPath(appId: string, env: string, key: string) {
  return `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`
}

function isRollbackVersionDisabled(option: VersionOption, usingVersion: number | null) {
  if (!option.published) return true
  return usingVersion !== null && option.version >= usingVersion
}

const impactGridSx = {
  display: 'grid',
  gridTemplateColumns: 'repeat(3, minmax(0, 1fr))',
  gap: 3,
}

type RollbackWizardFlowProps = {
  appId: string
  env: string
  elementKey: string
}

function RollbackWizardFlow({ appId, env, elementKey }: RollbackWizardFlowProps) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { showToast } = useToast()
  const [activeStep, setActiveStep] = useState(0)
  const [version, setVersion] = useState('')
  const [versionOptions, setVersionOptions] = useState<VersionOption[]>([])
  const [usingVersion, setUsingVersion] = useState<number | null>(null)
  const [optionsLoading, setOptionsLoading] = useState(false)
  const [currentVersion, setCurrentVersion] = useState<number | null>(null)
  const [diffPair, setDiffPair] = useState<{ base: Element; compare: Element } | null>(null)
  const [diffLoaded, setDiffLoaded] = useState(false)
  const [diffLoading, setDiffLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
  const diffRequestSeqRef = useRef(0)
  const detailPath = buildDetailPath(appId, env, elementKey)
  const missingParams = !appId || !env || !elementKey
  const selectedVersion = versionOptions.find((option) => option.value === version)
  const parsedVersion = selectedVersion?.version ?? null
  const versionIsValid = Boolean(selectedVersion && !isRollbackVersionDisabled(selectedVersion, usingVersion))

  const loadOptions = useCallback(async () => {
    if (missingParams) return

    setOptionsLoading(true)

    try {
      const data = await requestVersionOptions(appId, env, elementKey)
      if (!mountedRef.current) return
      setVersionOptions(data.options)
      setUsingVersion(data.usingVersion)
      setVersion('')
    } catch (err) {
      if (!mountedRef.current) return
      setVersionOptions([])
      setUsingVersion(null)
      showToast(getErrorMessage(err, t('operations.rollback.loadFailed')), 'error')
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
    const requestId = diffRequestSeqRef.current + 1
    diffRequestSeqRef.current = requestId

    setDiffLoading(true)
    setDiffLoaded(false)
    setDiffPair(null)
    setCurrentVersion(null)

    if (!targetVersion) {
      setDiffLoading(false)
      showToast(t('operations.rollback.selectVersion'), 'error')
      return
    }

    if (liveVersion) {
      if (targetVersion.version >= liveVersion.version) {
        setDiffLoading(false)
        showToast(t('operations.rollback.olderRequired'), 'error')
        return
      }

      setCurrentVersion(usingVersion)
      setDiffPair({ base: liveVersion.element, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(1)
      setDiffLoading(false)
      return
    }

    if (usingVersion === null) {
      setDiffLoading(false)
      showToast(t('operations.rollback.noComparison'), 'error')
      return
    }

    try {
      const currentVersionData = await requestComparisonElement(appId, env, elementKey, usingVersion)
      if (!mountedRef.current || diffRequestSeqRef.current !== requestId) return

      if (currentVersionData.version !== usingVersion) {
        showToast(t('operations.rollback.noComparison'), 'error')
        return
      }

      if (targetVersion.version >= currentVersionData.version) {
        showToast(t('operations.rollback.olderRequired'), 'error')
        return
      }

      setCurrentVersion(usingVersion)
      setDiffPair({ base: currentVersionData, compare: targetVersion.element })
      setDiffLoaded(true)
      setActiveStep(1)
    } catch (err) {
      if (!mountedRef.current || diffRequestSeqRef.current !== requestId) return
      showToast(getErrorMessage(err, t('operations.rollback.noComparison')), 'error')
    } finally {
      if (mountedRef.current && diffRequestSeqRef.current === requestId) setDiffLoading(false)
    }
  }

  async function handleRollback() {
    if (missingParams || !versionIsValid || submitting) return

    setSubmitting(true)

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(elementKey)}/rollback`,
        jsonBody({ version: parsedVersion }),
      )

      if (!mountedRef.current) return

      showToast(t('operations.rollback.success', { version: parsedVersion ?? '-' }), 'success')
      setActiveStep(3)
    } catch (err) {
      if (!mountedRef.current) return
      showToast(getErrorMessage(err, t('operations.rollback.failed')), 'error')
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
        showToast(t('operations.rollback.selectVersion'), 'error')
        return
      }

      await handleLoadDiffAndAdvance()
      return
    }

    if (activeStep === 1) {
      setActiveStep(2)
      return
    }

    if (activeStep === 2) {
      await handleRollback()
      return
    }

    navigate(detailPath)
  }

  function handleBack() {
    if (submitting) return
    setActiveStep((currentStep) => Math.max(currentStep - 1, 0))
  }

  const nextLabel = activeStep === 2 ? t('operations.rollback') : activeStep === 3 ? t('operations.done') : t('operations.next')
  const nextDisabled = missingParams || submitting || diffLoading || (activeStep === 0 && (optionsLoading || !versionIsValid)) || (activeStep === 1 && !diffLoaded)
  const backDisabled = activeStep === 0 || activeStep === 3 || submitting || diffLoading

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs
        items={[
          { label: t('operations.apps'), to: '/apps' },
          { label: appId || t('operations.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs` },
          { label: env || t('operations.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
          { label: elementKey || t('operations.unknown'), to: detailPath },
          { label: t('operations.rollback') },
        ]}
      />
      <WizardLayout
        title={t('operations.rollback.title')}
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
                {t('operations.rollback.chooseVersion')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.rollback.versionDescription')}
              </Typography>
              <TextField
                select
                label={t('operations.targetVersion')}
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
                helperText={optionsLoading ? t('operations.loadingVersions') : versionOptions.some((option) => !isRollbackVersionDisabled(option, usingVersion)) ? t('operations.rollback.selectOlderVersion') : t('operations.rollback.noPublishedVersion')}
              >
                {versionOptions.length > 0
                  ? versionOptions.map((option) => renderVersionMenuItem(option, t, isRollbackVersionDisabled(option, usingVersion), option.version === usingVersion))
                  : renderVersionMenuItem({ value: '', label: t('operations.rollback.noTargetVersions'), version: 0, published: true, element: { metadata: { key: elementKey } } }, t, true)}
              </TextField>
            </Stack>
          )}

          {activeStep === 1 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.reviewDiff')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.rollback.diffDescription')}
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
                        {t('operations.targetVersion')}
                      </Typography>
                      <Typography fontWeight={600}>{selectedVersion ? getVersionLabel(selectedVersion, t) : '-'}</Typography>
                    </Box>
                  </Stack>
                  {diffLoaded && diffPair ? (
                    <DiffViewer
                      oldValue={decodeRaw(diffPair.base.raw)}
                      newValue={decodeRaw(diffPair.compare.raw)}
                      baseLabel={t('operations.currentVersionLabel', { version: currentVersion ?? '-' })}
                      compareLabel={selectedVersion ? getVersionLabel(selectedVersion, t) : t('operations.target')}
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

          {activeStep === 2 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.impactConfirmation')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.rollback.impactDescription')}
              </Typography>
              <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
                <Box data-testid="rollback-impact-grid" sx={impactGridSx}>
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.app')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {appId || '-'}
                    </Typography>
                  </Box>
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.env')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {env || '-'}
                    </Typography>
                  </Box>
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.key')}
                    </Typography>
                    <Typography fontWeight={600} sx={{ overflowWrap: 'anywhere' }}>
                      {elementKey || '-'}
                    </Typography>
                  </Box>
                  <Box sx={{ minWidth: 0 }}>
                    <Typography variant="caption" color="text.secondary">
                      {t('operations.targetVersion')}
                    </Typography>
                    <Typography fontWeight={600}>{selectedVersion ? getVersionLabel(selectedVersion, t) : '-'}</Typography>
                  </Box>
                </Box>
              </Paper>
            </Stack>
          )}

          {activeStep === 3 && (
            <Stack spacing={2.5}>
              <Typography variant="h6" component="h2">
                {t('operations.result')}
              </Typography>
              <Typography color="text.secondary">
                {t('operations.rollback.resultDescription')}
              </Typography>
            </Stack>
          )}
        </Stack>
      </WizardLayout>
    </Stack>
  )
}

export function RollbackWizardPage() {
  const { appId = '', env = '', key = '' } = useParams()
  return <RollbackWizardFlow key={`${appId}/${env}/${key}`} appId={appId} env={env} elementKey={key} />
}

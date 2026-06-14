import { useEffect, useRef, useState } from 'react'
import { Alert, Box, Paper, Stack, TextField, Typography } from '@mui/material'
import { useNavigate, useParams } from 'react-router-dom'
import { LoadingState } from '../../components/StateView'
import { WizardLayout } from '../../components/WizardLayout'
import type { DiffResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'

const steps = ['Version', 'Review diff', 'Impact confirmation', 'Result']
const maxVersion = 4_294_967_295n

function getErrorMessage(error: unknown, fallback: string) {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error && error.message.trim()) return error.message
  return fallback
}

function buildDetailPath(appId: string, env: string, key: string) {
  return `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`
}

async function requestDiff(appId: string, env: string, key: string, base: number, compare: number) {
  return apiRequest<DiffResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}/diff${buildQuery({
      base,
      compare,
    })}`,
  )
}

function resolveBaseVersion(diff: DiffResponse) {
  const currentVersion = diff.base?.version ?? diff.base?.metadata?.usingVersion
  if (typeof currentVersion === 'number' && Number.isFinite(currentVersion) && currentVersion > 0) {
    return currentVersion
  }

  return null
}

type RollbackWizardFlowProps = {
  appId: string
  env: string
  elementKey: string
}

function RollbackWizardFlow({ appId, env, elementKey }: RollbackWizardFlowProps) {
  const navigate = useNavigate()
  const [activeStep, setActiveStep] = useState(0)
  const [version, setVersion] = useState('')
  const [currentVersion, setCurrentVersion] = useState<number | null>(null)
  const [diffText, setDiffText] = useState('')
  const [diffLoaded, setDiffLoaded] = useState(false)
  const [diffLoading, setDiffLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [resultMessage, setResultMessage] = useState('')
  const mountedRef = useRef(false)

  useEffect(() => {
    mountedRef.current = true
    return () => {
      mountedRef.current = false
    }
  }, [])

  const trimmedVersion = version.trim()
  const parsedVersion = /^[1-9]\d*$/.test(trimmedVersion) ? BigInt(trimmedVersion) : null
  const versionIsValid = parsedVersion !== null && parsedVersion <= maxVersion
  const detailPath = buildDetailPath(appId, env, elementKey)
  const missingParams = !appId || !env || !elementKey

  async function handleLoadDiffAndAdvance() {
    if (missingParams || !versionIsValid || diffLoading || submitting) return

    setDiffLoading(true)
    setError('')
    setDiffLoaded(false)
    setDiffText('')
    setCurrentVersion(null)
    setActiveStep(1)

    try {
      if (parsedVersion === null) {
        throw new Error('Enter a valid rollback target version.')
      }

      const diff = await requestDiff(appId, env, elementKey, 0, Number(parsedVersion))
      const baseVersion = resolveBaseVersion(diff)

      if (!baseVersion) {
        throw new Error('Unable to determine the current version for diff review.')
      }

      if (parsedVersion >= BigInt(baseVersion)) {
        throw new Error('Rollback target must be older than the current live version.')
      }

      if (!mountedRef.current) return

      setCurrentVersion(baseVersion)
      setDiffText(diff.diff || '')
      setDiffLoaded(true)
      setActiveStep(1)
    } catch (err) {
      if (!mountedRef.current) return
      setCurrentVersion(null)
      setDiffText('')
      setDiffLoaded(false)
      setActiveStep(0)
      setError(getErrorMessage(err, 'failed to load diff'))
    } finally {
      if (mountedRef.current) setDiffLoading(false)
    }
  }

  async function handleRollback() {
    if (missingParams || !versionIsValid || submitting) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(elementKey)}/rollback`,
        jsonBody({ version: Number(parsedVersion) }),
      )

      if (!mountedRef.current) return

      setResultMessage(`Rollback to version ${trimmedVersion} completed successfully.`)
      setActiveStep(3)
    } catch (err) {
      if (!mountedRef.current) return
      setError(getErrorMessage(err, 'failed to rollback element'))
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleNext() {
    if (submitting || diffLoading) return

    setError('')

    if (missingParams) {
      setError('missing app, environment, or key')
      return
    }

    if (activeStep === 0) {
      if (!versionIsValid) {
        setError('enter a valid version number before continuing')
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
    if (submitting || diffLoading) return
    setError('')
    setActiveStep((currentStep) => Math.max(currentStep - 1, 0))
  }

  const nextLabel = activeStep === 2 ? 'Rollback' : activeStep === 3 ? 'Done' : 'Next'
  const nextDisabled = missingParams || submitting || diffLoading || (activeStep === 0 && !versionIsValid) || (activeStep === 1 && !diffLoaded)
  const backDisabled = activeStep === 0 || activeStep === 3 || submitting || diffLoading

  return (
    <WizardLayout
      title="Rollback element"
      steps={steps}
      activeStep={activeStep}
      onBack={handleBack}
      onNext={() => void handleNext()}
      nextLabel={nextLabel}
      backDisabled={backDisabled}
      nextDisabled={nextDisabled}
    >
      <Stack spacing={3}>
        {error && <Alert severity="error">{error}</Alert>}

        {activeStep === 0 && (
          <Stack spacing={2.5}>
            <Typography variant="h6" component="h2">
              Choose the target version
            </Typography>
            <Typography color="text.secondary">
              Enter the version you want this element to return to. The next step will load a current-versus-target diff.
            </Typography>
            <TextField
              label="Target version"
              type="number"
              value={version}
              onChange={(event) => {
                setVersion(event.target.value)
                setDiffLoaded(false)
                setDiffText('')
                setCurrentVersion(null)
              }}
              required
              fullWidth
              disabled={submitting || diffLoading || missingParams}
              error={trimmedVersion !== '' && !versionIsValid}
              helperText={trimmedVersion !== '' && !versionIsValid ? 'Version must be a positive integer within uint32 range.' : 'Example: 5'}
            />
          </Stack>
        )}

        {activeStep === 1 && (
          <Stack spacing={2.5}>
            <Typography variant="h6" component="h2">
              Review diff
            </Typography>
            <Typography color="text.secondary">
              Inspect the change between the current live version and the requested rollback target before proceeding.
            </Typography>
            <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
              <Stack spacing={2}>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Current version</Typography>
                    <Typography fontWeight={600}>{currentVersion ?? '-'}</Typography>
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Target version</Typography>
                    <Typography fontWeight={600}>{trimmedVersion || '-'}</Typography>
                  </Box>
                </Stack>
                {diffLoading ? (
                  <LoadingState label="Loading diff" />
                ) : diffLoaded ? (
                  <TextField
                    label="Diff"
                    value={diffText || 'No differences returned for this comparison.'}
                    multiline
                    minRows={12}
                    fullWidth
                    InputProps={{ readOnly: true }}
                  />
                ) : (
                  <Typography color="text.secondary">No diff loaded yet.</Typography>
                )}
              </Stack>
            </Paper>
          </Stack>
        )}

        {activeStep === 2 && (
          <Stack spacing={2.5}>
            <Typography variant="h6" component="h2">
              Impact confirmation
            </Typography>
            <Typography color="text.secondary">
              Confirm the rollback target before writing the request to the backend.
            </Typography>
            <Paper variant="outlined" sx={{ p: 2.5, borderRadius: 3 }}>
              <Stack spacing={2}>
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">App</Typography>
                    <Typography fontWeight={600}>{appId || '-'}</Typography>
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Env</Typography>
                    <Typography fontWeight={600}>{env || '-'}</Typography>
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Key</Typography>
                    <Typography fontWeight={600}>{elementKey || '-'}</Typography>
                  </Box>
                </Stack>
                <Box>
                  <Typography variant="caption" color="text.secondary">Target version</Typography>
                  <Typography fontWeight={600}>{trimmedVersion || '-'}</Typography>
                </Box>
              </Stack>
            </Paper>
          </Stack>
        )}

        {activeStep === 3 && (
          <Stack spacing={2.5}>
            <Typography variant="h6" component="h2">
              Result
            </Typography>
            <Alert severity="success">{resultMessage || 'Element rollback request completed successfully.'}</Alert>
            <Typography color="text.secondary">
              Select Done to return to the element detail page and continue reviewing the element state.
            </Typography>
          </Stack>
        )}
      </Stack>
    </WizardLayout>
  )
}

export function RollbackWizardPage() {
  const { appId = '', env = '', key = '' } = useParams()
  return <RollbackWizardFlow key={`${appId}/${env}/${key}`} appId={appId} env={env} elementKey={key} />
}

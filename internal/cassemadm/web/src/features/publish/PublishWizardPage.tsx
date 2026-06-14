import { useEffect, useRef, useState } from 'react'
import {
  Alert,
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
import { WizardLayout } from '../../components/WizardLayout'
import { ApiError, apiRequest, jsonBody } from '../../lib/api'

const steps = ['Version', 'Strategy', 'Targets', 'Impact confirmation', 'Result']
const maxVersion = 4_294_967_295n

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function splitCsv(value: string) {
  return value
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function buildDetailPath(appId: string, env: string, key: string) {
  return `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(key)}`
}

function getStrategyLabel(publishMode: number) {
  return publishMode === 1 ? 'Gray publish' : 'Full publish'
}

type PublishWizardFlowProps = {
  appId: string
  env: string
  elementKey: string
}

function PublishWizardFlow({ appId, env, elementKey }: PublishWizardFlowProps) {
  const navigate = useNavigate()
  const [activeStep, setActiveStep] = useState(0)
  const [version, setVersion] = useState('')
  const [publishMode, setPublishMode] = useState(2)
  const [agentIdsCsv, setAgentIdsCsv] = useState('')
  const [instanceIdsCsv, setInstanceIdsCsv] = useState('')
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
  const agentIds = splitCsv(agentIdsCsv)
  const instanceIds = splitCsv(instanceIdsCsv)
  const requiresExplicitTargets = publishMode === 1
  const hasExplicitTargets = agentIds.length > 0 || instanceIds.length > 0
  const grayTargetsAreValid = !requiresExplicitTargets || hasExplicitTargets
  const targetsValidationMessage = 'Gray publish requires at least one agent or instance target.'
  const detailPath = buildDetailPath(appId, env, elementKey)
  const missingParams = !appId || !env || !elementKey

  async function handlePublish() {
    if (missingParams || !versionIsValid || !grayTargetsAreValid || submitting) return

    const payload = {
      version: Number(parsedVersion),
      publishMode,
      ...(publishMode === 1 ? { agentId: agentIds, instanceId: instanceIds } : {}),
    }

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(elementKey)}/publish`,
        jsonBody(payload),
      )

      if (!mountedRef.current) return

      setResultMessage(`Version ${trimmedVersion} was queued for ${getStrategyLabel(publishMode).toLowerCase()}.`)
      setActiveStep(4)
    } catch (err) {
      if (!mountedRef.current) return
      setError(getErrorMessage(err, 'failed to publish element'))
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleNext() {
    if (submitting) return

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

      setActiveStep(1)
      return
    }

    if (activeStep === 1) {
      setActiveStep(2)
      return
    }

    if (activeStep === 2) {
      if (!grayTargetsAreValid) {
        setError(targetsValidationMessage)
        return
      }

      setActiveStep(3)
      return
    }

    if (activeStep === 3) {
      await handlePublish()
      return
    }

    navigate(detailPath)
  }

  function handleBack() {
    if (submitting) return
    setError('')
    setActiveStep((currentStep) => Math.max(currentStep - 1, 0))
  }

  const nextLabel = activeStep === 3 ? 'Publish' : activeStep === 4 ? 'Done' : 'Next'
  const nextDisabled =
    missingParams ||
    submitting ||
    (activeStep === 0 && !versionIsValid) ||
    (activeStep === 2 && !grayTargetsAreValid)
  const backDisabled = activeStep === 0 || activeStep === 4 || submitting

  return (
    <WizardLayout
      title="Publish element"
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
              Choose the version to publish
            </Typography>
            <Typography color="text.secondary">
              Pick the draft or historical version you want to ship into the selected environment.
            </Typography>
            <TextField
              label="Version"
              type="number"
              value={version}
              onChange={(event) => setVersion(event.target.value)}
              required
              fullWidth
              disabled={submitting || missingParams}
              error={trimmedVersion !== '' && !versionIsValid}
              helperText={trimmedVersion !== '' && !versionIsValid ? 'Version must be a positive integer within uint32 range.' : 'Example: 12'}
            />
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
                    setAgentIdsCsv('')
                    setInstanceIdsCsv('')
                  }
                  if (error === targetsValidationMessage) setError('')
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
                  Provide comma-separated identifiers. Gray publish requires at least one agent or instance target before you can continue.
                </Typography>
                <TextField
                  label="Agent IDs"
                  value={agentIdsCsv}
                  onChange={(event) => {
                    setAgentIdsCsv(event.target.value)
                    if (error === targetsValidationMessage) setError('')
                  }}
                  fullWidth
                  multiline
                  minRows={2}
                  disabled={submitting || missingParams}
                  error={!hasExplicitTargets}
                  helperText={!hasExplicitTargets ? targetsValidationMessage : 'Example: agent-a, agent-b'}
                />
                <TextField
                  label="Instance IDs"
                  value={instanceIdsCsv}
                  onChange={(event) => {
                    setInstanceIdsCsv(event.target.value)
                    if (error === targetsValidationMessage) setError('')
                  }}
                  fullWidth
                  multiline
                  minRows={2}
                  disabled={submitting || missingParams}
                  error={!hasExplicitTargets}
                  helperText={!hasExplicitTargets ? targetsValidationMessage : 'Example: instance-01, instance-02'}
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
              Impact confirmation
            </Typography>
            <Typography color="text.secondary">
              Review the publish payload before dispatching it to the control plane.
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
                <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2}>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Version</Typography>
                    <Typography fontWeight={600}>{trimmedVersion || '-'}</Typography>
                  </Box>
                  <Box sx={{ flex: 1 }}>
                    <Typography variant="caption" color="text.secondary">Strategy</Typography>
                    <Typography fontWeight={600}>{getStrategyLabel(publishMode)}</Typography>
                  </Box>
                </Stack>
                <Box>
                  <Typography variant="caption" color="text.secondary">Agent IDs</Typography>
                  <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                    {agentIds.length > 0 ? agentIds.map((agentId) => <Chip key={agentId} label={agentId} variant="outlined" />) : <Typography color="text.secondary">No explicit agent targets.</Typography>}
                  </Stack>
                </Box>
                <Box>
                  <Typography variant="caption" color="text.secondary">Instance IDs</Typography>
                  <Stack direction="row" spacing={1} useFlexGap flexWrap="wrap" sx={{ mt: 1 }}>
                    {instanceIds.length > 0 ? instanceIds.map((instanceId) => <Chip key={instanceId} label={instanceId} variant="outlined" />) : <Typography color="text.secondary">No explicit instance targets.</Typography>}
                  </Stack>
                </Box>
              </Stack>
            </Paper>
          </Stack>
        )}

        {activeStep === 4 && (
          <Stack spacing={2.5}>
            <Typography variant="h6" component="h2">
              Result
            </Typography>
            <Alert severity="success">{resultMessage || 'Element publish request completed successfully.'}</Alert>
            <Typography color="text.secondary">
              Select Done to return to the element detail page and continue reviewing versions or operations.
            </Typography>
          </Stack>
        )}
      </Stack>
    </WizardLayout>
  )
}

export function PublishWizardPage() {
  const { appId = '', env = '', key = '' } = useParams()
  return <PublishWizardFlow key={`${appId}/${env}/${key}`} appId={appId} env={env} elementKey={key} />
}

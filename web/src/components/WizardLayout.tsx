import { useState, type ReactNode } from 'react'
import { Box, Button, Paper, Stack, Step, StepLabel, Stepper, Typography } from '@mui/material'
import { DangerConfirmDialog } from './DangerConfirmDialog'

type WizardLayoutProps = {
  title: string
  steps: string[]
  activeStep: number
  children: ReactNode
  onBack?: () => void
  onNext?: () => void
  onCancel?: () => void
  nextLabel?: string
  backDisabled?: boolean
  nextDisabled?: boolean
}

const workflowFrameSx = {
  maxWidth: 1080,
  width: '100%',
  mx: 'auto',
}

export function WizardLayout({
  title,
  steps,
  activeStep,
  children,
  onBack,
  onNext,
  onCancel,
  nextLabel = 'Next',
  backDisabled = false,
  nextDisabled = false,
}: WizardLayoutProps) {
  const [cancelOpen, setCancelOpen] = useState(false)
  const workflowInset = `calc(100% / ${steps.length * 2})`

  return (
    <Stack spacing={3} alignItems="center">
      <Stack
        data-testid="wizard-title-actions"
        direction={{ xs: 'column', md: 'row' }}
        spacing={2}
        justifyContent="space-between"
        alignItems={{ md: 'center' }}
        sx={workflowFrameSx}
      >
        <Box>
          <Typography
            variant="overline"
            sx={{
              color: 'text.secondary',
              letterSpacing: '0.22em',
              fontWeight: 700,
            }}
          >
            Element workflow
          </Typography>
          <Typography variant="h4" component="h1" sx={{ mt: 0.5 }}>
            {title}
          </Typography>
        </Box>
        {onCancel && (
          <Button color="warning" variant="outlined" onClick={() => setCancelOpen(true)}>
            Cancel
          </Button>
        )}
      </Stack>

      <DangerConfirmDialog
        open={cancelOpen}
        title="Cancel workflow"
        description="Current workflow progress will be discarded and you will return to the element detail page."
        confirmLabel="Confirm cancel"
        onClose={() => setCancelOpen(false)}
        onConfirm={() => {
          setCancelOpen(false)
          onCancel?.()
        }}
      />

      <Paper
        data-testid="wizard-surface"
        variant="outlined"
        sx={{
          ...workflowFrameSx,
          p: { xs: 2, md: 3 },
          borderRadius: 3,
          backgroundImage: 'linear-gradient(180deg, rgba(18,18,18,0.02) 0%, rgba(18,18,18,0) 100%)',
        }}
      >
        <Stack spacing={3}>
          <Box data-testid="wizard-stepper-frame" sx={workflowFrameSx}>
            <Stepper activeStep={activeStep} alternativeLabel sx={{ overflowX: 'auto', pb: 1 }}>
              {steps.map((step) => (
                <Step key={step}>
                  <StepLabel>{step}</StepLabel>
                </Step>
              ))}
            </Stepper>
          </Box>

          <Box
            data-testid="wizard-content-frame"
            sx={{
              ...workflowFrameSx,
              minHeight: 320,
              px: workflowInset,
              py: 1,
            }}
          >
            {children}
          </Box>

          <Box data-testid="wizard-actions-frame" sx={{ ...workflowFrameSx, px: workflowInset }}>
            <Stack direction="row" justifyContent="flex-end" spacing={1.5} data-testid="wizard-actions">
              <Button variant="text" onClick={onBack} disabled={!onBack || backDisabled}>
                Back
              </Button>
              <Button variant="contained" onClick={onNext} disabled={!onNext || nextDisabled}>
                {nextLabel}
              </Button>
            </Stack>
          </Box>
        </Stack>
      </Paper>
    </Stack>
  )
}

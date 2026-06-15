import type { ReactNode } from 'react'
import { Box, Button, Paper, Stack, Step, StepLabel, Stepper, Typography } from '@mui/material'

type WizardLayoutProps = {
  title: string
  steps: string[]
  activeStep: number
  children: ReactNode
  onBack?: () => void
  onNext?: () => void
  nextLabel?: string
  backDisabled?: boolean
  nextDisabled?: boolean
}

export function WizardLayout({
  title,
  steps,
  activeStep,
  children,
  onBack,
  onNext,
  nextLabel = 'Next',
  backDisabled = false,
  nextDisabled = false,
}: WizardLayoutProps) {
  return (
    <Stack spacing={3}>
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

      <Paper
        variant="outlined"
        sx={{
          p: { xs: 2, md: 3 },
          borderRadius: 3,
          backgroundImage: 'linear-gradient(180deg, rgba(18,18,18,0.02) 0%, rgba(18,18,18,0) 100%)',
        }}
      >
        <Stack spacing={3}>
          <Stepper activeStep={activeStep} alternativeLabel sx={{ overflowX: 'auto', pb: 1 }}>
            {steps.map((step) => (
              <Step key={step}>
                <StepLabel>{step}</StepLabel>
              </Step>
            ))}
          </Stepper>

          <Box
            sx={{
              minHeight: 320,
              px: { xs: 0, md: 1 },
              py: 1,
            }}
          >
            {children}
          </Box>

          <Stack direction="row" justifyContent="flex-end" spacing={1.5} data-testid="wizard-actions">
            <Button variant="text" onClick={onBack} disabled={!onBack || backDisabled}>
              Back
            </Button>
            <Button variant="contained" onClick={onNext} disabled={!onNext || nextDisabled}>
              {nextLabel}
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Stack>
  )
}

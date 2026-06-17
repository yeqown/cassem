import { useMemo, useState, type FormEvent } from 'react'
import { Alert, Box, Button, MenuItem, Paper, Stack, TextField, Typography } from '@mui/material'
import { roleOptions, type RoleValue } from '../../domain/types'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'

type Feedback = {
  severity: 'success' | 'error'
  message: string
}

type AclFormState = {
  account: string
  role: RoleValue
  domains: string
}

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function parseDomains(input: string) {
  return input
    .split(/[\n,]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function AclPage() {
  const [form, setForm] = useState<AclFormState>({ account: '', role: 'appdeveloper', domains: '' })
  const [submitting, setSubmitting] = useState(false)
  const [feedback, setFeedback] = useState<Feedback | null>(null)

  const account = form.account.trim()
  const domains = useMemo(() => parseDomains(form.domains), [form.domains])

  function updateField(field: keyof AclFormState, value: string) {
    setForm((current) => ({ ...current, [field]: value }))
  }

  function clearFeedback() {
    setFeedback(null)
  }

  async function submitAclAction(action: 'assign' | 'revoke') {
    if (!account) return

    setSubmitting(true)
    clearFeedback()

    try {
      await apiRequest<void>(`/api/account/acl/${action}${buildQuery({ account, role: form.role, domain: domains.length ? domains : undefined })}`)
      setFeedback({
        severity: 'success',
        message: action === 'assign' ? 'Role assigned successfully.' : 'Role revoked successfully.',
      })
    } catch (error) {
      setFeedback({
        severity: 'error',
        message: getErrorMessage(error, action === 'assign' ? 'Failed to assign role.' : 'Failed to revoke role.'),
      })
    } finally {
      setSubmitting(false)
    }
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
  }

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">
          ACL
        </Typography>
        <Typography color="text.secondary">
          Assign or revoke fixed backend roles. Leave domains empty to use the backend default cluster domain.
        </Typography>
      </Box>

      {feedback && <Alert severity={feedback.severity}>{feedback.message}</Alert>}

      <Paper component="form" onSubmit={handleSubmit} sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">
            Role binding
          </Typography>
          <TextField
            label="Account"
            value={form.account}
            onChange={(event) => updateField('account', event.target.value)}
            required
            fullWidth
            disabled={submitting}
          />
          <TextField
            select
            label="Role"
            value={form.role}
            onChange={(event) => updateField('role', event.target.value as RoleValue)}
            fullWidth
            disabled={submitting}
          >
            {roleOptions.map((role) => (
              <MenuItem key={role.value} value={role.value}>
                {role.label}
              </MenuItem>
            ))}
          </TextField>
          <TextField
            label="Domains"
            value={form.domains}
            onChange={(event) => updateField('domains', event.target.value)}
            fullWidth
            multiline
            minRows={3}
            disabled={submitting}
            helperText="Enter one domain per line or separate multiple domains with commas."
          />
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <Button variant="contained" disabled={submitting || !account} onClick={() => void submitAclAction('assign')}>
              Assign role
            </Button>
            <Button variant="outlined" color="warning" disabled={submitting || !account} onClick={() => void submitAclAction('revoke')}>
              Revoke role
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Stack>
  )
}

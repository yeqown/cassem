import { useMemo, useState, type FormEvent } from 'react'
import { Box, Button, MenuItem, Paper, Stack, TextField, Typography } from '@mui/material'
import { useToast } from '../../components/ToastProvider'
import { roleOptions, type RoleValue } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery } from '../../lib/api'

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
  const { t } = useI18n()
  const { showToast } = useToast()
  const [form, setForm] = useState<AclFormState>({ account: '', role: 'appdeveloper', domains: '' })
  const [submitting, setSubmitting] = useState(false)

  const account = form.account.trim()
  const domains = useMemo(() => parseDomains(form.domains), [form.domains])

  function updateField(field: keyof AclFormState, value: string) {
    setForm((current) => ({ ...current, [field]: value }))
  }

  async function submitAclAction(action: 'assign' | 'revoke') {
    if (!account) return

    setSubmitting(true)

    try {
      await apiRequest<void>(`/api/account/acl/${action}${buildQuery({ account, role: form.role, domain: domains.length ? domains : undefined })}`)
      showToast(action === 'assign' ? t('management.acl.assigned') : t('management.acl.revoked'), 'success')
    } catch (error) {
      showToast(getErrorMessage(error, action === 'assign' ? t('management.acl.assignFailed') : t('management.acl.revokeFailed')), 'error')
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
          {t('management.acl.title')}
        </Typography>
        <Typography color="text.secondary">
          {t('management.acl.description')}
        </Typography>
      </Box>

      <Paper component="form" onSubmit={handleSubmit} sx={{ p: 3 }}>
        <Stack spacing={2}>
          <Typography variant="h6" component="h2">
            {t('management.acl.roleBinding')}
          </Typography>
          <TextField
            label={t('management.users.account')}
            value={form.account}
            onChange={(event) => updateField('account', event.target.value)}
            required
            fullWidth
            disabled={submitting}
          />
          <TextField
            select
            label={t('management.users.role')}
            value={form.role}
            onChange={(event) => updateField('role', event.target.value as RoleValue)}
            fullWidth
            disabled={submitting}
          >
            {roleOptions.map((role) => (
              <MenuItem key={role.value} value={role.value}>
                {t(`management.roles.${role.value}`)}
              </MenuItem>
            ))}
          </TextField>
          <TextField
            label={t('management.acl.domains')}
            value={form.domains}
            onChange={(event) => updateField('domains', event.target.value)}
            fullWidth
            multiline
            minRows={3}
            disabled={submitting}
            helperText={t('management.acl.domainsHelp')}
          />
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
            <Button variant="contained" disabled={submitting || !account} onClick={() => void submitAclAction('assign')}>
              {t('management.acl.assign')}
            </Button>
            <Button variant="outlined" color="warning" disabled={submitting || !account} onClick={() => void submitAclAction('revoke')}>
              {t('management.acl.revoke')}
            </Button>
          </Stack>
        </Stack>
      </Paper>
    </Stack>
  )
}

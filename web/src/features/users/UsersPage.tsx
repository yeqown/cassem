import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings'
import BlockIcon from '@mui/icons-material/Block'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import KeyIcon from '@mui/icons-material/Key'
import LockResetIcon from '@mui/icons-material/LockReset'
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts'
import PersonAddAlt1Icon from '@mui/icons-material/PersonAddAlt1'
import PersonIcon from '@mui/icons-material/Person'
import SecurityIcon from '@mui/icons-material/Security'
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  InputAdornment,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useErrorState } from '../../components/useErrorState'
import { useToast } from '../../components/ToastProvider'
import type { DomainOptionsResponse, RoleValue, User, UserAccessBinding, UserAccessResponse, UsersResponse } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { roleOptions } from '../../domain/types'

type CreateFormState = {
  account: string
  nickname: string
  password: string
}

type ScopeMode = 'cluster' | 'app-all' | 'app-env'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function isUserDisabled(status?: number) {
  return status === 1
}

function isSuperadminUser(user: User) {
  return user.roles?.includes('superadmin') || user.accessSummary?.some((binding) => binding.role === 'superadmin') || false
}

const assignableRoleOptions = roleOptions.filter((role) => role.value !== 'superadmin')

function UserStatusChip({ status }: { status?: number }) {
  const { t } = useI18n()
  return isUserDisabled(status) ? (
    <Chip size="small" color="error" variant="outlined" icon={<BlockIcon />} label={t('management.users.disabled')} />
  ) : (
    <Chip size="small" color="success" variant="outlined" icon={<CheckCircleIcon />} label={t('management.users.enabled')} />
  )
}

async function requestUsers() {
  return apiRequest<UsersResponse>(`/api/account/users${buildQuery({ limit: 100 })}`)
}

async function requestUserAccess(account: string) {
  return apiRequest<UserAccessResponse>(`/api/account/users/${encodeURIComponent(account)}/acl`)
}

async function requestDomainOptions() {
  return apiRequest<DomainOptionsResponse>('/api/account/acl/domains')
}

function buildDomain(scopeMode: ScopeMode, app: string, env: string) {
  if (scopeMode === 'cluster') return 'cluster'
  if (scopeMode === 'app-all') return `${app}/*`
  return `${app}/${env}`
}

export function UsersPage() {
  const { t, formatNumber } = useI18n()
  const { showToast } = useToast()
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useErrorState()
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState<CreateFormState>({ account: '', nickname: '', password: '' })
  const [disableTarget, setDisableTarget] = useState<User | null>(null)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [resetPassword, setResetPassword] = useState('')
  const [accessTarget, setAccessTarget] = useState<User | null>(null)
  const [accessBindings, setAccessBindings] = useState<UserAccessBinding[]>([])
  const [accessLoading, setAccessLoading] = useState(false)
  const [domainOptions, setDomainOptions] = useState<string[]>([])
  const [roleValue, setRoleValue] = useState<RoleValue>('admin')
  const [scopeMode, setScopeMode] = useState<ScopeMode>('cluster')
  const [selectedApp, setSelectedApp] = useState('')
  const [selectedEnv, setSelectedEnv] = useState('')
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')

  const appOptions = useMemo(
    () => domainOptions.filter((item) => item !== 'cluster' && item.endsWith('/*')).map((item) => item.slice(0, -2)),
    [domainOptions],
  )
  const envOptions = useMemo(() => {
    if (!selectedApp) return []
    return domainOptions
      .filter((item) => item.startsWith(`${selectedApp}/`) && !item.endsWith('/*'))
      .map((item) => item.slice(selectedApp.length + 1))
  }, [domainOptions, selectedApp])

  const loadUsers = useCallback(async () => {
    const requestId = ++requestSeq.current
    setLoading(true)

    try {
      const data = await requestUsers()
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setUsers(data.users || [])
      setError('')
    } catch (err) {
      if (!mountedRef.current || requestId !== requestSeq.current) return
      setUsers([])
      setError(getErrorMessage(err, t('management.users.loadFailed')))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [setError, t])

  const loadDomainOptions = useCallback(async () => {
    try {
      const data = await requestDomainOptions()
      if (!mountedRef.current) return
      setDomainOptions(data.domains || [])
    } catch (err) {
      if (!mountedRef.current) return
      setError(getErrorMessage(err, t('management.users.loadDomainsFailed')))
    }
  }, [setError, t])

  useEffect(() => {
    mountedRef.current = true

    const loadKey = 'users-and-domains'
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadUsers()
        void loadDomainOptions()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [loadUsers, loadDomainOptions])

  function roleLabel(role: string) {
    const option = roleOptions.find((item) => item.value === role)
    return option ? t(`management.roles.${option.value}`) : role
  }

  function updateCreateField(field: keyof CreateFormState, value: string) {
    setCreateForm((current) => ({ ...current, [field]: value }))
  }

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setCreateForm({ account: '', nickname: '', password: '' })
  }

  function closeDisableDialog() {
    if (submitting) return
    setDisableTarget(null)
  }

  function closeResetDialog() {
    if (submitting) return
    setResetTarget(null)
    setResetPassword('')
  }

  function closeAccessDialog() {
    if (submitting || accessLoading) return
    setAccessTarget(null)
    setAccessBindings([])
    setRoleValue('admin')
    setScopeMode('cluster')
    setSelectedApp('')
    setSelectedEnv('')
  }

  async function handleAddUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const account = createForm.account.trim()
    const nickname = createForm.nickname.trim()
    const password = createForm.password
    if (!account || !nickname || !password.trim()) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>('/api/account/add', jsonBody({ account, password, nickname }))
      setCreateOpen(false)
      setCreateForm({ account: '', nickname: '', password: '' })
      showToast(t('management.users.added'), 'success')
      await loadUsers()
    } catch (err) {
      showToast(getErrorMessage(err, t('management.users.addFailed')), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDisableUser() {
    const account = disableTarget?.account
    if (!account) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(`/api/account/disable${buildQuery({ account })}`)
      setDisableTarget(null)
      showToast(t('management.users.disabledSuccess'), 'success')
      await loadUsers()
    } catch (err) {
      showToast(getErrorMessage(err, t('management.users.disableFailed')), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleResetPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const account = resetTarget?.account?.trim()
    if (!account || !resetPassword.trim()) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>('/api/account/reset', jsonBody({ account, password: resetPassword }))
      setResetTarget(null)
      setResetPassword('')
      showToast(t('management.users.passwordReset'), 'success')
      await loadUsers()
    } catch (err) {
      showToast(getErrorMessage(err, t('management.users.resetFailed')), 'error')
    } finally {
      setSubmitting(false)
    }
  }

  async function handleOpenAccess(user: User) {
    setAccessTarget(user)
    setAccessLoading(true)
    setError('')

    try {
      const data = await requestUserAccess(user.account)
      if (!mountedRef.current) return
      setAccessBindings(data.bindings || [])
    } catch (err) {
      if (!mountedRef.current) return
      setError(getErrorMessage(err, t('management.users.loadAccessFailed')))
    } finally {
      if (mountedRef.current) setAccessLoading(false)
    }
  }

  async function refreshAccessBindings(account: string) {
    try {
      const data = await requestUserAccess(account)
      if (!mountedRef.current) return false
      setAccessBindings(data.bindings || [])
      return true
    } catch (err) {
      if (mountedRef.current) {
        showToast(getErrorMessage(err, t('management.users.refreshAccessFailed')), 'error')
      }
      return false
    }
  }

  async function handleAddBinding() {
    const account = accessTarget?.account
    if (!account) return
    if (scopeMode !== 'cluster' && !selectedApp) return
    if (scopeMode === 'app-env' && !selectedEnv) return

    const domain = buildDomain(scopeMode, selectedApp, selectedEnv)

    setSubmitting(true)
    setError('')
    try {
      await apiRequest<void>(`/api/account/acl/assign${buildQuery({ account, role: roleValue, domain })}`)
      showToast(t('management.acl.assigned'), 'success')
      await refreshAccessBindings(account)
      void loadUsers()
    } catch (err) {
      showToast(getErrorMessage(err, t('management.acl.assignFailed')), 'error')
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleRevokeBinding(binding: UserAccessBinding) {
    const account = accessTarget?.account
    if (!account) return

    setSubmitting(true)
    setError('')
    try {
      await apiRequest<void>(`/api/account/acl/revoke${buildQuery({ account, role: binding.role, domain: binding.domain })}`)
      showToast(t('management.acl.revoked'), 'success')
      await refreshAccessBindings(account)
      void loadUsers()
    } catch (err) {
      showToast(getErrorMessage(err, t('management.acl.revokeFailed')), 'error')
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} justifyContent="space-between" alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <ManageAccountsIcon color="primary" />
            <Typography variant="h4" component="h1">
              {t('management.users.title')}
            </Typography>
          </Stack>
          <Typography color="text.secondary">{t('management.users.description')}</Typography>
        </Box>
        <Button variant="contained" startIcon={<PersonAddAlt1Icon />} onClick={() => setCreateOpen(true)} disabled={submitting}>
          {t('management.users.add')}
        </Button>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      <DangerConfirmDialog
        open={Boolean(disableTarget)}
        title={t('management.users.disableUser')}
        description={t('management.users.disableDescription', { account: disableTarget?.account || '' })}
        confirmLabel={t('management.users.disable')}
        loading={submitting}
        onClose={closeDisableDialog}
        onConfirm={() => void handleDisableUser()}
      />

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleAddUser(event)}>
          <DialogTitle>{t('management.users.add')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                label={t('management.users.account')}
                value={createForm.account}
                onChange={(event) => updateCreateField('account', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText={t('management.users.accountHelp')}
                InputProps={{ startAdornment: <InputAdornment position="start"><PersonIcon fontSize="small" /></InputAdornment> }}
              />
              <TextField
                label={t('management.users.nickname')}
                value={createForm.nickname}
                onChange={(event) => updateCreateField('nickname', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText={t('management.users.nicknameHelp')}
                InputProps={{ startAdornment: <InputAdornment position="start"><ManageAccountsIcon fontSize="small" /></InputAdornment> }}
              />
              <TextField
                label={t('management.users.password')}
                type="password"
                value={createForm.password}
                onChange={(event) => updateCreateField('password', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText={t('management.users.passwordHelp')}
                InputProps={{ startAdornment: <InputAdornment position="start"><KeyIcon fontSize="small" /></InputAdornment> }}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>{t('management.common.cancel')}</Button>
            <Button type="submit" variant="contained" disabled={submitting || !createForm.account.trim() || !createForm.nickname.trim() || !createForm.password.trim()}>{t('management.common.create')}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={Boolean(resetTarget)} onClose={closeResetDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleResetPassword(event)}>
          <DialogTitle>{t('management.users.resetPassword')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField label={t('management.users.account')} value={resetTarget?.account || ''} fullWidth disabled InputProps={{ startAdornment: <InputAdornment position="start"><PersonIcon fontSize="small" /></InputAdornment> }} />
              <TextField
                label={t('management.users.password')}
                type="password"
                value={resetPassword}
                onChange={(event) => setResetPassword(event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText={t('management.users.newPasswordHelp')}
                InputProps={{ startAdornment: <InputAdornment position="start"><LockResetIcon fontSize="small" /></InputAdornment> }}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeResetDialog} disabled={submitting}>{t('management.common.cancel')}</Button>
            <Button type="submit" variant="contained" disabled={submitting || !resetPassword.trim()}>{t('management.users.reset')}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={Boolean(accessTarget)} onClose={closeAccessDialog} fullWidth maxWidth="md">
        <DialogTitle>{t('management.users.manageAccess')}</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField label={t('management.users.account')} value={accessTarget?.account || ''} fullWidth disabled InputProps={{ startAdornment: <InputAdornment position="start"><SecurityIcon fontSize="small" /></InputAdornment> }} />
            {accessLoading ? (
              <LoadingState label={t('management.users.loadingAccess')} />
            ) : accessBindings.length === 0 ? (
              <EmptyState title={t('management.users.emptyAccess')} description={t('management.users.emptyAccessDescription')} />
            ) : (
              <TableContainer component={Paper} variant="outlined">
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>{t('management.users.role')}</TableCell>
                      <TableCell>{t('management.users.domain')}</TableCell>
                      <TableCell align="right">{t('management.common.actions')}</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {accessBindings.map((binding) => (
                      <TableRow key={`${binding.role}:${binding.domain}`} hover>
                        <TableCell>{roleLabel(binding.role)}</TableCell>
                        <TableCell>{binding.domain}</TableCell>
                        <TableCell align="right">
                          <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} disabled={submitting || binding.role === 'superadmin'} onClick={() => void handleRevokeBinding(binding)}>
                            {t('management.users.revoke')}
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            )}
            <Paper variant="outlined" sx={{ p: 2 }}>
              <Stack spacing={2}>
                <Typography variant="subtitle1">{t('management.users.addBinding')}</Typography>
                <FormControl fullWidth>
                  <InputLabel id="acl-role-label">{t('management.users.role')}</InputLabel>
                  <Select labelId="acl-role-label" value={roleValue} label={t('management.users.role')} onChange={(event) => setRoleValue(event.target.value as RoleValue)}>
                    {assignableRoleOptions.map((role) => (
                      <MenuItem key={role.value} value={role.value}>{t(`management.roles.${role.value}`)}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl fullWidth>
                  <InputLabel id="scope-mode-label">{t('management.users.scope')}</InputLabel>
                  <Select labelId="scope-mode-label" value={scopeMode} label={t('management.users.scope')} onChange={(event) => setScopeMode(event.target.value as ScopeMode)}>
                    <MenuItem value="cluster">{t('management.users.cluster')}</MenuItem>
                    <MenuItem value="app-all">{t('management.users.entireApp')}</MenuItem>
                    <MenuItem value="app-env">{t('management.users.specificEnvironment')}</MenuItem>
                  </Select>
                </FormControl>
                {scopeMode !== 'cluster' && (
                  <FormControl fullWidth>
                    <InputLabel id="scope-app-label">{t('management.users.app')}</InputLabel>
                    <Select labelId="scope-app-label" value={selectedApp} label={t('management.users.app')} onChange={(event) => { setSelectedApp(event.target.value); setSelectedEnv('') }}>
                      {appOptions.map((app) => (
                        <MenuItem key={app} value={app}>{app}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
                {scopeMode === 'app-env' && (
                  <FormControl fullWidth>
                    <InputLabel id="scope-env-label">{t('management.common.environment')}</InputLabel>
                    <Select labelId="scope-env-label" value={selectedEnv} label={t('management.common.environment')} onChange={(event) => setSelectedEnv(event.target.value)}>
                      {envOptions.map((env) => (
                        <MenuItem key={env} value={env}>{env}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
                <TextField
                  label={t('management.users.computedDomain')}
                  value={scopeMode === 'cluster' ? 'cluster' : selectedApp ? buildDomain(scopeMode, selectedApp, selectedEnv) : ''}
                  fullWidth
                  helperText={t('management.users.domainHelp')}
                  InputProps={{ readOnly: true }}
                />
                <Stack direction="row" justifyContent="flex-end">
                  <Button
                    variant="contained"
                    startIcon={<AdminPanelSettingsIcon />}
                    disabled={submitting || (scopeMode !== 'cluster' && !selectedApp) || (scopeMode === 'app-env' && !selectedEnv)}
                    onClick={() => void handleAddBinding()}
                  >
                    {t('management.users.addBinding')}
                  </Button>
                </Stack>
              </Stack>
            </Paper>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeAccessDialog} disabled={submitting || accessLoading}>{t('management.common.close')}</Button>
        </DialogActions>
      </Dialog>

      {loading ? (
        <LoadingState label={t('management.users.loading')} />
      ) : error.message ? null : users.length === 0 ? (
        <EmptyState title={t('management.users.empty')} description={t('management.users.emptyDescription')} />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>{t('management.users.nickname')}</TableCell>
                <TableCell>{t('management.users.account')}</TableCell>
                <TableCell>{t('management.common.status')}</TableCell>
                <TableCell>{t('management.users.roles')}</TableCell>
                <TableCell>{t('management.users.bindings')}</TableCell>
                <TableCell align="right">{t('management.common.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((user) => {
                const disabled = isUserDisabled(user.status)
                const protectedUser = isSuperadminUser(user)
                return (
                  <TableRow key={user.account} hover>
                    <TableCell>{user.nickname || '-'}</TableCell>
                    <TableCell>{user.account}</TableCell>
                    <TableCell><UserStatusChip status={user.status} /></TableCell>
                    <TableCell>{user.roles?.length ? user.roles.map(roleLabel).join(', ') : '-'}</TableCell>
                    <TableCell>{formatNumber(user.bindingCount ?? user.accessSummary?.length ?? 0)}</TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Button size="small" startIcon={<AdminPanelSettingsIcon />} disabled={submitting || protectedUser} onClick={() => void handleOpenAccess(user)}>
                          {t('management.users.manageAccess')}
                        </Button>
                        <Button color="warning" size="small" startIcon={<DeleteOutlineIcon />} disabled={submitting || disabled || protectedUser} onClick={() => setDisableTarget(user)}>
                          {t('management.users.disable')}
                        </Button>
                        <Button size="small" startIcon={<LockResetIcon />} disabled={submitting || protectedUser} onClick={() => setResetTarget(user)}>
                          {t('management.users.resetPassword')}
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  )
}

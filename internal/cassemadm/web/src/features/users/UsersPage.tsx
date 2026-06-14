import { useCallback, useEffect, useMemo, useRef, useState, type FormEvent } from 'react'
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import KeyIcon from '@mui/icons-material/Key'
import LockResetIcon from '@mui/icons-material/LockReset'
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts'
import PersonAddAlt1Icon from '@mui/icons-material/PersonAddAlt1'
import PersonIcon from '@mui/icons-material/Person'
import SecurityIcon from '@mui/icons-material/Security'
import {
  Alert,
  Box,
  Button,
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
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import type { DomainOptionsResponse, RoleValue, User, UserAccessBinding, UserAccessResponse, UsersResponse } from '../../domain/types'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { roleOptions } from '../../domain/types'

type Feedback = {
  severity: 'success' | 'error'
  message: string
}

type CreateFormState = {
  account: string
  nickname: string
  password: string
}

type ScopeMode = 'cluster' | 'app-all' | 'app-env'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function getStatusLabel(status?: number) {
  return status === 1 ? 'disabled' : 'normal'
}

function roleLabel(role: string) {
  return roleOptions.find((item) => item.value === role)?.label || role
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
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [feedback, setFeedback] = useState<Feedback | null>(null)
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState<CreateFormState>({ account: '', nickname: '', password: '' })
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
      setError(getErrorMessage(err, 'failed to load users'))
    } finally {
      if (mountedRef.current && requestId === requestSeq.current) setLoading(false)
    }
  }, [])

  const loadDomainOptions = useCallback(async () => {
    try {
      const data = await requestDomainOptions()
      if (!mountedRef.current) return
      setDomainOptions(data.domains || [])
    } catch (err) {
      if (!mountedRef.current) return
      setError(getErrorMessage(err, 'failed to load ACL domains'))
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true

    queueMicrotask(() => {
      void loadUsers()
      void loadDomainOptions()
    })

    return () => {
      mountedRef.current = false
    }
  }, [loadUsers, loadDomainOptions])

  function updateCreateField(field: keyof CreateFormState, value: string) {
    setCreateForm((current) => ({ ...current, [field]: value }))
  }

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setCreateForm({ account: '', nickname: '', password: '' })
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
    setFeedback(null)
    setError('')

    try {
      await apiRequest<void>('/api/account/add', jsonBody({ account, password, nickname }))
      setCreateOpen(false)
      setCreateForm({ account: '', nickname: '', password: '' })
      setFeedback({ severity: 'success', message: 'User added successfully.' })
      await loadUsers()
    } catch (err) {
      setFeedback({ severity: 'error', message: getErrorMessage(err, 'Failed to add user.') })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleDisableUser(account: string) {
    if (!window.confirm(`Disable user ${account}?`)) return

    setSubmitting(true)
    setFeedback(null)
    setError('')

    try {
      await apiRequest<void>(`/api/account/disable${buildQuery({ account })}`)
      setFeedback({ severity: 'success', message: 'User disabled successfully.' })
      await loadUsers()
    } catch (err) {
      setFeedback({ severity: 'error', message: getErrorMessage(err, 'Failed to disable user.') })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleResetPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const account = resetTarget?.account?.trim()
    if (!account || !resetPassword.trim()) return

    setSubmitting(true)
    setFeedback(null)
    setError('')

    try {
      await apiRequest<void>('/api/account/reset', jsonBody({ account, password: resetPassword }))
      setResetTarget(null)
      setResetPassword('')
      setFeedback({ severity: 'success', message: 'Password reset successfully.' })
      await loadUsers()
    } catch (err) {
      setFeedback({ severity: 'error', message: getErrorMessage(err, 'Failed to reset password.') })
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
      setError(getErrorMessage(err, 'Failed to load access bindings.'))
    } finally {
      if (mountedRef.current) setAccessLoading(false)
    }
  }

  async function handleAddBinding() {
    const account = accessTarget?.account
    if (!account) return
    if (scopeMode !== 'cluster' && !selectedApp) return
    if (scopeMode === 'app-env' && !selectedEnv) return

    const domain = buildDomain(scopeMode, selectedApp, selectedEnv)

    setSubmitting(true)
    setFeedback(null)
    setError('')
    try {
      await apiRequest<void>(`/api/account/acl/assign${buildQuery({ account, role: roleValue, domain })}`)
      const data = await requestUserAccess(account)
      if (!mountedRef.current) return
      setAccessBindings(data.bindings || [])
      setFeedback({ severity: 'success', message: 'Role assigned successfully.' })
      await loadUsers()
    } catch (err) {
      setFeedback({ severity: 'error', message: getErrorMessage(err, 'Failed to assign role.') })
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleRevokeBinding(binding: UserAccessBinding) {
    const account = accessTarget?.account
    if (!account) return

    setSubmitting(true)
    setFeedback(null)
    setError('')
    try {
      await apiRequest<void>(`/api/account/acl/revoke${buildQuery({ account, role: binding.role, domain: binding.domain })}`)
      const data = await requestUserAccess(account)
      if (!mountedRef.current) return
      setAccessBindings(data.bindings || [])
      setFeedback({ severity: 'success', message: 'Role revoked successfully.' })
      await loadUsers()
    } catch (err) {
      setFeedback({ severity: 'error', message: getErrorMessage(err, 'Failed to revoke role.') })
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
              Users
            </Typography>
          </Stack>
          <Typography color="text.secondary">Review accounts, add new users, manage access, disable access, and reset passwords.</Typography>
        </Box>
        <Button variant="contained" startIcon={<PersonAddAlt1Icon />} onClick={() => setCreateOpen(true)} disabled={submitting}>
          Add user
        </Button>
      </Stack>

      {feedback && <Alert severity={feedback.severity}>{feedback.message}</Alert>}
      {error && <ErrorState message={error} />}

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleAddUser(event)}>
          <DialogTitle>Add user</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField
                label="Account"
                value={createForm.account}
                onChange={(event) => updateCreateField('account', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText="User login account. Must be a valid email address."
                InputProps={{ startAdornment: <InputAdornment position="start"><PersonIcon fontSize="small" /></InputAdornment> }}
              />
              <TextField
                label="Nickname"
                value={createForm.nickname}
                onChange={(event) => updateCreateField('nickname', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText="Display name shown in the admin UI and account menu."
                InputProps={{ startAdornment: <InputAdornment position="start"><ManageAccountsIcon fontSize="small" /></InputAdornment> }}
              />
              <TextField
                label="Password"
                type="password"
                value={createForm.password}
                onChange={(event) => updateCreateField('password', event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText="Initial password for the new account."
                InputProps={{ startAdornment: <InputAdornment position="start"><KeyIcon fontSize="small" /></InputAdornment> }}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>Cancel</Button>
            <Button type="submit" variant="contained" disabled={submitting || !createForm.account.trim() || !createForm.nickname.trim() || !createForm.password.trim()}>Create</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={Boolean(resetTarget)} onClose={closeResetDialog} fullWidth maxWidth="sm">
        <Box component="form" onSubmit={(event) => void handleResetPassword(event)}>
          <DialogTitle>Reset password</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField label="Account" value={resetTarget?.account || ''} fullWidth disabled InputProps={{ startAdornment: <InputAdornment position="start"><PersonIcon fontSize="small" /></InputAdornment> }} />
              <TextField
                label="Password"
                type="password"
                value={resetPassword}
                onChange={(event) => setResetPassword(event.target.value)}
                required
                fullWidth
                disabled={submitting}
                helperText="New password to assign to this user."
                InputProps={{ startAdornment: <InputAdornment position="start"><LockResetIcon fontSize="small" /></InputAdornment> }}
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeResetDialog} disabled={submitting}>Cancel</Button>
            <Button type="submit" variant="contained" disabled={submitting || !resetPassword.trim()}>Reset</Button>
          </DialogActions>
        </Box>
      </Dialog>

      <Dialog open={Boolean(accessTarget)} onClose={closeAccessDialog} fullWidth maxWidth="md">
        <DialogTitle>Manage access</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField label="Account" value={accessTarget?.account || ''} fullWidth disabled InputProps={{ startAdornment: <InputAdornment position="start"><SecurityIcon fontSize="small" /></InputAdornment> }} />
            {accessLoading ? (
              <LoadingState label="Loading access bindings" />
            ) : accessBindings.length === 0 ? (
              <EmptyState title="No access bindings" description="Add a role binding for this user." />
            ) : (
              <TableContainer component={Paper} variant="outlined">
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell>Role</TableCell>
                      <TableCell>Domain</TableCell>
                      <TableCell align="right">Actions</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {accessBindings.map((binding) => (
                      <TableRow key={`${binding.role}:${binding.domain}`} hover>
                        <TableCell>{roleLabel(binding.role)}</TableCell>
                        <TableCell>{binding.domain}</TableCell>
                        <TableCell align="right">
                          <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} disabled={submitting} onClick={() => void handleRevokeBinding(binding)}>
                            Revoke
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
                <Typography variant="subtitle1">Add binding</Typography>
                <FormControl fullWidth>
                  <InputLabel id="acl-role-label">Role</InputLabel>
                  <Select labelId="acl-role-label" value={roleValue} label="Role" onChange={(event) => setRoleValue(event.target.value as RoleValue)}>
                    {roleOptions.map((role) => (
                      <MenuItem key={role.value} value={role.value}>{role.label}</MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl fullWidth>
                  <InputLabel id="scope-mode-label">Scope</InputLabel>
                  <Select labelId="scope-mode-label" value={scopeMode} label="Scope" onChange={(event) => setScopeMode(event.target.value as ScopeMode)}>
                    <MenuItem value="cluster">Cluster</MenuItem>
                    <MenuItem value="app-all">Entire app</MenuItem>
                    <MenuItem value="app-env">Specific environment</MenuItem>
                  </Select>
                </FormControl>
                {scopeMode !== 'cluster' && (
                  <FormControl fullWidth>
                    <InputLabel id="scope-app-label">App</InputLabel>
                    <Select labelId="scope-app-label" value={selectedApp} label="App" onChange={(event) => { setSelectedApp(event.target.value); setSelectedEnv('') }}>
                      {appOptions.map((app) => (
                        <MenuItem key={app} value={app}>{app}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
                {scopeMode === 'app-env' && (
                  <FormControl fullWidth>
                    <InputLabel id="scope-env-label">Environment</InputLabel>
                    <Select labelId="scope-env-label" value={selectedEnv} label="Environment" onChange={(event) => setSelectedEnv(event.target.value)}>
                      {envOptions.map((env) => (
                        <MenuItem key={env} value={env}>{env}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                )}
                <TextField
                  label="Computed domain"
                  value={scopeMode === 'cluster' ? 'cluster' : selectedApp ? buildDomain(scopeMode, selectedApp, selectedEnv) : ''}
                  fullWidth
                  helperText="Preview of the exact domain string that will be written to ACL bindings."
                  InputProps={{ readOnly: true }}
                />
                <Stack direction="row" justifyContent="flex-end">
                  <Button
                    variant="contained"
                    startIcon={<AdminPanelSettingsIcon />}
                    disabled={submitting || (scopeMode !== 'cluster' && !selectedApp) || (scopeMode === 'app-env' && !selectedEnv)}
                    onClick={() => void handleAddBinding()}
                  >
                    Add binding
                  </Button>
                </Stack>
              </Stack>
            </Paper>
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeAccessDialog} disabled={submitting || accessLoading}>Close</Button>
        </DialogActions>
      </Dialog>

      {loading ? (
        <LoadingState label="Loading users" />
      ) : error ? null : users.length === 0 ? (
        <EmptyState title="No users found" description="Add a user to start managing account access." />
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Nickname</TableCell>
                <TableCell>Account</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Roles</TableCell>
                <TableCell>Bindings</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((user) => (
                <TableRow key={user.account} hover>
                  <TableCell>{user.nickname || '-'}</TableCell>
                  <TableCell>{user.account}</TableCell>
                  <TableCell>{getStatusLabel(user.status)}</TableCell>
                  <TableCell>{user.roles?.length ? user.roles.map(roleLabel).join(', ') : '-'}</TableCell>
                  <TableCell>{user.bindingCount ?? user.accessSummary?.length ?? 0}</TableCell>
                  <TableCell align="right">
                    <Stack direction="row" spacing={1} justifyContent="flex-end">
                      <Button size="small" startIcon={<AdminPanelSettingsIcon />} disabled={submitting} onClick={() => void handleOpenAccess(user)}>
                        Manage access
                      </Button>
                      <Button color="warning" size="small" startIcon={<DeleteOutlineIcon />} disabled={submitting} onClick={() => void handleDisableUser(user.account)}>
                        Disable
                      </Button>
                      <Button size="small" startIcon={<LockResetIcon />} disabled={submitting} onClick={() => setResetTarget(user)}>
                        Reset password
                      </Button>
                    </Stack>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Stack>
  )
}

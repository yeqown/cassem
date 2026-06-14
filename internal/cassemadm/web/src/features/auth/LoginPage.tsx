import { useState, type FormEvent } from 'react'
import { Alert, Box, Button, Paper, Stack, TextField, Typography } from '@mui/material'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../../auth/AuthProvider'
import { ApiError } from '../../lib/api'

export function LoginPage() {
  const { login } = useAuth()
  const location = useLocation()
  const state = location.state as { from?: unknown } | null
  const redirectTo = typeof state?.from === 'string' && state.from.trim() ? state.from : '/dashboard'
  const [account, setAccount] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')

  async function submit(event: FormEvent) {
    event.preventDefault()
    setError('')
    try {
      await login(account, password, redirectTo)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'login failed')
    }
  }

  return (
    <Box sx={{ minHeight: '100vh', display: 'grid', placeItems: 'center', bgcolor: 'background.default', px: 2 }}>
      <Paper component="form" onSubmit={submit} sx={{ width: '100%', maxWidth: 380, p: { xs: 3, sm: 4 } }} elevation={3}>
        <Stack spacing={2}>
          <Box>
            <Typography variant="overline" color="primary">
              CASSEM
            </Typography>
            <Typography variant="h4" component="h1">
              Cassem Admin
            </Typography>
            <Typography color="text.secondary">Sign in with a cassemadm account.</Typography>
          </Box>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="Account"
            type="email"
            value={account}
            onChange={(event) => setAccount(event.target.value)}
            required
            autoComplete="username"
            helperText="Use your cassemadm account email, for example superadmin@example.com."
          />
          <TextField
            label="Password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            autoComplete="current-password"
            helperText="Enter the password currently assigned to this account."
          />
          <Button type="submit" variant="contained" size="large">
            Login
          </Button>
        </Stack>
      </Paper>
    </Box>
  )
}

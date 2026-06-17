import { useState, type FormEvent } from 'react'
import { Alert, Box, Button, Paper, Stack, TextField, Typography } from '@mui/material'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../../auth/AuthProvider'
import { ApiError } from '../../lib/api'
import { assetUrl } from '../../lib/assets'

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
    <Box
      data-testid="login-background"
      style={{ backgroundImage: `url(${assetUrl('login-topology.svg')})` }}
      sx={{
        minHeight: '100vh',
        display: 'grid',
        placeItems: 'center',
        bgcolor: '#eef7f5',
        backgroundPosition: 'center',
        backgroundRepeat: 'no-repeat',
        backgroundSize: 'cover',
        px: 2,
      }}
    >
      <Paper
        data-testid="login-card"
        data-visual="glass"
        component="form"
        onSubmit={submit}
        sx={{
          width: '100%',
          maxWidth: 380,
          p: { xs: 3, sm: 4 },
          bgcolor: 'rgba(255,255,255,0.84)',
          backdropFilter: 'blur(18px)',
          border: 1,
          borderColor: 'rgba(255,255,255,0.72)',
          boxShadow: '0 24px 70px rgba(71, 98, 105, 0.22)',
        }}
        elevation={0}
      >
        <Stack spacing={2}>
          <Stack data-testid="login-brand" direction="row" spacing={2} alignItems="center">
            <Box component="img" src={assetUrl('logo.svg')} alt="Cassem logo" sx={{ width: 72, height: 72, flexShrink: 0 }} />
            <Box>
              <Typography variant="overline" color="primary">
                CASSEM
              </Typography>
              <Typography variant="h4" component="h1">
                Configuration Center
              </Typography>
              <Typography color="text.secondary">Sign in with a cassemadm account.</Typography>
            </Box>
          </Stack>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="Account"
            type="email"
            value={account}
            onChange={(event) => setAccount(event.target.value)}
            required
            autoComplete="username"
          />
          <TextField
            label="Password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            required
            autoComplete="current-password"
          />
          <Button type="submit" variant="contained" size="large">
            Login
          </Button>
        </Stack>
      </Paper>
    </Box>
  )
}

/* eslint-disable react-refresh/only-export-components */
import AppsIcon from '@mui/icons-material/Apps'
import DnsIcon from '@mui/icons-material/Dns'
import GroupIcon from '@mui/icons-material/Group'
import HubIcon from '@mui/icons-material/Hub'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import { Box, Paper, Stack, Typography } from '@mui/material'
import { Navigate } from 'react-router-dom'
import type { RouteObject } from 'react-router-dom'
import { AuthRoot, RequireAuth } from './auth/RequireAuth'
import { LoginPage } from './features/auth/LoginPage'
import { AppsPage } from './features/apps/AppsPage'
import { AgentsPage } from './features/cluster/AgentsPage'
import { InstancesPage } from './features/cluster/InstancesPage'
import { EnvsPage } from './features/envs/EnvsPage'
import { ElementsPage } from './features/elements/ElementsPage'
import { ElementDetailPage } from './features/elements/ElementDetailPage'
import { PublishWizardPage } from './features/publish/PublishWizardPage'
import { RollbackWizardPage } from './features/publish/RollbackWizardPage'
import { UsersPage } from './features/users/UsersPage'
import { AppShell } from './layout/AppShell'

function SummaryCard({ icon, title, description }: { icon: React.ReactNode; title: string; description: string }) {
  return (
    <Paper variant="outlined" sx={{ p: 2.5 }}>
      <Stack spacing={1.5}>
        <Box sx={{ color: 'primary.main' }}>{icon}</Box>
        <Typography variant="h6">{title}</Typography>
        <Typography color="text.secondary">{description}</Typography>
      </Stack>
    </Paper>
  )
}

function DashboardPage() {
  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h4" component="h1">Dashboard</Typography>
        <Typography color="text.secondary">Use this workspace as the entry point for apps, users, cluster nodes, and versioned configuration operations.</Typography>
      </Box>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <SummaryCard icon={<AppsIcon />} title="Apps and environments" description="Create app namespaces, manage environments, and drill into element inventories." />
        <SummaryCard icon={<GroupIcon />} title="Users and access" description="Add users, reset passwords, disable accounts, and manage ACL bindings from one place." />
      </Stack>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
        <SummaryCard icon={<Inventory2Icon />} title="Elements and rollout" description="Inspect versioned config data, compare history, and use publish or rollback workflows." />
        <SummaryCard icon={<HubIcon />} title="Cluster visibility" description="Inspect agent nodes and client instances grouped under Cluster navigation." />
        <SummaryCard icon={<DnsIcon />} title="Instance tracing" description="Filter cluster instances by app, env, and key to trace live consumers." />
      </Stack>
    </Stack>
  )
}

export const routes: RouteObject[] = [
  {
    element: <AuthRoot />,
    children: [
      { path: '/login', element: <LoginPage /> },
      {
        element: <RequireAuth />,
        children: [
          {
            element: <AppShell />,
            children: [
              { path: '/', element: <Navigate to="/dashboard" replace /> },
              { path: '/dashboard', element: <DashboardPage /> },
              { path: '/apps', element: <AppsPage /> },
              { path: '/apps/:appId/envs', element: <EnvsPage /> },
              { path: '/apps/:appId/envs/:env/elements', element: <ElementsPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key/publish', element: <PublishWizardPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key/rollback', element: <RollbackWizardPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key', element: <ElementDetailPage /> },
              { path: '/users', element: <UsersPage /> },
              { path: '/cluster/agents', element: <AgentsPage /> },
              { path: '/cluster/instances', element: <InstancesPage /> },
            ],
          },
        ],
      },
      { path: '*', element: <Navigate to="/dashboard" replace /> },
    ],
  },
]

/* eslint-disable react-refresh/only-export-components */
import { Navigate, useLocation } from 'react-router-dom'
import type { RouteObject } from 'react-router-dom'
import { AuthRoot, RequireAuth } from './auth/RequireAuth'
import { LoginPage } from './features/auth/LoginPage'
import { AppsPage } from './features/apps/AppsPage'
import { AgentsPage } from './features/cluster/AgentsPage'
import { InstancesPage } from './features/cluster/InstancesPage'
import { DocsPage } from './features/docs/DocsPage'
import { EnvsPage } from './features/envs/EnvsPage'
import { ElementsPage } from './features/elements/ElementsPage'
import { ElementDetailPage } from './features/elements/ElementDetailPage'
import { PublishWizardPage } from './features/publish/PublishWizardPage'
import { RollbackWizardPage } from './features/publish/RollbackWizardPage'
import { SettingsPage } from './features/settings/SettingsPage'
import { UsersPage } from './features/users/UsersPage'
import { AppShell } from './layout/AppShell'

function GuideRedirect() {
  const { search, hash } = useLocation()
  return <Navigate to={{ pathname: '/docs', search, hash }} replace />
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
              { path: '/', element: <GuideRedirect /> },
              { path: '/dashboard', element: <GuideRedirect /> },
              { path: '/docs', element: <DocsPage /> },
              { path: '/apps', element: <AppsPage /> },
              { path: '/apps/:appId/envs', element: <EnvsPage /> },
              { path: '/apps/:appId/envs/:env/elements', element: <ElementsPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key/publish', element: <PublishWizardPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key/rollback', element: <RollbackWizardPage /> },
              { path: '/apps/:appId/envs/:env/elements/:key', element: <ElementDetailPage /> },
              { path: '/users', element: <UsersPage /> },
              { path: '/settings', element: <SettingsPage /> },
              { path: '/cluster/agents', element: <Navigate to="/cluster/topology" replace /> },
              { path: '/cluster/topology', element: <AgentsPage /> },
              { path: '/cluster/instances', element: <InstancesPage /> },
            ],
          },
        ],
      },
      { path: '*', element: <GuideRedirect /> },
    ],
  },
]

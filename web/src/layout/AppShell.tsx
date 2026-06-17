import { useMemo, useState, type MouseEvent, type ReactElement } from 'react'
import AccountCircleIcon from '@mui/icons-material/AccountCircle'
import AppsIcon from '@mui/icons-material/Apps'
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft'
import ChevronRightIcon from '@mui/icons-material/ChevronRight'
import DashboardIcon from '@mui/icons-material/Dashboard'
import DnsIcon from '@mui/icons-material/Dns'
import ExpandLessIcon from '@mui/icons-material/ExpandLess'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import FolderOpenIcon from '@mui/icons-material/FolderOpen'
import GroupIcon from '@mui/icons-material/Group'
import HubIcon from '@mui/icons-material/Hub'
import LogoutIcon from '@mui/icons-material/Logout'
import MenuIcon from '@mui/icons-material/Menu'
import SettingsIcon from '@mui/icons-material/Settings'
import {
  AppBar,
  Box,
  Collapse,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Stack,
  Toolbar,
  Tooltip,
  Typography,
  useMediaQuery,
  useTheme,
} from '@mui/material'
import { Link as RouterLink, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'
import { assetUrl } from '../lib/assets'

const expandedDrawerWidth = 248
const collapsedDrawerWidth = 88
const nav = [
  { label: 'Dashboard', path: '/dashboard', icon: <DashboardIcon /> },
  { label: 'Apps', path: '/apps', icon: <AppsIcon /> },
  { label: 'Users', path: '/users', icon: <GroupIcon /> },
  {
    label: 'Cluster',
    icon: <FolderOpenIcon />,
    children: [
      { label: 'Topology', path: '/cluster/topology', icon: <HubIcon /> },
      { label: 'Instances', path: '/cluster/instances', icon: <DnsIcon /> },
    ],
  },
  { label: 'Settings', path: '/settings', icon: <SettingsIcon /> },
] as const

function getStatusLabel(status?: number) {
  return status === 1 ? 'disabled' : 'normal'
}

export function AppShell() {
  const location = useLocation()
  const { user, logout } = useAuth()
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const [mobileOpen, setMobileOpen] = useState(false)
  const [desktopCollapsed, setDesktopCollapsed] = useState(false)
  const [clusterExpanded, setClusterExpanded] = useState(true)
  const [accountAnchor, setAccountAnchor] = useState<HTMLElement | null>(null)
  const drawerWidth = isDesktop ? (desktopCollapsed ? collapsedDrawerWidth : expandedDrawerWidth) : expandedDrawerWidth
  const accountLabel = user?.nickname || user?.account || 'signed in'
  const rolesLabel = user?.roles?.length ? user.roles.join(', ') : '-'

  const selectedPath = useMemo(
    () => (itemPath: string) => location.pathname === itemPath || location.pathname.startsWith(`${itemPath}/`),
    [location.pathname],
  )

  const clusterSelected = selectedPath('/cluster/topology') || selectedPath('/cluster/instances')

  function handleAccountMenuOpen(event: MouseEvent<HTMLElement>) {
    setAccountAnchor(event.currentTarget)
  }

  function handleAccountMenuClose() {
    setAccountAnchor(null)
  }

  function handleLogout() {
    handleAccountMenuClose()
    logout()
  }

  function renderNavButton(label: string, path: string, icon: ReactElement, collapsed: boolean, nested = false) {
    return (
      <Tooltip key={path} title={label} placement="right" disableHoverListener={!collapsed}>
        <ListItemButton
          component={RouterLink}
          to={path}
          selected={selectedPath(path)}
          onClick={() => setMobileOpen(false)}
          aria-label={label}
          sx={{ minHeight: 44, px: collapsed ? 2.25 : nested ? 4 : 3, justifyContent: collapsed ? 'center' : 'flex-start' }}
        >
          <ListItemIcon sx={{ minWidth: collapsed ? 0 : 36, mr: collapsed ? 0 : 1.5, justifyContent: 'center' }}>{icon}</ListItemIcon>
          {!collapsed && <ListItemText primary={label} />}
        </ListItemButton>
      </Tooltip>
    )
  }

  function renderDrawerContent(collapsed: boolean) {
    return (
      <Box>
        <Toolbar
          data-testid="sidebar-brand"
          style={{ backgroundColor: theme.palette.primary.main, color: theme.palette.primary.contrastText }}
          sx={{ px: collapsed ? 1.5 : 2, justifyContent: collapsed ? 'center' : 'flex-start' }}
        >
          <Stack direction="row" spacing={collapsed ? 0 : 1.5} alignItems="center" sx={{ overflow: 'hidden' }}>
            <Box component="img" data-testid="sidebar-logo" src={assetUrl('logo.svg')} alt="Cassem logo" sx={{ width: 34, height: 34, flexShrink: 0 }} />
            {!collapsed && <Typography variant="h6" sx={{ fontWeight: 700, letterSpacing: 1 }}>CASSEM</Typography>}
          </Stack>
        </Toolbar>
        <Divider />
        <List>
          {nav.map((item) =>
            'path' in item ? (
              renderNavButton(item.label, item.path, item.icon, collapsed)
            ) : (
              <Box key={item.label}>
                <Tooltip title={item.label} placement="right" disableHoverListener={!collapsed}>
                  <ListItemButton
                    selected={clusterSelected}
                    aria-label={item.label}
                    onClick={() => {
                      if (collapsed) {
                        setMobileOpen(false)
                        return
                      }
                      setClusterExpanded((value) => !value)
                    }}
                    sx={{ minHeight: 44, px: collapsed ? 2.25 : 3, justifyContent: collapsed ? 'center' : 'flex-start' }}
                  >
                    <ListItemIcon sx={{ minWidth: collapsed ? 0 : 36, mr: collapsed ? 0 : 1.5, justifyContent: 'center' }}>{item.icon}</ListItemIcon>
                    {!collapsed && <ListItemText primary={item.label} />}
                    {!collapsed && (clusterExpanded ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />)}
                  </ListItemButton>
                </Tooltip>
                <Collapse in={!collapsed && clusterExpanded} timeout="auto" unmountOnExit>
                  <List disablePadding>
                    {item.children.map((child) => renderNavButton(child.label, child.path, child.icon, false, true))}
                  </List>
                </Collapse>
                {collapsed && item.children.map((child) => renderNavButton(child.label, child.path, child.icon, true))}
              </Box>
            ),
          )}
        </List>
      </Box>
    )
  }

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: 'background.default' }}>
      <AppBar position="fixed" sx={{ zIndex: (appTheme) => appTheme.zIndex.drawer + 1, width: isDesktop ? `calc(100% - ${drawerWidth}px)` : '100%', ml: isDesktop ? `${drawerWidth}px` : 0 }}>
        <Toolbar>
          {isDesktop ? (
            <IconButton color="inherit" edge="start" onClick={() => setDesktopCollapsed((value) => !value)} aria-label={desktopCollapsed ? 'expand navigation' : 'collapse navigation'} sx={{ mr: 1 }}>
              {desktopCollapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
            </IconButton>
          ) : (
            <IconButton color="inherit" edge="start" onClick={() => setMobileOpen(true)} sx={{ mr: 2 }} aria-label="open navigation">
              <MenuIcon />
            </IconButton>
          )}
          <Box sx={{ flexGrow: 1 }} />
          <IconButton color="inherit" aria-label="account menu" onClick={handleAccountMenuOpen}>
            <AccountCircleIcon />
          </IconButton>
          <Typography variant="body2" sx={{ ml: 1 }}>{accountLabel}</Typography>
          <Menu anchorEl={accountAnchor} open={Boolean(accountAnchor)} onClose={handleAccountMenuClose} keepMounted>
            <Box sx={{ px: 2, py: 1.5, minWidth: 240 }}>
              <Typography variant="subtitle1">{accountLabel}</Typography>
              <Typography variant="body2" color="text.secondary">{user?.account || '-'}</Typography>
              <Typography variant="body2" color="text.secondary">Roles: {rolesLabel}</Typography>
              <Typography variant="body2" color="text.secondary">Status: {getStatusLabel(user?.status)}</Typography>
            </Box>
            <Divider />
            <MenuItem onClick={handleLogout}>
              <ListItemIcon><LogoutIcon fontSize="small" /></ListItemIcon>
              Logout
            </MenuItem>
          </Menu>
        </Toolbar>
      </AppBar>
      <Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }} aria-label="navigation">
        <Drawer variant="temporary" open={!isDesktop && mobileOpen} onClose={() => setMobileOpen(false)} ModalProps={{ keepMounted: true }} sx={{ display: { xs: 'block', md: 'none' }, '& .MuiDrawer-paper': { width: expandedDrawerWidth, boxSizing: 'border-box' } }}>
          {renderDrawerContent(false)}
        </Drawer>
        <Drawer variant="permanent" sx={{ display: { xs: 'none', md: 'block' }, '& .MuiDrawer-paper': { width: drawerWidth, boxSizing: 'border-box', overflowX: 'hidden' } }} open>
          {renderDrawerContent(desktopCollapsed)}
        </Drawer>
      </Box>
      <Box component="main" sx={{ flexGrow: 1, p: 6, width: { md: `calc(100% - ${drawerWidth}px)` } }}>
        <Toolbar />
        <Outlet />
      </Box>
    </Box>
  )
}

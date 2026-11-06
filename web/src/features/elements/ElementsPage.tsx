import { useCallback, useEffect, useLayoutEffect, useRef, useState, type FormEvent } from 'react'
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import SearchIcon from '@mui/icons-material/Search'
import Inventory2Icon from '@mui/icons-material/Inventory2'
import OpenInNewIcon from '@mui/icons-material/OpenInNew'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  MenuItem,
  Paper,
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
import { Link as RouterLink, useParams } from 'react-router-dom'
import { AppBreadcrumbs } from '../../components/AppBreadcrumbs'
import { DangerConfirmDialog } from '../../components/DangerConfirmDialog'
import { EmptyState, ErrorState, LoadingState } from '../../components/StateView'
import { useErrorState } from '../../components/useErrorState'
import { contentTypes, formatVersionLabel, type Element, type ElementsResponse } from '../../domain/types'
import { useI18n } from '../../i18n'
import { ApiError, apiRequest, buildQuery, jsonBody } from '../../lib/api'
import { readSettings } from '../../lib/settings'
import { ContentEditor } from './ContentEditor'
import { validateContent, type ContentValidationResult } from './contentValidation'

function getErrorMessage(error: unknown, fallback: string) {
  return error instanceof ApiError ? error.message : fallback
}

function getContentTypeLabel(contentType?: number | string) {
  const value = Number(contentType)
  return contentTypes.find((item) => item.value === value)?.label || String(contentType || '-')
}

const pageSizeOptions = [15, 30, 50, 100]
const defaultPageSize = 15

async function requestElements(appId: string, env: string, query: string, limit: number, seek = '') {
  return apiRequest<ElementsResponse>(
    `/api/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements${buildQuery({ limit, query: query || undefined, seek: seek || undefined })}`,
  )
}

export function ElementsPage() {
  const { t, formatNumber } = useI18n()
  const { appId = '', env = '' } = useParams()
  const [elements, setElements] = useState<Element[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useErrorState()
  const [queryInput, setQueryInput] = useState('')
  const [query, setQuery] = useState('')
  const [pageSize, setPageSize] = useState(defaultPageSize)
  const [hasMore, setHasMore] = useState(false)
  const [nextSeek, setNextSeek] = useState('')
  const [loadingMore, setLoadingMore] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState('')
  const [settings] = useState(readSettings)
  const [createKey, setCreateKey] = useState('')
  const [createRaw, setCreateRaw] = useState('')
  const [createContentType, setCreateContentType] = useState<number>(contentTypes[0].value)
  const [createValidation, setCreateValidation] = useState<ContentValidationResult>({ valid: true })
  const requestSeq = useRef(0)
  const mountedRef = useRef(false)
  const lastLoadKeyRef = useRef('')
  const appIdRef = useRef(appId)
  const envRef = useRef(env)
  const queryRef = useRef(query)
  const pageSizeRef = useRef(pageSize)

  useLayoutEffect(() => {
    appIdRef.current = appId
    envRef.current = env
    queryRef.current = query
    pageSizeRef.current = pageSize
  }, [appId, env, pageSize, query])

  const canApplyMutationResult = useCallback(
    (startedAppId: string, startedEnv: string) => mountedRef.current && appIdRef.current === startedAppId && envRef.current === startedEnv,
    [],
  )

  const loadElements = useCallback(
    async (targetQuery = queryRef.current, limit = pageSizeRef.current, seek = '', append = false) => {
      const requestId = ++requestSeq.current

      if (!appId || !env) {
        if (mountedRef.current && requestId === requestSeq.current) {
          setElements([])
          setHasMore(false)
          setNextSeek('')
          setError(t('elements.error.missingLocation'))
          setLoading(false)
          setLoadingMore(false)
        }
        return
      }

      try {
        const data = await requestElements(appId, env, targetQuery, limit, seek)
        if (!mountedRef.current || requestId !== requestSeq.current) return
        setElements((items) => (append ? [...items, ...(data.elements || [])] : data.elements || []))
        setHasMore(Boolean(data.hasMore))
        setNextSeek(data.nextSeek || '')
        setError('')
      } catch (err) {
        if (!mountedRef.current || requestId !== requestSeq.current) return
        if (!append) setElements([])
        setHasMore(false)
        setNextSeek('')
        setError(getErrorMessage(err, t('elements.error.load')))
      } finally {
        if (mountedRef.current && requestId === requestSeq.current) {
          setLoading(false)
          setLoadingMore(false)
        }
      }
    },
    [appId, env, setError, t],
  )

  useEffect(() => {
    mountedRef.current = true

    const loadKey = JSON.stringify({ appId, env })
    if (lastLoadKeyRef.current !== loadKey) {
      lastLoadKeyRef.current = loadKey
      queueMicrotask(() => {
        void loadElements()
      })
    }

    return () => {
      mountedRef.current = false
    }
  }, [appId, env, loadElements])

  useEffect(() => {
    if (!createOpen) return

    const timer = window.setTimeout(() => {
      setCreateValidation(validateContent(createContentType, createRaw, t))
    }, 250)

    return () => window.clearTimeout(timer)
  }, [createOpen, createContentType, createRaw, t])

  async function handleSearchSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const nextQuery = queryInput.trim()
    queryRef.current = nextQuery
    setQuery(nextQuery)
    setLoading(true)
    setHasMore(false)
    setNextSeek('')
    void loadElements(nextQuery, pageSize)
  }

  function handleLoadMore() {
    if (!hasMore || !nextSeek || loadingMore) return
    setLoadingMore(true)
    void loadElements(query, pageSize, nextSeek, true)
  }

  function handleClearSearch() {
    queryRef.current = ''
    setQueryInput('')
    setQuery('')
    setLoading(true)
    setHasMore(false)
    setNextSeek('')
    void loadElements('', pageSize)
  }

  function handlePageSizeChange(value: string) {
    const nextPageSize = Number(value)
    pageSizeRef.current = nextPageSize
    setPageSize(nextPageSize)
    setLoading(true)
    setHasMore(false)
    setNextSeek('')
    void loadElements(query, nextPageSize)
  }

  function closeCreateDialog() {
    if (submitting) return
    setCreateOpen(false)
    setCreateKey('')
    setCreateRaw('')
    setCreateContentType(contentTypes[0].value)
    setCreateValidation({ valid: true })
  }

  function closeDeleteDialog() {
    if (submitting) return
    setDeleteTarget('')
  }

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const startedAppId = appId
    const startedEnv = env
    const nextKey = createKey.trim()
    if (!startedAppId || !startedEnv || !nextKey || !createRaw.trim()) return

    const validation = validateContent(createContentType, createRaw, t)
    setCreateValidation(validation)
    if (!validation.valid) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(nextKey)}`,
        jsonBody({ raw: createRaw, contentType: createContentType }),
      )

      if (!canApplyMutationResult(startedAppId, startedEnv)) return

      setCreateOpen(false)
      setCreateKey('')
      setCreateRaw('')
      setCreateContentType(contentTypes[0].value)
      setCreateValidation({ valid: true })
      setLoading(true)
      setHasMore(false)
      setNextSeek('')
      await loadElements(query, pageSize)
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv)) {
        setError(getErrorMessage(err, t('elements.error.create')))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  async function handleDelete() {
    const startedAppId = appId
    const startedEnv = env
    const targetKey = deleteTarget
    if (!startedAppId || !startedEnv || !targetKey) return

    setSubmitting(true)
    setError('')

    try {
      await apiRequest<void>(
        `/api/apps/${encodeURIComponent(startedAppId)}/envs/${encodeURIComponent(startedEnv)}/elements/${encodeURIComponent(targetKey)}`,
        { method: 'DELETE' },
      )

      if (!canApplyMutationResult(startedAppId, startedEnv)) return

      setDeleteTarget('')
      setLoading(true)
      setHasMore(false)
      setNextSeek('')
      await loadElements(query, pageSize)
    } catch (err) {
      if (canApplyMutationResult(startedAppId, startedEnv)) {
        setError(getErrorMessage(err, t('elements.error.delete')))
      }
    } finally {
      if (mountedRef.current) setSubmitting(false)
    }
  }

  return (
    <Stack spacing={3}>
      <AppBreadcrumbs items={[
        { label: t('elements.apps'), to: '/apps' },
        { label: appId || t('elements.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs` },
        { label: env || t('elements.unknown'), to: `/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements` },
        { label: t('elements.title') },
      ]} />

      <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" spacing={2} alignItems={{ md: 'center' }}>
        <Box>
          <Stack direction="row" spacing={1} alignItems="center">
            <Inventory2Icon color="primary" />
            <Typography variant="h4" component="h1">
              {t('elements.title')}
            </Typography>
          </Stack>
          <Typography color="text.secondary">{t('elements.description')}</Typography>
        </Box>
        <Button variant="contained" startIcon={<AddCircleOutlineIcon />} onClick={() => setCreateOpen(true)} disabled={!appId || !env || submitting}>
          {t('elements.add')}
        </Button>
      </Stack>

      {error.message && <ErrorState message={error.message} eventKey={error.eventKey} />}

      <DangerConfirmDialog
        open={Boolean(deleteTarget)}
        title={t('elements.deleteTitle')}
        description={t('elements.deleteDescription', { key: deleteTarget, app: appId, env })}
        confirmLabel={t('elements.delete')}
        loading={submitting}
        onClose={closeDeleteDialog}
        onConfirm={() => void handleDelete()}
      />

      <Stack component="form" onSubmit={(event) => void handleSearchSubmit(event)} direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }}>
        <TextField
          label={t('elements.searchLabel')}
          value={queryInput}
          onChange={(event) => setQueryInput(event.target.value)}
          size="small"
          fullWidth
          disabled={loading || loadingMore || submitting || !appId || !env}
        />
        <Button type="submit" variant="contained" startIcon={<SearchIcon />} disabled={loading || loadingMore || submitting || !appId || !env} sx={{ minWidth: 128, height: 40, px: 3 }}>
          {t('elements.search')}
        </Button>
        <Button variant="text" onClick={handleClearSearch} disabled={loading || loadingMore || submitting || (!query && !queryInput)}>
          {t('elements.clear')}
        </Button>
        <TextField
          select
          label={t('elements.rowsPerPage')}
          value={String(pageSize)}
          onChange={(event) => handlePageSizeChange(event.target.value)}
          size="small"
          sx={{ minWidth: 140 }}
          disabled={loading || loadingMore || submitting || !appId || !env}
        >
          {pageSizeOptions.map((option) => (
            <MenuItem key={option} value={option}>{formatNumber(option)}</MenuItem>
          ))}
        </TextField>
      </Stack>

      <Dialog open={createOpen} onClose={closeCreateDialog} fullWidth maxWidth="md">
        <Box component="form" onSubmit={(event) => void handleCreate(event)}>
          <DialogTitle>{t('elements.add')}</DialogTitle>
          <DialogContent>
            <Stack spacing={2} sx={{ mt: 1 }}>
              <TextField label={t('elements.key')} value={createKey} onChange={(event) => setCreateKey(event.target.value)} required fullWidth disabled={submitting} helperText={t('elements.keyHelp')} />
              <TextField
                select
                label={t('elements.contentType')}
                value={String(createContentType)}
                onChange={(event) => setCreateContentType(Number(event.target.value))}
                fullWidth
                disabled={submitting}
              >
                {contentTypes.map((contentType) => (
                  <MenuItem key={contentType.value} value={String(contentType.value)}>
                    {contentType.label}
                  </MenuItem>
                ))}
              </TextField>
              <Stack spacing={0.75}>
                <Typography variant="caption" color="text.secondary">
                  {t('elements.rawRequired')}
                </Typography>
                <ContentEditor value={createRaw} contentType={createContentType} ariaLabel={t('elements.raw')} codeTheme={settings.codeTheme} disabled={submitting} lineWrapping={settings.editorLineWrapping} minRows={10} showContentType={false} validation={createValidation} onChange={setCreateRaw} />
                <Typography variant="caption" color="text.secondary">
                  {t('elements.rawHelp')}
                </Typography>
              </Stack>
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button onClick={closeCreateDialog} disabled={submitting}>{t('elements.cancel')}</Button>
            <Button type="submit" variant="contained" disabled={submitting || !createKey.trim() || !createRaw.trim() || !createValidation.valid}>{t('elements.create')}</Button>
          </DialogActions>
        </Box>
      </Dialog>

      {loading ? (
        <LoadingState label={t('elements.loading')} />
      ) : elements.length === 0 ? (
        error.message ? null : <EmptyState title={t('elements.noMatchingRecords')} description={query ? t('elements.noSearchResults') : t('elements.emptyDescription')} />
      ) : (
        <Paper>
          <TableContainer data-testid="elements-table-scroll" sx={{ maxHeight: 560, overflowY: 'auto' }}>
            <Table stickyHeader>
            <TableHead>
              <TableRow>
                <TableCell>{t('elements.key')}</TableCell>
                <TableCell>{t('elements.latest')}</TableCell>
                <TableCell>{t('elements.using')}</TableCell>
                <TableCell>{t('elements.draft')}</TableCell>
                <TableCell>{t('elements.type')}</TableCell>
                <TableCell align="right">{t('elements.actions')}</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {elements.map((element) => {
                const metadata = element.metadata || { key: '' }
                return (
                  <TableRow key={metadata.key} hover>
                    <TableCell>{metadata.key || '-'}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.latestVersion)}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.usingVersion)}</TableCell>
                    <TableCell>{formatVersionLabel(metadata.unpublishedVersion)}</TableCell>
                    <TableCell>{getContentTypeLabel(metadata.contentType)}</TableCell>
                    <TableCell align="right">
                      <Stack direction="row" spacing={1} justifyContent="flex-end">
                        <Button component={RouterLink} to={`/apps/${encodeURIComponent(appId)}/envs/${encodeURIComponent(env)}/elements/${encodeURIComponent(metadata.key)}`} size="small" startIcon={<OpenInNewIcon />}>
                          {t('elements.detail')}
                        </Button>
                        <Button color="error" size="small" startIcon={<DeleteOutlineIcon />} onClick={() => setDeleteTarget(metadata.key)} disabled={submitting}>
                          {t('elements.delete')}
                        </Button>
                      </Stack>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
            </Table>
          </TableContainer>
          {hasMore && (
            <Stack direction="row" justifyContent="center" sx={{ p: 2 }}>
              <Button onClick={handleLoadMore} disabled={loadingMore || submitting}>
                {loadingMore ? t('elements.loadingMore') : t('elements.loadMore')}
              </Button>
            </Stack>
          )}
        </Paper>
      )}
    </Stack>
  )
}

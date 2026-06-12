const readStoredUser = () => {
  try {
    return JSON.parse(localStorage.getItem('cassem.user') || 'null')
  } catch {
    localStorage.removeItem('cassem.user')
    return null
  }
}

const isAuthError = (payload) => payload?.errcode === 16 || /unauthenticated|session expired|invalid session/i.test(payload?.errmsg || '')

document.addEventListener('alpine:init', () => {
  Alpine.data('adminApp', () => ({
    session: localStorage.getItem('cassem.session') || '',
    user: readStoredUser(),
    theme: localStorage.getItem('cassem.theme') || 'dark',
    activeTab: localStorage.getItem('cassem.activeTab') || 'config',
    notice: null,
    noticeTimer: null,
    loginForm: { account: '', password: '' },
    apps: [],
    envs: [],
    elements: [],
    versions: [],
    operations: [],
    selectedApp: '',
    selectedEnv: '',
    selectedElement: null,
    appForm: { id: '', description: '' },
    envForm: { name: '' },
    elementForm: { key: '', contentType: 1, raw: '' },
    elementQuery: { key: '' },
    versionForm: { base: '', compare: '', publish: '', publishMode: 2, agentIds: '', instanceIds: '', rollback: '' },
    diffResult: null,
    userForm: { account: '', nickname: '', password: '', role: 'developer', domains: 'cluster' },
    cluster: { agents: [], instances: [], detailId: '', detail: null, filter: { app: '', env: '', key: '' } },
    contentTypes: [
      { value: 1, name: 'JSON', label: 'JSON' },
      { value: 2, name: 'TOML', label: 'TOML' },
      { value: 3, name: 'INI', label: 'INI' },
      { value: 4, name: 'PLAINTEXT', label: 'PLAINTEXT' }
    ],

    init() {
      document.documentElement.dataset.theme = this.theme
      if (this.session) {
        this.loadApps()
        if (this.activeTab === 'cluster') {
          this.loadAgents()
          this.loadInstances()
        }
      }
    },

    setTab(tab) {
      this.activeTab = tab
      localStorage.setItem('cassem.activeTab', tab)
      if (tab === 'config' && this.apps.length === 0) this.loadApps()
      if (tab === 'cluster') {
        this.loadAgents()
        this.loadInstances()
      }
    },

    toggleTheme() {
      this.theme = this.theme === 'dark' ? 'light' : 'dark'
      localStorage.setItem('cassem.theme', this.theme)
      document.documentElement.dataset.theme = this.theme
    },

    show(kind, text) {
      this.notice = { kind, text }
      window.clearTimeout(this.noticeTimer)
      this.noticeTimer = window.setTimeout(() => { this.notice = null }, 4500)
    },

    requireValue(value, label) {
      if (!String(value || '').trim()) throw new Error(`${label} is required`)
    },

    enc(value) {
      return encodeURIComponent(value)
    },

    csv(value) {
      return String(value || '').split(',').map(item => item.trim()).filter(Boolean)
    },

    url(path, params = {}) {
      const search = new URLSearchParams()
      Object.entries(params).forEach(([key, value]) => {
        if (value === undefined || value === null || value === '') return
        if (Array.isArray(value)) value.forEach(item => search.append(key, item))
        else search.append(key, value)
      })
      const query = search.toString()
      return query ? `${path}?${query}` : path
    },

    async api(path, options = {}) {
      const init = { method: options.method || 'GET', headers: { Accept: 'application/json', ...(options.headers || {}) } }
      if (this.session) init.headers['X-CASSEM-SESSION'] = this.session
      if (options.body !== undefined) {
        init.headers['Content-Type'] = 'application/json'
        init.body = JSON.stringify(options.body)
      }

      const response = await fetch(path, init)
      const payload = await response.json().catch(() => ({ errcode: -1, errmsg: `HTTP ${response.status}` }))
      if (response.status === 401 || isAuthError(payload)) {
        this.clearSession()
        this.resetWorkspace()
        throw new Error('session expired')
      }

      if (!response.ok || payload.errcode !== 0) throw new Error(payload.errmsg || `HTTP ${response.status}`)
      return payload.data
    },

    async run(action, success) {
      try {
        const result = await action()
        if (success) this.show('success', success)
        return result
      } catch (err) {
        this.show('error', err.message || 'operation failed')
        return null
      }
    },

    async login() {
      await this.run(async () => {
        this.requireValue(this.loginForm.account, 'account')
        this.requireValue(this.loginForm.password, 'password')
        const data = await this.api('/api/account/login', { method: 'POST', body: this.loginForm })
        this.resetWorkspace()
        this.session = data.session
        this.user = data.user
        localStorage.setItem('cassem.session', this.session)
        localStorage.setItem('cassem.user', JSON.stringify(this.user))
        await this.loadApps()
      }, 'logged in')
    },

    resetWorkspace() {
      this.apps = []
      this.envs = []
      this.elements = []
      this.versions = []
      this.operations = []
      this.selectedApp = ''
      this.selectedEnv = ''
      this.appForm = { id: '', description: '' }
      this.envForm = { name: '' }
      this.elementQuery = { key: '' }
      this.clearElementForm()
      this.userForm = { account: '', nickname: '', password: '', role: 'developer', domains: 'cluster' }
      this.cluster = { agents: [], instances: [], detailId: '', detail: null, filter: { app: '', env: '', key: '' } }
    },

    clearSession() {
      this.session = ''
      this.user = null
      localStorage.removeItem('cassem.session')
      localStorage.removeItem('cassem.user')
    },

    logout() {
      this.clearSession()
      this.resetWorkspace()
    },

    decodeRaw(raw) {
      if (!raw) return ''
      try {
        const bytes = Uint8Array.from(atob(raw), char => char.charCodeAt(0))
        return new TextDecoder().decode(bytes)
      } catch (err) {
        return raw
      }
    },

    contentTypeValue(value) {
      const numberValue = Number(value)
      if (Number.isFinite(numberValue) && numberValue > 0) return numberValue
      return this.contentTypes.find(item => item.name === value)?.value || 1
    },

    contentTypeLabel(value) {
      const numberValue = Number(value)
      if (Number.isFinite(numberValue)) return this.contentTypes.find(item => item.value === numberValue)?.label || 'UNKNOWN'
      return this.contentTypes.find(item => item.name === value)?.label || 'UNKNOWN'
    },

    async loadApps() {
      await this.run(async () => {
        const data = await this.api('/api/apps?limit=100')
        this.apps = data?.apps || []
        if (!this.selectedApp && this.apps.length > 0) await this.selectApp(this.apps[0])
      })
    },

    async createApp() {
      await this.run(async () => {
        this.requireValue(this.appForm.id, 'app id')
        this.requireValue(this.appForm.description, 'description')
        await this.api(`/api/apps/${this.enc(this.appForm.id)}`, { method: 'POST', body: { name: this.appForm.id, description: this.appForm.description } })
        this.appForm = { id: '', description: '' }
        await this.loadApps()
      }, 'app created')
    },

    async deleteApp() {
      if (!this.selectedApp || !confirm(`Delete app ${this.selectedApp}?`)) return
      await this.run(async () => {
        await this.api(`/api/apps/${this.enc(this.selectedApp)}`, { method: 'DELETE' })
        this.selectedApp = ''
        this.selectedEnv = ''
        this.envs = []
        this.elements = []
        this.clearElementForm()
        await this.loadApps()
      }, 'app deleted')
    },

    async selectApp(app) {
      this.selectedApp = app.id
      this.selectedEnv = ''
      this.envs = []
      this.elements = []
      this.clearElementForm()
      await this.loadEnvs()
    },

    async loadEnvs() {
      if (!this.selectedApp) return
      await this.run(async () => {
        const data = await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs?limit=100`)
        this.envs = data?.envs || []
        if (!this.selectedEnv && this.envs.length > 0) await this.selectEnv(this.envs[0])
      })
    },

    async createEnv() {
      await this.run(async () => {
        this.requireValue(this.selectedApp, 'app')
        this.requireValue(this.envForm.name, 'env')
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.envForm.name)}`, { method: 'POST' })
        this.envForm = { name: '' }
        await this.loadEnvs()
      }, 'env created')
    },

    async deleteEnv() {
      if (!this.selectedApp || !this.selectedEnv || !confirm(`Delete env ${this.selectedEnv}?`)) return
      await this.run(async () => {
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}`, { method: 'DELETE' })
        this.selectedEnv = ''
        this.elements = []
        this.clearElementForm()
        await this.loadEnvs()
      }, 'env deleted')
    },

    async selectEnv(env) {
      this.selectedEnv = env
      this.clearElementForm()
      await this.loadElements()
    },

    async loadElements() {
      if (!this.selectedApp || !this.selectedEnv) return
      await this.run(async () => {
        const params = { limit: 100 }
        if (this.elementQuery.key) params.key = this.elementQuery.key
        const data = await this.api(this.url(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements`, params))
        this.elements = data?.elements || []
      })
    },

    clearElementForm() {
      this.selectedElement = null
      this.elementForm = { key: '', contentType: 1, raw: '' }
      this.versions = []
      this.operations = []
      this.diffResult = null
      this.versionForm = { base: '', compare: '', publish: '', publishMode: 2, agentIds: '', instanceIds: '', rollback: '' }
    },

    async selectElement(element) {
      this.selectedElement = element
      this.elementForm = { key: element.metadata.key, contentType: this.contentTypeValue(element.metadata.contentType), raw: this.decodeRaw(element.raw) }
      this.versionForm.publish = element.metadata.unpublishedVersion || element.version || ''
      this.versionForm.rollback = element.metadata.usingVersion || ''
      await this.loadVersions()
      await this.loadOperations()
    },

    async createElement() {
      await this.run(async () => {
        this.requireValue(this.selectedApp, 'app')
        this.requireValue(this.selectedEnv, 'env')
        this.requireValue(this.elementForm.key, 'element key')
        this.requireValue(this.elementForm.raw, 'raw')
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}`, { method: 'POST', body: { raw: this.elementForm.raw, contentType: Number(this.elementForm.contentType) } })
        await this.loadElements()
      }, 'element created')
    },

    async updateElement() {
      if (!this.selectedElement) return
      await this.run(async () => {
        const key = this.selectedElement.metadata.key
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(key)}`, { method: 'PUT', body: { raw: this.elementForm.raw } })
        await this.loadElements()
        const updated = this.elements.find(element => element.metadata.key === key)
        if (updated) await this.selectElement(updated)
        else this.clearElementForm()
      }, 'element updated')
    },

    async deleteElement() {
      if (!this.selectedElement || !confirm(`Delete element ${this.elementForm.key}?`)) return
      await this.run(async () => {
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}`, { method: 'DELETE' })
        this.clearElementForm()
        await this.loadElements()
      }, 'element deleted')
    },

    async loadVersions() {
      if (!this.selectedElement) return
      await this.run(async () => {
        const data = await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}/versions?limit=100`)
        this.versions = data?.elements || []
      })
    },

    async loadOperations() {
      if (!this.selectedElement) return
      await this.run(async () => {
        const data = await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}/operations?limit=100`)
        this.operations = data?.operations || []
      })
    },

    selectVersion(version) {
      this.versionForm.base = this.versionForm.base || version.version
      this.versionForm.compare = version.version
      this.versionForm.publish = version.version
      this.versionForm.rollback = version.version
    },

    async diffVersions() {
      await this.run(async () => {
        this.requireValue(this.versionForm.base, 'base version')
        this.requireValue(this.versionForm.compare, 'compare version')
        this.diffResult = await this.api(this.url(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}/diff`, { base: this.versionForm.base, compare: this.versionForm.compare }))
      }, 'diff loaded')
    },

    async publishVersion() {
      await this.run(async () => {
        this.requireValue(this.versionForm.publish, 'publish version')
        const body = { version: Number(this.versionForm.publish), publishMode: Number(this.versionForm.publishMode), agentId: this.csv(this.versionForm.agentIds), instanceId: this.csv(this.versionForm.instanceIds) }
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}/publish`, { method: 'POST', body })
        await this.loadElements()
        await this.loadVersions()
      }, 'version published')
    },

    async rollbackVersion() {
      if (!confirm(`Rollback ${this.elementForm.key} to v${this.versionForm.rollback}?`)) return
      await this.run(async () => {
        this.requireValue(this.versionForm.rollback, 'rollback version')
        await this.api(`/api/apps/${this.enc(this.selectedApp)}/envs/${this.enc(this.selectedEnv)}/elements/${this.enc(this.elementForm.key)}/rollback`, { method: 'POST', body: { version: Number(this.versionForm.rollback) } })
        await this.loadElements()
        await this.loadVersions()
      }, 'rollback created')
    },

    async addUser() {
      await this.run(async () => {
        this.requireValue(this.userForm.account, 'account')
        this.requireValue(this.userForm.nickname, 'nickname')
        this.requireValue(this.userForm.password, 'password')
        await this.api('/api/account/add', { method: 'POST', body: { account: this.userForm.account, nickname: this.userForm.nickname, password: this.userForm.password } })
      }, 'user added')
    },

    async disableUser() {
      if (!this.userForm.account || !confirm(`Disable user ${this.userForm.account}?`)) return
      await this.run(async () => {
        await this.api(this.url('/api/account/disable', { account: this.userForm.account }))
      }, 'user disabled')
    },

    async resetUser() {
      await this.run(async () => {
        this.requireValue(this.userForm.account, 'account')
        this.requireValue(this.userForm.password, 'password')
        await this.api('/api/account/reset', { method: 'POST', body: { account: this.userForm.account, password: this.userForm.password } })
      }, 'password reset')
    },

    async assignRole() {
      await this.run(async () => {
        this.requireValue(this.userForm.account, 'account')
        this.requireValue(this.userForm.role, 'role')
        await this.api(this.url('/api/account/acl/assign', { account: this.userForm.account, role: this.userForm.role, domain: this.csv(this.userForm.domains) }))
      }, 'role assigned')
    },

    async revokeRole() {
      await this.run(async () => {
        this.requireValue(this.userForm.account, 'account')
        this.requireValue(this.userForm.role, 'role')
        await this.api(this.url('/api/account/acl/revoke', { account: this.userForm.account, role: this.userForm.role, domain: this.csv(this.userForm.domains) }))
      }, 'role revoked')
    },

    async loadAgents() {
      await this.run(async () => {
        const data = await this.api('/api/cluster/agents?limit=100')
        this.cluster.agents = Array.isArray(data) ? data : (data?.agents || [])
      })
    },

    async loadInstances() {
      await this.run(async () => {
        const data = await this.api('/api/cluster/instances?limit=100')
        this.cluster.instances = data?.instances || []
      })
    },

    async loadInstanceDetail(id) {
      await this.run(async () => {
        this.cluster.detailId = id
        this.cluster.detail = await this.api(`/api/cluster/instances/detail/${this.enc(id)}`)
      }, 'instance loaded')
    },

    async filterInstances() {
      await this.run(async () => {
        this.requireValue(this.cluster.filter.app, 'app')
        this.requireValue(this.cluster.filter.env, 'env')
        this.requireValue(this.cluster.filter.key, 'key')
        const data = await this.api(this.url('/api/cluster/instances/filter', { app: this.cluster.filter.app, env: this.cluster.filter.env, key: this.cluster.filter.key }))
        this.cluster.instances = data?.instances || []
      }, 'instances filtered')
    }
  }))
})

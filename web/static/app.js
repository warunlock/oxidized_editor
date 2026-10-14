/**
 * Oxidized Router DB Manager - Frontend Application
 */

(function () {
  'use strict';

  // Application State
  const state = {
    authenticated: false,
    username: '',
    dbPath: '',
    devices: [],
    models: {},
    groups: {},
    lastModified: '',
    serviceStatus: 'unknown',
    searchTerm: '',
    filterModel: '',
    filterGroup: '',
    activeView: 'table', // 'table' | 'raw'
    pendingDeleteId: null,
    visiblePasswords: new Set(),
  };

  // DOM Elements
  const el = {
    // Auth
    authScreen: document.getElementById('authScreen'),
    loginForm: document.getElementById('loginForm'),
    loginUsername: document.getElementById('loginUsername'),
    loginPassword: document.getElementById('loginPassword'),
    toggleLoginPassword: document.getElementById('toggleLoginPassword'),
    loginError: document.getElementById('loginError'),
    btnLoginSubmit: document.getElementById('btnLoginSubmit'),
    btnLogout: document.getElementById('btnLogout'),

    // Main App
    appContainer: document.getElementById('appContainer'),
    headerDbPath: document.getElementById('headerDbPath'),
    serviceStatusPill: document.getElementById('serviceStatusPill'),
    serviceStatusDot: document.getElementById('serviceStatusDot'),
    serviceStatusText: document.getElementById('serviceStatusText'),
    btnRefreshStatus: document.getElementById('btnRefreshStatus'),
    btnOpenRestartModal: document.getElementById('btnOpenRestartModal'),
    btnToggleView: document.getElementById('btnToggleView'),
    toggleViewText: document.getElementById('toggleViewText'),
    btnOpenChangePassword: document.getElementById('btnOpenChangePassword'),

    // Stats
    statTotalDevices: document.getElementById('statTotalDevices'),
    statTotalModels: document.getElementById('statTotalModels'),
    statModelsDesc: document.getElementById('statModelsDesc'),
    statTotalGroups: document.getElementById('statTotalGroups'),
    statGroupsDesc: document.getElementById('statGroupsDesc'),
    statServiceState: document.getElementById('statServiceState'),
    statServiceIcon: document.getElementById('statServiceIcon'),
    statLastModified: document.getElementById('statLastModified'),

    // Table view
    tableViewSection: document.getElementById('tableViewSection'),
    searchInput: document.getElementById('searchInput'),
    filterModel: document.getElementById('filterModel'),
    filterGroup: document.getElementById('filterGroup'),
    btnOpenAddModal: document.getElementById('btnOpenAddModal'),
    btnEmptyAdd: document.getElementById('btnEmptyAdd'),
    deviceCounter: document.getElementById('deviceCounter'),
    devicesTable: document.getElementById('devicesTable'),
    devicesTableBody: document.getElementById('devicesTableBody'),
    emptyState: document.getElementById('emptyState'),

    // Raw view
    rawViewSection: document.getElementById('rawViewSection'),
    rawContentArea: document.getElementById('rawContentArea'),
    btnReloadRaw: document.getElementById('btnReloadRaw'),
    btnSaveRaw: document.getElementById('btnSaveRaw'),

    // Device Modal
    deviceModal: document.getElementById('deviceModal'),
    deviceModalTitle: document.getElementById('deviceModalTitle'),
    deviceForm: document.getElementById('deviceForm'),
    devEditId: document.getElementById('devEditId'),
    devName: document.getElementById('devName'),
    devIP: document.getElementById('devIP'),
    devModel: document.getElementById('devModel'),
    devUsername: document.getElementById('devUsername'),
    devPassword: document.getElementById('devPassword'),
    devGroup: document.getElementById('devGroup'),
    devPort: document.getElementById('devPort'),
    devPreviewLine: document.getElementById('devPreviewLine'),
    toggleDevPassword: document.getElementById('toggleDevPassword'),
    btnCloseDeviceModal: document.getElementById('btnCloseDeviceModal'),
    btnCancelDeviceModal: document.getElementById('btnCancelDeviceModal'),

    // Delete Modal
    deleteModal: document.getElementById('deleteModal'),
    deleteDeviceName: document.getElementById('deleteDeviceName'),
    deleteDeviceIP: document.getElementById('deleteDeviceIP'),
    btnConfirmDelete: document.getElementById('btnConfirmDelete'),
    btnCancelDelete: document.getElementById('btnCancelDelete'),
    btnCloseDeleteModal: document.getElementById('btnCloseDeleteModal'),

    // Restart Modal
    restartModal: document.getElementById('restartModal'),
    btnExecuteRestart: document.getElementById('btnExecuteRestart'),
    btnCancelRestart: document.getElementById('btnCancelRestart'),
    btnCloseRestartModal: document.getElementById('btnCloseRestartModal'),
    restartTerminalOutput: document.getElementById('restartTerminalOutput'),

    // Change Password Modal
    changePasswordModal: document.getElementById('changePasswordModal'),
    changePasswordForm: document.getElementById('changePasswordForm'),
    currentPassword: document.getElementById('currentPassword'),
    newPassword: document.getElementById('newPassword'),
    confirmPassword: document.getElementById('confirmPassword'),
    btnCancelChangePwd: document.getElementById('btnCancelChangePwd'),
    btnCloseChangePwdModal: document.getElementById('btnCloseChangePwdModal'),

    // Toast
    toastContainer: document.getElementById('toastContainer'),
  };

  // API Client with error handling
  async function apiRequest(endpoint, options = {}) {
    try {
      const response = await fetch(endpoint, {
        headers: {
          'Content-Type': 'application/json',
          ...(options.headers || {}),
        },
        ...options,
      });

      if (response.status === 401 && endpoint !== '/api/login' && endpoint !== '/api/auth/me') {
        showLoginScreen();
        showToast('Сесія завершилась. Будь ласка, увійдіть знову.', 'error');
        throw new Error('Unauthorized');
      }

      const data = await response.json();
      return { ok: response.ok, status: response.status, data };
    } catch (err) {
      if (err.message !== 'Unauthorized') {
        console.error('API Error:', err);
      }
      return { ok: false, error: err.message };
    }
  }

  // Toast Notification System
  function showToast(message, type = 'info', duration = 3500) {
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;

    let iconSvg = '';
    if (type === 'success') {
      iconSvg = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path><polyline points="22 4 12 14.01 9 11.01"></polyline></svg>`;
    } else if (type === 'error') {
      iconSvg = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>`;
    } else {
      iconSvg = `<svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="12" y1="16" x2="12" y2="12"></line><line x1="12" y1="8" x2="12.01" y2="8"></line></svg>`;
    }

    toast.innerHTML = `${iconSvg}<span>${escapeHtml(message)}</span>`;
    el.toastContainer.appendChild(toast);

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateX(100%)';
      toast.style.transition = 'all 0.25s ease';
      setTimeout(() => toast.remove(), 250);
    }, duration);
  }

  function escapeHtml(text) {
    if (!text) return '';
    return String(text)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');
  }

  // App Initialization
  async function init() {
    setupEventListeners();
    await checkAuth();
  }

  async function checkAuth() {
    const res = await apiRequest('/api/auth/me');
    if (res.ok && res.data && res.data.authenticated) {
      state.authenticated = true;
      state.username = res.data.username;
      state.dbPath = res.data.db_path || '/var/lib/oxidized/router.db';
      el.headerDbPath.textContent = state.dbPath;
      showAppScreen();
      await Promise.all([loadDevices(), checkOxidizedStatus()]);
    } else {
      showLoginScreen();
    }
  }

  function showLoginScreen() {
    state.authenticated = false;
    el.authScreen.classList.remove('hidden');
    el.appContainer.classList.add('hidden');
    el.loginPassword.value = '';
    el.loginError.classList.add('hidden');
  }

  function showAppScreen() {
    el.authScreen.classList.add('hidden');
    el.appContainer.classList.remove('hidden');
  }

  // Authentication Handlers
  async function handleLogin(e) {
    e.preventDefault();
    el.loginError.classList.add('hidden');
    el.btnLoginSubmit.disabled = true;
    el.btnLoginSubmit.innerHTML = `<div class="spinner"></div><span>Перевірка...</span>`;

    const username = el.loginUsername.value.trim();
    const password = el.loginPassword.value;

    const res = await apiRequest('/api/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });

    el.btnLoginSubmit.disabled = false;
    el.btnLoginSubmit.innerHTML = `<span>Увійти в систему</span>`;

    if (res.ok && res.data && res.data.success) {
      state.authenticated = true;
      state.username = res.data.username;
      showToast('Авторизація успішна!', 'success');
      await checkAuth();
    } else {
      el.loginError.textContent = (res.data && res.data.error) || 'Невірний логін або пароль';
      el.loginError.classList.remove('hidden');
    }
  }

  async function handleLogout() {
    await apiRequest('/api/logout', { method: 'POST' });
    showLoginScreen();
    showToast('Ви вийшли із системи', 'info');
  }

  // Load Devices Data
  async function loadDevices() {
    const res = await apiRequest('/api/devices');
    if (!res.ok || !res.data) {
      showToast('Помилка завантаження router.db', 'error');
      return;
    }

    state.devices = res.data.devices || [];
    state.models = res.data.models || {};
    state.groups = res.data.groups || {};
    state.lastModified = res.data.last_modified;
    if (res.data.db_path) {
      state.dbPath = res.data.db_path;
      el.headerDbPath.textContent = state.dbPath;
    }

    updateStats();
    populateFilters();
    renderDevicesTable();
  }

  // Update Stats Cards
  function updateStats() {
    el.statTotalDevices.textContent = state.devices.length;

    const modelKeys = Object.keys(state.models);
    el.statTotalModels.textContent = modelKeys.length;
    el.statModelsDesc.textContent = modelKeys.slice(0, 3).join(', ') + (modelKeys.length > 3 ? '...' : '');

    const groupKeys = Object.keys(state.groups);
    el.statTotalGroups.textContent = groupKeys.length;
    el.statGroupsDesc.textContent = groupKeys.slice(0, 3).join(', ') + (groupKeys.length > 3 ? '...' : '');

    if (state.lastModified) {
      const d = new Date(state.lastModified);
      el.statLastModified.textContent = `Змінено: ${d.toLocaleTimeString()}`;
    }
  }

  // Populate Model and Group dropdown filters
  function populateFilters() {
    const currentModel = el.filterModel.value;
    const currentGroup = el.filterGroup.value;

    // Models dropdown
    el.filterModel.innerHTML = '<option value="">Всі моделі</option>';
    Object.keys(state.models).sort().forEach(m => {
      const opt = document.createElement('option');
      opt.value = m;
      opt.textContent = `${m} (${state.models[m]})`;
      if (m === currentModel) opt.selected = true;
      el.filterModel.appendChild(opt);
    });

    // Groups dropdown
    el.filterGroup.innerHTML = '<option value="">Всі групи</option>';
    Object.keys(state.groups).sort().forEach(g => {
      const opt = document.createElement('option');
      opt.value = g;
      opt.textContent = `${g} (${state.groups[g]})`;
      if (g === currentGroup) opt.selected = true;
      el.filterGroup.appendChild(opt);
    });
  }

  // Render Table of Devices
  function renderDevicesTable() {
    const tbody = el.devicesTableBody;
    tbody.innerHTML = '';

    const term = state.searchTerm.toLowerCase();
    const filterM = state.filterModel;
    const filterG = state.filterGroup;

    const filtered = state.devices.filter(dev => {
      if (filterM && dev.model !== filterM) return false;
      if (filterG && dev.group !== filterG) return false;
      if (!term) return true;

      return (
        (dev.name && dev.name.toLowerCase().includes(term)) ||
        (dev.ip && dev.ip.toLowerCase().includes(term)) ||
        (dev.model && dev.model.toLowerCase().includes(term)) ||
        (dev.username && dev.username.toLowerCase().includes(term)) ||
        (dev.group && dev.group.toLowerCase().includes(term)) ||
        (dev.port && dev.port.toLowerCase().includes(term))
      );
    });

    el.deviceCounter.textContent = `Показано: ${filtered.length} з ${state.devices.length}`;

    if (filtered.length === 0) {
      el.devicesTable.classList.add('hidden');
      el.emptyState.classList.remove('hidden');
      return;
    }

    el.devicesTable.classList.remove('hidden');
    el.emptyState.classList.add('hidden');

    filtered.forEach((dev, index) => {
      const tr = document.createElement('tr');
      const isPwdVisible = state.visiblePasswords.has(dev.id);

      // Model badge class helper
      const modelClass = getModelBadgeClass(dev.model);

      tr.innerHTML = `
        <td style="color: var(--text-dim); font-size: 0.8rem;">${index + 1}</td>
        <td>
          <div class="device-name">
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="color: var(--primary);">
              <rect x="2" y="2" width="20" height="8" rx="2"></rect>
              <rect x="2" y="14" width="20" height="8" rx="2"></rect>
            </svg>
            <span>${escapeHtml(dev.name)}</span>
          </div>
        </td>
        <td>
          <div class="ip-cell">
            <span>${escapeHtml(dev.ip)}</span>
            <button class="copy-btn btn-copy-ip" data-ip="${escapeHtml(dev.ip)}" title="Скопіювати IP">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
            </button>
          </div>
        </td>
        <td>
          <span class="badge-model ${modelClass}">${escapeHtml(dev.model || '-')}</span>
        </td>
        <td style="color: var(--text-muted); font-size: 0.85rem;">
          ${escapeHtml(dev.username || '-')}
        </td>
        <td>
          <div class="pwd-cell">
            <span class="pwd-text" id="pwd-text-${dev.id}">
              ${isPwdVisible ? escapeHtml(dev.password || '—') : (dev.password ? '••••••••' : '—')}
            </span>
            ${dev.password ? `
              <button class="copy-btn btn-toggle-row-pwd" data-id="${dev.id}" title="${isPwdVisible ? 'Приховати' : 'Показати'}">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"></path>
                  <circle cx="12" cy="12" r="3"></circle>
                </svg>
              </button>
            ` : ''}
          </div>
        </td>
        <td>
          ${dev.group ? `<span class="badge-group">${escapeHtml(dev.group)}</span>` : '<span style="color:var(--text-dim);">-</span>'}
        </td>
        <td style="font-family: var(--font-mono); font-size: 0.8rem; color: var(--text-muted);">
          ${escapeHtml(dev.port || 'default')}
        </td>
        <td style="text-align: right;">
          <div class="table-actions" style="justify-content: flex-end;">
            <button class="btn btn-secondary btn-icon btn-edit-dev" data-id="${dev.id}" title="Редагувати">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"></path>
                <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"></path>
              </svg>
            </button>
            <button class="btn btn-secondary btn-icon btn-clone-dev" data-id="${dev.id}" title="Дублювати (Клон)">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
              </svg>
            </button>
            <button class="btn btn-danger btn-icon btn-delete-dev" data-id="${dev.id}" title="Видалити">
              <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
            </button>
          </div>
        </td>
      `;

      tbody.appendChild(tr);
    });

    // Attach row events
    tbody.querySelectorAll('.btn-copy-ip').forEach(btn => {
      btn.addEventListener('click', () => {
        navigator.clipboard.writeText(btn.dataset.ip);
        showToast(`IP ${btn.dataset.ip} скопійовано!`, 'info', 2000);
      });
    });

    tbody.querySelectorAll('.btn-toggle-row-pwd').forEach(btn => {
      btn.addEventListener('click', () => {
        const id = btn.dataset.id;
        if (state.visiblePasswords.has(id)) {
          state.visiblePasswords.delete(id);
        } else {
          state.visiblePasswords.add(id);
        }
        renderDevicesTable();
      });
    });

    tbody.querySelectorAll('.btn-edit-dev').forEach(btn => {
      btn.addEventListener('click', () => openEditDeviceModal(btn.dataset.id));
    });

    tbody.querySelectorAll('.btn-clone-dev').forEach(btn => {
      btn.addEventListener('click', () => cloneDevice(btn.dataset.id));
    });

    tbody.querySelectorAll('.btn-delete-dev').forEach(btn => {
      btn.addEventListener('click', () => openDeleteModal(btn.dataset.id));
    });
  }

  function getModelBadgeClass(model) {
    if (!model) return '';
    const m = model.toLowerCase();
    if (m.includes('cisco') || m.includes('ios') || m.includes('nxos')) return 'cisco';
    if (m.includes('junos') || m.includes('juniper')) return 'junos';
    if (m.includes('routeros') || m.includes('mikrotik')) return 'routeros';
    if (m.includes('huawei') || m.includes('vrp')) return 'huawei';
    if (m.includes('fortios') || m.includes('fortinet')) return 'fortios';
    if (m.includes('edgeos') || m.includes('ubnt')) return 'edgeos';
    if (m.includes('vyos')) return 'vyos';
    return '';
  }

  // Device Add / Edit Modal
  function openAddDeviceModal() {
    el.devEditId.value = '';
    el.deviceModalTitle.textContent = 'Додати новий пристрій';
    el.deviceForm.reset();
    updateDevicePreview();
    el.deviceModal.classList.add('open');
    el.devName.focus();
  }

  function openEditDeviceModal(id) {
    const dev = state.devices.find(d => d.id === id);
    if (!dev) return;

    el.devEditId.value = dev.id;
    el.deviceModalTitle.textContent = `Редагування: ${dev.name}`;
    el.devName.value = dev.name || '';
    el.devIP.value = dev.ip || '';
    el.devModel.value = dev.model || '';
    el.devUsername.value = dev.username || '';
    el.devPassword.value = dev.password || '';
    el.devGroup.value = dev.group || '';
    el.devPort.value = dev.port || '';

    updateDevicePreview();
    el.deviceModal.classList.add('open');
  }

  function cloneDevice(id) {
    const dev = state.devices.find(d => d.id === id);
    if (!dev) return;

    el.devEditId.value = '';
    el.deviceModalTitle.textContent = `Клонування пристрою: ${dev.name}`;
    el.devName.value = `${dev.name}-copy`;
    el.devIP.value = dev.ip || '';
    el.devModel.value = dev.model || '';
    el.devUsername.value = dev.username || '';
    el.devPassword.value = dev.password || '';
    el.devGroup.value = dev.group || '';
    el.devPort.value = dev.port || '';

    updateDevicePreview();
    el.deviceModal.classList.add('open');
  }

  function closeDeviceModal() {
    el.deviceModal.classList.remove('open');
  }

  function updateDevicePreview() {
    const name = el.devName.value.trim() || 'name';
    const ip = el.devIP.value.trim() || 'ip';
    const model = el.devModel.value.trim() || 'model';
    const user = el.devUsername.value.trim();
    const pass = el.devPassword.value.trim();
    const group = el.devGroup.value.trim();
    const port = el.devPort.value.trim();

    el.devPreviewLine.textContent = `${name}:${ip}:${model}:${user}:${pass}:${group}:${port}`;
  }

  async function handleDeviceSubmit(e) {
    e.preventDefault();

    const id = el.devEditId.value;
    const payload = {
      name: el.devName.value.trim(),
      ip: el.devIP.value.trim(),
      model: el.devModel.value.trim(),
      username: el.devUsername.value.trim(),
      password: el.devPassword.value.trim(),
      group: el.devGroup.value.trim(),
      port: el.devPort.value.trim(),
    };

    if (payload.name.includes(':') || payload.ip.includes(':')) {
      showToast("Поля 'Назва' та 'IP' не можуть містити двокрапку ':'", 'error');
      return;
    }

    let res;
    if (id) {
      res = await apiRequest(`/api/devices/${id}`, {
        method: 'PUT',
        body: JSON.stringify(payload),
      });
    } else {
      res = await apiRequest('/api/devices', {
        method: 'POST',
        body: JSON.stringify(payload),
      });
    }

    if (res.ok && res.data && res.data.success) {
      closeDeviceModal();
      showToast(id ? 'Пристрій успішно оновлено!' : 'Пристрій успішно додано!', 'success');
      await loadDevices();
    } else {
      showToast((res.data && res.data.error) || 'Помилка збереження пристрою', 'error');
    }
  }

  // Delete Modal
  function openDeleteModal(id) {
    const dev = state.devices.find(d => d.id === id);
    if (!dev) return;

    state.pendingDeleteId = id;
    el.deleteDeviceName.textContent = dev.name;
    el.deleteDeviceIP.textContent = dev.ip;
    el.deleteModal.classList.add('open');
  }

  function closeDeleteModal() {
    state.pendingDeleteId = null;
    el.deleteModal.classList.remove('open');
  }

  async function handleConfirmDelete() {
    if (!state.pendingDeleteId) return;

    const res = await apiRequest(`/api/devices/${state.pendingDeleteId}`, {
      method: 'DELETE',
    });

    closeDeleteModal();

    if (res.ok && res.data && res.data.success) {
      showToast('Пристрій видалено з router.db', 'success');
      await loadDevices();
    } else {
      showToast((res.data && res.data.error) || 'Помилка видалення пристрою', 'error');
    }
  }

  // Restart Oxidized Service
  function openRestartModal() {
    el.restartTerminalOutput.textContent = 'Готовий до виконання: systemctl restart oxidized.service\nНатисніть "Перезапустити зараз"...';
    el.restartModal.classList.add('open');
  }

  function closeRestartModal() {
    el.restartModal.classList.remove('open');
  }

  async function handleExecuteRestart() {
    el.btnExecuteRestart.disabled = true;
    el.btnExecuteRestart.innerHTML = `<div class="spinner"></div><span>Перезапуск...</span>`;
    el.restartTerminalOutput.textContent = 'Виконується: systemctl restart oxidized.service...\nБудь ласка, зачекайте...';

    const res = await apiRequest('/api/oxidized/restart', { method: 'POST' });

    el.btnExecuteRestart.disabled = false;
    el.btnExecuteRestart.innerHTML = `<span>Перезапустити зараз</span>`;

    if (res.ok && res.data) {
      const output = res.data.output || (res.data.success ? 'Команда виконана успішно' : 'Помилка виконання');
      const timeStr = new Date().toLocaleTimeString();
      el.restartTerminalOutput.textContent = `[${timeStr}] Результат виконання:\n${output}\n\nСтатус: ${res.data.status} (тривалість: ${res.data.duration})`;

      if (res.data.success) {
        showToast('Сервіс oxidized успішно перезавантажено!', 'success');
        updateServicePill(res.data.status);
      } else {
        showToast('Помилка при перезавантаженні сервісу oxidized', 'error');
      }
    } else {
      el.restartTerminalOutput.textContent = `Помилка запиту: ${res.error || 'Невдале звернення до сервера'}`;
      showToast('Не вдалося зв’язатися із сервісом', 'error');
    }
  }

  async function checkOxidizedStatus() {
    const res = await apiRequest('/api/oxidized/status');
    if (res.ok && res.data) {
      updateServicePill(res.data.status);
    }
  }

  function updateServicePill(status) {
    state.serviceStatus = status;
    const isActive = status === 'active' || status === 'simulated_active';

    if (isActive) {
      el.serviceStatusDot.className = 'status-dot active';
      el.serviceStatusText.textContent = 'Oxidized: Running';
      el.statServiceState.textContent = 'Active';
      el.statServiceState.style.color = 'var(--success)';
      el.statServiceIcon.className = 'stat-icon green';
    } else {
      el.serviceStatusDot.className = 'status-dot inactive';
      el.serviceStatusText.textContent = `Oxidized: ${status || 'Stopped'}`;
      el.statServiceState.textContent = status || 'Inactive';
      el.statServiceState.style.color = 'var(--danger)';
      el.statServiceIcon.className = 'stat-icon amber';
    }
  }

  // Change Password Modal
  function openChangePasswordModal() {
    el.changePasswordForm.reset();
    el.changePasswordModal.classList.add('open');
    el.currentPassword.focus();
  }

  function closeChangePasswordModal() {
    el.changePasswordModal.classList.remove('open');
  }

  async function handleChangePasswordSubmit(e) {
    e.preventDefault();

    const curr = el.currentPassword.value;
    const next = el.newPassword.value;
    const conf = el.confirmPassword.value;

    if (next !== conf) {
      showToast('Новий пароль та підтвердження не співпадають!', 'error');
      return;
    }

    if (next.length < 4) {
      showToast('Новий пароль має бути не менше 4 символів!', 'error');
      return;
    }

    const res = await apiRequest('/api/auth/change-password', {
      method: 'POST',
      body: JSON.stringify({
        current_password: curr,
        new_password: next,
      }),
    });

    if (res.ok && res.data && res.data.success) {
      closeChangePasswordModal();
      showToast('Пароль успішно змінено та записано в конфіг!', 'success');
    } else {
      showToast((res.data && res.data.error) || 'Помилка зміни пароля', 'error');
    }
  }

  // Raw Editor View
  async function toggleView() {
    if (state.activeView === 'table') {
      state.activeView = 'raw';
      el.tableViewSection.classList.add('hidden');
      el.rawViewSection.classList.remove('hidden');
      el.toggleViewText.textContent = 'Таблиця';
      await loadRawContent();
    } else {
      state.activeView = 'table';
      el.rawViewSection.classList.add('hidden');
      el.tableViewSection.classList.remove('hidden');
      el.toggleViewText.textContent = 'Raw редактор';
      await loadDevices();
    }
  }

  async function loadRawContent() {
    const res = await apiRequest('/api/devices/raw');
    if (res.ok && res.data) {
      el.rawContentArea.value = res.data.content || '';
    } else {
      showToast('Помилка завантаження вмісту файлу', 'error');
    }
  }

  async function handleSaveRaw() {
    el.btnSaveRaw.disabled = true;
    el.btnSaveRaw.innerHTML = `<div class="spinner"></div><span>Збереження...</span>`;

    const res = await apiRequest('/api/devices/raw', {
      method: 'PUT',
      body: JSON.stringify({ content: el.rawContentArea.value }),
    });

    el.btnSaveRaw.disabled = false;
    el.btnSaveRaw.innerHTML = `
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11l5 5v11a2 2 0 0 1-2 2z"></path>
        <polyline points="17 21 17 13 7 13 7 21"></polyline>
        <polyline points="7 3 7 8 15 8"></polyline>
      </svg>
      <span>Зберегти файл</span>
    `;

    if (res.ok && res.data && res.data.success) {
      showToast('Файл router.db збережено успішно! (.bak створено)', 'success');
    } else {
      showToast((res.data && res.data.error) || 'Помилка збереження файлу', 'error');
    }
  }

  // Event Listeners setup
  function setupEventListeners() {
    // Auth
    el.loginForm.addEventListener('submit', handleLogin);
    el.btnLogout.addEventListener('click', handleLogout);
    el.toggleLoginPassword.addEventListener('click', () => {
      const type = el.loginPassword.type === 'password' ? 'text' : 'password';
      el.loginPassword.type = type;
    });

    // Search and filters
    el.searchInput.addEventListener('input', (e) => {
      state.searchTerm = e.target.value.trim();
      renderDevicesTable();
    });

    el.filterModel.addEventListener('change', (e) => {
      state.filterModel = e.target.value;
      renderDevicesTable();
    });

    el.filterGroup.addEventListener('change', (e) => {
      state.filterGroup = e.target.value;
      renderDevicesTable();
    });

    // View toggle
    el.btnToggleView.addEventListener('click', toggleView);
    el.btnReloadRaw.addEventListener('click', loadRawContent);
    el.btnSaveRaw.addEventListener('click', handleSaveRaw);

    // Modals open / close
    el.btnOpenAddModal.addEventListener('click', openAddDeviceModal);
    el.btnEmptyAdd.addEventListener('click', openAddDeviceModal);
    el.btnCloseDeviceModal.addEventListener('click', closeDeviceModal);
    el.btnCancelDeviceModal.addEventListener('click', closeDeviceModal);
    el.deviceForm.addEventListener('submit', handleDeviceSubmit);

    // Device modal inputs live preview
    [el.devName, el.devIP, el.devModel, el.devUsername, el.devPassword, el.devGroup, el.devPort].forEach(input => {
      input.addEventListener('input', updateDevicePreview);
    });

    el.toggleDevPassword.addEventListener('click', () => {
      const type = el.devPassword.type === 'password' ? 'text' : 'password';
      el.devPassword.type = type;
    });

    // Delete modal
    el.btnCloseDeleteModal.addEventListener('click', closeDeleteModal);
    el.btnCancelDelete.addEventListener('click', closeDeleteModal);
    el.btnConfirmDelete.addEventListener('click', handleConfirmDelete);

    // Restart modal
    el.btnOpenRestartModal.addEventListener('click', openRestartModal);
    el.btnCloseRestartModal.addEventListener('click', closeRestartModal);
    el.btnCancelRestart.addEventListener('click', closeRestartModal);
    el.btnExecuteRestart.addEventListener('click', handleExecuteRestart);
    el.btnRefreshStatus.addEventListener('click', (e) => {
      e.stopPropagation();
      checkOxidizedStatus();
      showToast('Оновлення статусу oxidized.service...', 'info', 1500);
    });

    // Change password modal
    el.btnOpenChangePassword.addEventListener('click', openChangePasswordModal);
    el.btnCloseChangePwdModal.addEventListener('click', closeChangePasswordModal);
    el.btnCancelChangePwd.addEventListener('click', closeChangePasswordModal);
    el.changePasswordForm.addEventListener('submit', handleChangePasswordSubmit);

    // Close modals on Escape key or outside backdrop click
    window.addEventListener('keydown', (e) => {
      if (e.key === 'Escape') {
        closeDeviceModal();
        closeDeleteModal();
        closeRestartModal();
        closeChangePasswordModal();
      }
    });

    [el.deviceModal, el.deleteModal, el.restartModal, el.changePasswordModal].forEach(modal => {
      modal.addEventListener('click', (e) => {
        if (e.target === modal) {
          modal.classList.remove('open');
        }
      });
    });
  }

  // Start app on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

})();

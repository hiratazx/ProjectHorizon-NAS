/**
 * ProjectHorizon - Dashboard Application
 * TrueNAS-inspired NAS Dashboard
 */

// API Base URL
const API_BASE = '/api';

// State
let currentPath = '/media';
let refreshInterval = null;
let currentUser = null;

// ============================================
// Initialization
// ============================================

document.addEventListener('DOMContentLoaded', () => {
    initAuthForms();
    checkAuth();
});

async function checkAuth() {
    try {
        const response = await fetch(`${API_BASE}/auth/check`, {
            headers: getAuthHeaders()
        });
        const data = await response.json();

        if (data.setupRequired) {
            showSetupForm();
        } else if (!data.authenticated) {
            showLoginForm();
        } else {
            currentUser = data.user;
            showApp();
        }
    } catch (error) {
        console.error('Auth check failed:', error);
        showLoginForm();
    }
}

function showSetupForm() {
    document.getElementById('authOverlay').style.display = 'flex';
    document.getElementById('setupForm').style.display = 'block';
    document.getElementById('loginForm').style.display = 'none';
    document.getElementById('mainApp').style.display = 'none';
}

function showLoginForm() {
    document.getElementById('authOverlay').style.display = 'flex';
    document.getElementById('setupForm').style.display = 'none';
    document.getElementById('loginForm').style.display = 'block';
    document.getElementById('mainApp').style.display = 'none';
}

function showApp() {
    document.getElementById('authOverlay').style.display = 'none';
    document.getElementById('mainApp').style.display = 'flex';

    // Update user info in header
    if (currentUser) {
        document.getElementById('currentUsername').textContent = currentUser.username;
        document.getElementById('currentRole').textContent = currentUser.role.toUpperCase();

        // Show user management for admins
        if (currentUser.role === 'admin') {
            const usersSection = document.getElementById('usersSection');
            if (usersSection) usersSection.style.display = 'block';
        }
    }

    initNavigation();
    initSidebar();
    initUserMenu();
    loadDashboard();
    startAutoRefresh();
}

function initAuthForms() {
    // Setup form
    document.getElementById('setupForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('setupUsername').value;
        const password = document.getElementById('setupPassword').value;
        const confirm = document.getElementById('setupConfirm').value;

        if (password !== confirm) {
            document.getElementById('setupError').textContent = 'Passwords do not match';
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/auth/setup`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Setup failed');
            }

            localStorage.setItem('token', data.token);
            currentUser = data.user;
            showApp();
        } catch (error) {
            document.getElementById('setupError').textContent = error.message;
        }
    });

    // Login form
    document.getElementById('loginForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('loginUsername').value;
        const password = document.getElementById('loginPassword').value;

        try {
            const response = await fetch(`${API_BASE}/auth/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password })
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Login failed');
            }

            localStorage.setItem('token', data.token);
            currentUser = data.user;
            showApp();
        } catch (error) {
            document.getElementById('loginError').textContent = error.message;
        }
    });
}

function logout() {
    localStorage.removeItem('token');
    currentUser = null;
    if (refreshInterval) clearInterval(refreshInterval);
    showLoginForm();
}

function getAuthHeaders() {
    const token = localStorage.getItem('token');
    return token ? { 'Authorization': `Bearer ${token}` } : {};
}

function initUserMenu() {
    const btn = document.getElementById('userMenuBtn');
    const dropdown = document.getElementById('userDropdown');

    btn.addEventListener('click', (e) => {
        e.stopPropagation();
        dropdown.classList.toggle('open');
    });

    document.addEventListener('click', () => {
        dropdown.classList.remove('open');
    });
}

// ============================================
// Navigation
// ============================================

function initNavigation() {
    const navItems = document.querySelectorAll('.nav-item[data-page]');

    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            e.preventDefault();
            const page = item.dataset.page;
            navigateTo(page);
        });
    });
}

function navigateTo(page) {
    // Update nav active state
    document.querySelectorAll('.nav-item').forEach(item => {
        item.classList.remove('active');
        if (item.dataset.page === page) {
            item.classList.add('active');
        }
    });

    // Update page visibility
    document.querySelectorAll('.page').forEach(p => {
        p.classList.remove('active');
    });

    const pageEl = document.getElementById(`page-${page}`);
    if (pageEl) {
        pageEl.classList.add('active');
    }

    // Update title
    const titles = {
        dashboard: 'Dashboard',
        storage: 'Storage',
        docker: 'Containers',
        files: 'File Browser',
        network: 'Network',
        settings: 'Settings'
    };
    document.getElementById('pageTitle').textContent = titles[page] || page;

    // Load page data
    loadPageData(page);

    // Close sidebar on mobile
    document.getElementById('sidebar').classList.remove('open');
}

function loadPageData(page) {
    switch (page) {
        case 'dashboard':
            loadDashboard();
            break;
        case 'storage':
            loadStoragePage();
            break;
        case 'docker':
            loadContainersPage();
            break;
        case 'files':
            loadFilesPage();
            break;
        case 'network':
            loadNetworkPage();
            break;
        case 'settings':
            loadSettingsPage();
            break;
    }
}

// ============================================
// Sidebar
// ============================================

function initSidebar() {
    const menuToggle = document.getElementById('menuToggle');
    const sidebar = document.getElementById('sidebar');

    menuToggle.addEventListener('click', () => {
        sidebar.classList.toggle('open');
    });

    // Close sidebar when clicking outside
    document.addEventListener('click', (e) => {
        if (window.innerWidth <= 768) {
            if (!sidebar.contains(e.target) && !menuToggle.contains(e.target)) {
                sidebar.classList.remove('open');
            }
        }
    });
}

// ============================================
// Dashboard
// ============================================

async function loadDashboard() {
    await Promise.all([
        loadSystemInfo(),
        loadCPUInfo(),
        loadMemoryInfo(),
        loadStorageWidget(),
        loadDockerWidget(),
        loadNetworkWidget()
    ]);
}

async function loadSystemInfo() {
    try {
        const data = await fetchAPI('/system/info');

        document.getElementById('hostname').textContent = data.hostname;
        document.getElementById('systemPlatform').textContent = data.platform || '--';
        document.getElementById('systemOS').textContent = data.os || '--';
        document.getElementById('systemKernel').textContent = data.kernelVersion || '--';
        document.getElementById('systemArch').textContent = data.arch || '--';
        document.getElementById('systemUptime').textContent = formatUptime(data.uptime);
    } catch (error) {
        console.error('Failed to load system info:', error);
    }
}

async function loadCPUInfo() {
    try {
        const data = await fetchAPI('/system/cpu');

        const usage = Math.round(data.usage || 0);
        document.getElementById('cpuUsage').textContent = `${usage}%`;
        document.getElementById('cpuRingValue').textContent = usage;

        // Update ring
        const ring = document.getElementById('cpuRing');
        const circumference = 2 * Math.PI * 52;
        const offset = circumference - (usage / 100) * circumference;
        ring.style.strokeDashoffset = offset;

        // Update cores
        const coresContainer = document.getElementById('cpuCores');
        if (data.perCoreUsage && data.perCoreUsage.length > 0) {
            coresContainer.innerHTML = data.perCoreUsage.map(usage => `
                <div class="cpu-core">
                    <div class="cpu-core-fill" style="width: ${Math.round(usage)}%"></div>
                </div>
            `).join('');
        }
    } catch (error) {
        console.error('Failed to load CPU info:', error);
    }
}

async function loadMemoryInfo() {
    try {
        const data = await fetchAPI('/system/memory');

        const usage = Math.round(data.usedPercent || 0);
        document.getElementById('memUsage').textContent = `${usage}%`;
        document.getElementById('memRingValue').textContent = usage;
        document.getElementById('memUsed').textContent = formatBytes(data.used);
        document.getElementById('memTotal').textContent = formatBytes(data.total);

        // Update ring
        const ring = document.getElementById('memRing');
        const circumference = 2 * Math.PI * 52;
        const offset = circumference - (usage / 100) * circumference;
        ring.style.strokeDashoffset = offset;
    } catch (error) {
        console.error('Failed to load memory info:', error);
    }
}

async function loadStorageWidget() {
    try {
        const data = await fetchAPI('/storage/usage');
        const container = document.getElementById('storageList');

        if (!data || data.length === 0) {
            container.innerHTML = '<p class="text-secondary">No storage devices found</p>';
            return;
        }

        // Filter out virtual filesystems
        const realFs = data.filter(fs =>
            !fs.filesystem.startsWith('/dev/loop') &&
            !fs.mount.startsWith('/snap') &&
            fs.total > 0
        );

        container.innerHTML = realFs.map(fs => {
            const percent = Math.round(fs.usedPercent || 0);
            const statusClass = percent > 90 ? 'danger' : percent > 75 ? 'warning' : '';

            return `
                <div class="storage-item">
                    <div class="storage-item-header">
                        <span class="storage-name">${fs.mount}</span>
                        <span class="storage-usage">${percent}%</span>
                    </div>
                    <div class="storage-bar">
                        <div class="storage-bar-fill ${statusClass}" style="width: ${percent}%"></div>
                    </div>
                    <div class="storage-details">
                        <span>${formatBytes(fs.used)} used</span>
                        <span>${formatBytes(fs.available)} free</span>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load storage:', error);
        document.getElementById('storageList').innerHTML = '<p class="text-secondary">Failed to load storage</p>';
    }
}

async function loadDockerWidget() {
    try {
        const data = await fetchAPI('/docker/info');

        document.getElementById('dockerRunning').textContent = data.containersRunning || 0;
        document.getElementById('dockerStopped').textContent = data.containersStopped || 0;
        document.getElementById('dockerImages').textContent = data.images || 0;
    } catch (error) {
        console.error('Failed to load Docker info:', error);
        document.getElementById('dockerRunning').textContent = '-';
        document.getElementById('dockerStopped').textContent = '-';
        document.getElementById('dockerImages').textContent = '-';
    }
}

async function loadNetworkWidget() {
    try {
        const data = await fetchAPI('/system/network');
        const container = document.getElementById('networkInterfaces');

        if (!data || data.length === 0) {
            container.innerHTML = '<p class="text-secondary">No network interfaces</p>';
            return;
        }

        // Filter and show main interfaces
        const mainInterfaces = data.filter(iface =>
            iface.name !== 'lo' &&
            iface.addresses &&
            iface.addresses.length > 0
        ).slice(0, 4);

        container.innerHTML = mainInterfaces.map(iface => {
            const ipv4 = iface.addresses.find(addr => !addr.includes(':')) || iface.addresses[0];
            return `
                <div class="network-interface">
                    <span class="network-name">${iface.name}</span>
                    <span class="network-ip">${ipv4 || '--'}</span>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load network:', error);
        document.getElementById('networkInterfaces').innerHTML = '<p class="text-secondary">Failed to load network</p>';
    }
}

// ============================================
// Storage Page
// ============================================

async function loadStoragePage() {
    try {
        const data = await fetchAPI('/storage/usage');
        const container = document.getElementById('storagePools');

        if (!data || data.length === 0) {
            container.innerHTML = '<p>No storage devices found</p>';
            return;
        }

        container.innerHTML = data.filter(fs => fs.total > 0).map(fs => {
            const percent = Math.round(fs.usedPercent || 0);
            const statusClass = percent > 90 ? 'danger' : percent > 75 ? 'warning' : '';

            return `
                <div class="storage-item" style="margin-bottom: 20px;">
                    <div class="storage-item-header">
                        <div>
                            <span class="storage-name">${fs.filesystem}</span>
                            <span style="color: var(--text-tertiary); font-size: 0.875rem; margin-left: 8px;">${fs.type}</span>
                        </div>
                        <span class="storage-usage">${percent}% used</span>
                    </div>
                    <div class="storage-bar" style="height: 12px; margin: 12px 0;">
                        <div class="storage-bar-fill ${statusClass}" style="width: ${percent}%"></div>
                    </div>
                    <div class="storage-details">
                        <span>Mount: ${fs.mount}</span>
                        <span>Total: ${formatBytes(fs.total)}</span>
                        <span>Used: ${formatBytes(fs.used)}</span>
                        <span>Free: ${formatBytes(fs.available)}</span>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load storage page:', error);
    }
}

// ============================================
// Containers Page
// ============================================

async function loadContainersPage() {
    try {
        const data = await fetchAPI('/docker/containers');
        const container = document.getElementById('containersList');

        if (!data || data.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">No containers found</p>';
            return;
        }

        container.innerHTML = data.map(cont => {
            const stateClass = cont.state === 'running' ? 'running' :
                cont.state === 'paused' ? 'paused' : 'stopped';

            return `
                <div class="container-item">
                    <div class="container-info">
                        <div class="container-status ${stateClass}"></div>
                        <div>
                            <div class="container-name">${cont.name}</div>
                            <div class="container-image">${cont.image}</div>
                        </div>
                    </div>
                    <div class="container-actions">
                        ${cont.state === 'running' ? `
                            <button class="btn btn-secondary btn-sm" onclick="containerAction('${cont.id}', 'stop')">Stop</button>
                            <button class="btn btn-secondary btn-sm" onclick="containerAction('${cont.id}', 'restart')">Restart</button>
                        ` : `
                            <button class="btn btn-primary btn-sm" onclick="containerAction('${cont.id}', 'start')">Start</button>
                        `}
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load containers:', error);
        document.getElementById('containersList').innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Docker not available</p>';
    }
}

async function containerAction(id, action) {
    try {
        await fetchAPI(`/docker/containers/${id}/${action}`, { method: 'POST' });
        // Reload after action
        setTimeout(() => loadContainersPage(), 500);
    } catch (error) {
        console.error(`Failed to ${action} container:`, error);
        alert(`Failed to ${action} container`);
    }
}

function refreshContainers() {
    loadContainersPage();
}

// ============================================
// Files Page
// ============================================

async function loadFilesPage(path = currentPath) {
    try {
        const data = await fetchAPI(`/storage/browse?path=${encodeURIComponent(path)}`);
        currentPath = data.currentPath;

        // Update breadcrumb
        updateBreadcrumb(data.currentPath);

        const container = document.getElementById('filesList');

        if (!data.items || data.items.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary); padding: 40px;">Empty directory</p>';
            return;
        }

        // Add parent directory link
        let html = '';
        if (data.parent && data.parent !== data.currentPath) {
            html += `
                <div class="file-item" onclick="loadFilesPage('${data.parent}')">
                    <svg class="file-icon folder" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                    </svg>
                    <span class="file-name">..</span>
                    <span class="file-size">Parent Directory</span>
                </div>
            `;
        }

        html += data.items.map(item => {
            const icon = item.isDirectory ? `
                <svg class="file-icon folder" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                </svg>
            ` : `
                <svg class="file-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
                    <polyline points="14 2 14 8 20 8"/>
                </svg>
            `;

            const clickHandler = item.isDirectory ?
                `onclick="loadFilesPage('${item.path}')"` : '';

            return `
                <div class="file-item" ${clickHandler}>
                    ${icon}
                    <span class="file-name">${item.name}</span>
                    <span class="file-size">${item.isDirectory ? '--' : formatBytes(item.size)}</span>
                    <span class="file-modified">${formatDate(item.modified)}</span>
                </div>
            `;
        }).join('');

        container.innerHTML = html;
    } catch (error) {
        console.error('Failed to load files:', error);
        document.getElementById('filesList').innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Failed to load directory</p>';
    }
}

function updateBreadcrumb(path) {
    const breadcrumb = document.getElementById('fileBreadcrumb');
    const parts = path.split('/').filter(p => p);

    let currentPath = '';
    const items = parts.map((part, index) => {
        currentPath += '/' + part;
        const isLast = index === parts.length - 1;
        return `
            <span class="breadcrumb-item" ${isLast ? '' : `onclick="loadFilesPage('${currentPath}')"`} style="cursor: ${isLast ? 'default' : 'pointer'}; ${isLast ? 'color: var(--text-primary)' : ''}">${part}</span>
            ${isLast ? '' : '<span class="breadcrumb-separator">/</span>'}
        `;
    });

    breadcrumb.innerHTML = `
        <span class="breadcrumb-item" onclick="loadFilesPage('/')" style="cursor: pointer;">/</span>
        ${items.join('')}
    `;
}

// ============================================
// Network Page
// ============================================

async function loadNetworkPage() {
    try {
        const data = await fetchAPI('/system/network');
        const container = document.getElementById('networkList');

        if (!data || data.length === 0) {
            container.innerHTML = '<p>No network interfaces found</p>';
            return;
        }

        container.innerHTML = data.filter(iface => iface.name !== 'lo').map(iface => {
            const addresses = iface.addresses || [];

            return `
                <div class="network-card">
                    <div class="network-card-header">
                        <span class="network-card-name">${iface.name}</span>
                        <span class="network-card-mac">${iface.mac || '--'}</span>
                    </div>
                    <div style="margin-bottom: 12px;">
                        ${addresses.map(addr => `<div style="color: var(--text-secondary); font-family: monospace; font-size: 0.875rem;">${addr}</div>`).join('')}
                    </div>
                    <div class="network-card-stats">
                        <div class="network-stat">
                            <span class="network-stat-label">Sent</span>
                            <span class="network-stat-value">${formatBytes(iface.bytesSent || 0)}</span>
                        </div>
                        <div class="network-stat">
                            <span class="network-stat-label">Received</span>
                            <span class="network-stat-value">${formatBytes(iface.bytesRecv || 0)}</span>
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load network page:', error);
    }
}

// ============================================
// Auto Refresh
// ============================================

function startAutoRefresh() {
    // Refresh dashboard data every 5 seconds
    refreshInterval = setInterval(() => {
        const activePage = document.querySelector('.page.active');
        if (activePage && activePage.id === 'page-dashboard') {
            loadCPUInfo();
            loadMemoryInfo();
        }
    }, 5000);
}

// ============================================
// Utilities
// ============================================

async function fetchAPI(endpoint, options = {}) {
    const response = await fetch(`${API_BASE}${endpoint}`, {
        ...options,
        headers: {
            'Content-Type': 'application/json',
            ...getAuthHeaders(),
            ...options.headers
        }
    });

    if (response.status === 401) {
        logout();
        throw new Error('Session expired');
    }

    if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error || `API Error: ${response.status}`);
    }

    return response.json();
}

function formatBytes(bytes) {
    if (!bytes || bytes === 0) return '0 B';

    const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));

    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${sizes[i]}`;
}

function formatUptime(seconds) {
    if (!seconds) return '--';

    const days = Math.floor(seconds / 86400);
    const hours = Math.floor((seconds % 86400) / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (days > 0) {
        return `${days}d ${hours}h`;
    } else if (hours > 0) {
        return `${hours}h ${minutes}m`;
    } else {
        return `${minutes}m`;
    }
}

function formatDate(timestamp) {
    if (!timestamp) return '--';

    const date = new Date(timestamp * 1000);
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

// ============================================
// Settings Page
// ============================================

async function loadSettingsPage() {
    loadVolumes();
    if (currentUser && currentUser.role === 'admin') {
        loadUsers();
    }
}

async function loadVolumes() {
    try {
        const volumes = await fetchAPI('/settings/volumes');
        const container = document.getElementById('volumesList');

        if (!volumes || volumes.length === 0) {
            container.innerHTML = '<p style="color: var(--text-secondary); text-align: center; padding: 20px;">No volumes configured. Click "Add Volume" to add one.</p>';
            return;
        }

        container.innerHTML = volumes.map(vol => `
            <div class="volume-item">
                <div class="volume-info">
                    <div class="volume-name">${vol.name}</div>
                    <div class="volume-paths">
                        <span><strong>Host:</strong> ${vol.hostPath}</span>
                        <span><strong>NAS:</strong> ${vol.mountPath}</span>
                        <span class="badge ${vol.mode === 'rw' ? 'badge-rw' : 'badge-ro'}">${vol.mode === 'rw' ? 'Read-Write' : 'Read-Only'}</span>
                    </div>
                </div>
                <div class="volume-actions">
                    <button class="btn btn-secondary btn-sm" onclick="deleteVolume('${vol.id}')">Delete</button>
                </div>
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load volumes:', error);
        document.getElementById('volumesList').innerHTML = '<p style="color: var(--text-secondary);">Failed to load volumes</p>';
    }
}

async function loadUsers() {
    try {
        const users = await fetchAPI('/users');
        const container = document.getElementById('usersList');

        if (!users || users.length === 0) {
            container.innerHTML = '<p style="color: var(--text-secondary);">No users found</p>';
            return;
        }

        container.innerHTML = users.map(user => `
            <div class="user-item">
                <div class="user-info-row">
                    <div class="user-name-cell">${user.username}</div>
                    <div class="user-details">
                        <span class="badge ${user.role === 'admin' ? 'badge-admin' : 'badge-user'}">${user.role}</span>
                        <span class="badge ${user.permission === 'read-write' ? 'badge-rw' : 'badge-ro'}">${user.permission}</span>
                    </div>
                </div>
                ${currentUser && currentUser.id !== user.id ? `
                    <div class="user-actions">
                        <button class="btn btn-secondary btn-sm" onclick="deleteUser('${user.id}')">Delete</button>
                    </div>
                ` : ''}
            </div>
        `).join('');
    } catch (error) {
        console.error('Failed to load users:', error);
        document.getElementById('usersList').innerHTML = '<p style="color: var(--text-secondary);">Failed to load users</p>';
    }
}

// Volume Modal
function showAddVolumeModal() {
    document.getElementById('volumeModal').style.display = 'flex';
    document.getElementById('volumeForm').reset();
    document.getElementById('volumeError').textContent = '';
}

function closeVolumeModal() {
    document.getElementById('volumeModal').style.display = 'none';
}

document.getElementById('volumeForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();

    const data = {
        name: document.getElementById('volumeName').value,
        hostPath: document.getElementById('hostPath').value,
        mountPath: document.getElementById('mountPath').value,
        mode: document.getElementById('volumeMode').value
    };

    try {
        await fetchAPI('/settings/volumes', {
            method: 'POST',
            body: JSON.stringify(data)
        });
        closeVolumeModal();
        loadVolumes();
    } catch (error) {
        document.getElementById('volumeError').textContent = error.message;
    }
});

async function deleteVolume(id) {
    if (!confirm('Are you sure you want to delete this volume?')) return;

    try {
        await fetchAPI(`/settings/volumes/${id}`, { method: 'DELETE' });
        loadVolumes();
    } catch (error) {
        alert('Failed to delete volume: ' + error.message);
    }
}

// User Modal
function showAddUserModal() {
    document.getElementById('userModal').style.display = 'flex';
    document.getElementById('userForm').reset();
    document.getElementById('userError').textContent = '';
}

function closeUserModal() {
    document.getElementById('userModal').style.display = 'none';
}

document.getElementById('userForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();

    const data = {
        username: document.getElementById('newUsername').value,
        password: document.getElementById('newPassword').value,
        role: document.getElementById('userRole').value,
        permission: document.getElementById('userPerm').value
    };

    try {
        await fetchAPI('/users', {
            method: 'POST',
            body: JSON.stringify(data)
        });
        closeUserModal();
        loadUsers();
    } catch (error) {
        document.getElementById('userError').textContent = error.message;
    }
});

async function deleteUser(id) {
    if (!confirm('Are you sure you want to delete this user?')) return;

    try {
        await fetchAPI(`/users/${id}`, { method: 'DELETE' });
        loadUsers();
    } catch (error) {
        alert('Failed to delete user: ' + error.message);
    }
}


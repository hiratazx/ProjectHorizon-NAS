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
    initOnboarding();
    checkAuth();
});

// Onboarding state
let currentOnboardingStep = 1;
let onboardingData = {
    account: null,
    volume: null
};

async function checkAuth() {
    try {
        const response = await fetch(`${API_BASE}/auth/check`, {
            headers: getAuthHeaders()
        });
        const data = await response.json();

        if (data.setupRequired) {
            showOnboarding();
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

function showOnboarding() {
    document.getElementById('authOverlay').style.display = 'flex';
    document.getElementById('onboardingContainer').style.display = 'block';
    document.getElementById('loginModal').style.display = 'none';
    document.getElementById('mainApp').style.display = 'none';
    currentOnboardingStep = 1;
    updateOnboardingUI();
}

function showLoginForm() {
    document.getElementById('authOverlay').style.display = 'flex';
    document.getElementById('onboardingContainer').style.display = 'none';
    document.getElementById('loginModal').style.display = 'block';
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

// Onboarding Navigation
function initOnboarding() {
    // Account form
    document.getElementById('onboardingAccountForm')?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('obUsername').value;
        const password = document.getElementById('obPassword').value;
        const confirm = document.getElementById('obConfirm').value;

        if (password !== confirm) {
            document.getElementById('obAccountError').textContent = 'Passwords do not match';
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
            onboardingData.account = { username };
            nextOnboardingStep();
        } catch (error) {
            document.getElementById('obAccountError').textContent = error.message;
        }
    });

    // Volume form
    document.getElementById('onboardingVolumeForm')?.addEventListener('submit', async (e) => {
        e.preventDefault();
        const name = document.getElementById('obVolumeName').value;
        const hostPath = document.getElementById('obHostPath').value;
        const mountPath = document.getElementById('obMountPath').value;

        if (!hostPath) {
            document.getElementById('obVolumeError').textContent = 'Host path is required';
            return;
        }

        try {
            const response = await fetch(`${API_BASE}/settings/volumes`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    ...getAuthHeaders()
                },
                body: JSON.stringify({ name, hostPath, mountPath, mode: 'rw' })
            });

            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Failed to add volume');
            }

            onboardingData.volume = { name, hostPath };
            document.getElementById('volumeSummary').style.display = 'flex';
            document.getElementById('volumeSummaryText').textContent = `Volume "${name}" configured`;
            nextOnboardingStep();
        } catch (error) {
            document.getElementById('obVolumeError').textContent = error.message;
        }
    });
}

function nextOnboardingStep() {
    if (currentOnboardingStep < 5) {
        currentOnboardingStep++;
        updateOnboardingUI();
    }
}

function prevOnboardingStep() {
    if (currentOnboardingStep > 1) {
        currentOnboardingStep--;
        updateOnboardingUI();
    }
}

function skipVolumeStep() {
    nextOnboardingStep();
}

function updateOnboardingUI() {
    // Update steps visibility
    for (let i = 1; i <= 5; i++) {
        const step = document.getElementById(`step${i}`);
        const progressStep = document.querySelector(`.progress-step[data-step="${i}"]`);

        if (step) {
            step.classList.toggle('active', i === currentOnboardingStep);
        }
        if (progressStep) {
            progressStep.classList.remove('active', 'completed');
            if (i === currentOnboardingStep) {
                progressStep.classList.add('active');
            } else if (i < currentOnboardingStep) {
                progressStep.classList.add('completed');
            }
        }
    }
}

function finishOnboarding() {
    showApp();
}

function initAuthForms() {
    // Login form (for returning users)
    document.getElementById('loginForm')?.addEventListener('submit', async (e) => {
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
    loadContainersList();
    loadImagesList();
}

let currentLogsContainerId = null;

async function loadContainersList() {
    try {
        const data = await fetchAPI('/docker/containers');
        const container = document.getElementById('containersList');

        if (!data || data.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary); padding: 40px;">No containers found</p>';
            return;
        }

        container.innerHTML = data.map(cont => {
            const stateClass = cont.state === 'running' ? 'running' :
                cont.state === 'paused' ? 'paused' : 'stopped';

            return `
                <div class="container-card">
                    <div class="container-status ${stateClass}"></div>
                    <div class="container-info">
                        <div class="container-name">${cont.name}</div>
                        <div class="container-image">${cont.image}</div>
                    </div>
                    <div class="container-stats" id="stats-${cont.id}">
                        <span class="container-stat">
                            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <rect x="4" y="4" width="16" height="16" rx="2" ry="2"/>
                                <rect x="9" y="9" width="6" height="6"/>
                                <line x1="9" y1="1" x2="9" y2="4"/>
                                <line x1="15" y1="1" x2="15" y2="4"/>
                                <line x1="9" y1="20" x2="9" y2="23"/>
                                <line x1="15" y1="20" x2="15" y2="23"/>
                            </svg>
                            ${cont.status}
                        </span>
                    </div>
                    <div class="container-actions">
                        ${cont.state === 'running' ? `
                            <button class="btn btn-secondary" onclick="containerAction('${cont.id}', 'stop')">Stop</button>
                            <button class="btn btn-secondary" onclick="containerAction('${cont.id}', 'restart')">Restart</button>
                        ` : `
                            <button class="btn btn-primary" onclick="containerAction('${cont.id}', 'start')">Start</button>
                        `}
                        <button class="btn btn-secondary" onclick="showContainerLogs('${cont.id}', '${cont.name}')">Logs</button>
                        <button class="btn btn-danger" onclick="deleteContainer('${cont.id}', '${cont.name}')">Delete</button>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load containers:', error);
        document.getElementById('containersList').innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Docker not available</p>';
    }
}

async function loadImagesList() {
    try {
        const data = await fetchAPI('/docker/images');
        const container = document.getElementById('imagesList');

        if (!data || data.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary); padding: 20px;">No images found</p>';
            return;
        }

        container.innerHTML = data.map(img => {
            const tags = img.tags && img.tags.length > 0 ? img.tags.join(', ') : '<none>';
            const size = formatBytes(img.size);

            return `
                <div class="image-card">
                    <div class="image-icon">
                        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="3" y="3" width="18" height="18" rx="2" ry="2"/>
                            <circle cx="8.5" cy="8.5" r="1.5"/>
                            <polyline points="21 15 16 10 5 21"/>
                        </svg>
                    </div>
                    <div class="image-info">
                        <div class="image-tag">${tags}</div>
                        <div class="image-meta">ID: ${img.id} • Size: ${size}</div>
                    </div>
                    <div class="image-actions">
                        <button class="btn btn-danger" onclick="deleteImage('${img.id}')">Delete</button>
                    </div>
                </div>
            `;
        }).join('');
    } catch (error) {
        console.error('Failed to load images:', error);
        document.getElementById('imagesList').innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Failed to load images</p>';
    }
}

async function containerAction(id, action) {
    try {
        await fetchAPI(`/docker/containers/${id}/${action}`, { method: 'POST' });
        setTimeout(() => loadContainersList(), 500);
    } catch (error) {
        console.error(`Failed to ${action} container:`, error);
        alert(`Failed to ${action} container`);
    }
}

async function deleteContainer(id, name) {
    if (!confirm(`Are you sure you want to delete container "${name}"? This cannot be undone.`)) {
        return;
    }

    try {
        await fetchAPI(`/docker/containers/${id}`, { method: 'DELETE' });
        loadContainersList();
    } catch (error) {
        console.error('Failed to delete container:', error);
        alert('Failed to delete container');
    }
}

async function deleteImage(id) {
    if (!confirm('Are you sure you want to delete this image? This cannot be undone.')) {
        return;
    }

    try {
        await fetchAPI(`/docker/images/${id}`, { method: 'DELETE' });
        loadImagesList();
    } catch (error) {
        console.error('Failed to delete image:', error);
        alert('Failed to delete image: ' + error.message);
    }
}

async function showContainerLogs(id, name) {
    currentLogsContainerId = id;
    document.getElementById('logsModalTitle').textContent = `Logs: ${name}`;
    document.getElementById('logsContent').textContent = 'Loading logs...';
    document.getElementById('logsModal').style.display = 'flex';

    await refreshLogs();
}

async function refreshLogs() {
    if (!currentLogsContainerId) return;

    try {
        const response = await fetch(`${API_BASE}/docker/containers/${currentLogsContainerId}/logs`, {
            headers: getAuthHeaders()
        });
        const logs = await response.text();
        document.getElementById('logsContent').textContent = logs || 'No logs available';
    } catch (error) {
        document.getElementById('logsContent').textContent = 'Failed to load logs: ' + error.message;
    }
}

function closeLogsModal() {
    document.getElementById('logsModal').style.display = 'none';
    currentLogsContainerId = null;
}

function refreshContainers() {
    loadContainersPage();
}


// ============================================
// Files Page - Volume-Based Browser
// ============================================

let currentVolume = null;
let navigationHistory = [];
let historyIndex = -1;
let allFileItems = []; // Store for filtering

// Navigation functions
function goBack() {
    if (historyIndex > 0) {
        historyIndex--;
        const entry = navigationHistory[historyIndex];
        currentVolume = entry.volume;
        loadFilesPageInternal(entry.path, false);
    }
}

function goForward() {
    if (historyIndex < navigationHistory.length - 1) {
        historyIndex++;
        const entry = navigationHistory[historyIndex];
        currentVolume = entry.volume;
        loadFilesPageInternal(entry.path, false);
    }
}

function goUp() {
    if (!currentVolume) return;

    // If at volume root, go back to volumes list
    if (currentPath === currentVolume.path) {
        backToVolumes();
    } else {
        // Go to parent directory
        const parent = currentPath.substring(0, currentPath.lastIndexOf('/')) || currentVolume.path;
        if (parent.startsWith(currentVolume.path)) {
            loadFilesPage(parent);
        } else {
            loadFilesPage(currentVolume.path);
        }
    }
}

function refreshFiles() {
    if (currentVolume) {
        loadFilesPageInternal(currentPath, false);
    } else {
        loadFilesPage(null);
    }
}

function handlePathInput(event) {
    if (event.key === 'Enter') {
        goToPath();
    }
}

function goToPath() {
    const pathInput = document.getElementById('pathInput');
    const path = pathInput.value.trim();
    if (path) {
        // Check if path is within current volume or reset
        if (currentVolume && path.startsWith(currentVolume.path)) {
            loadFilesPage(path);
        } else {
            // Try to navigate directly
            currentVolume = { id: 'custom', path: path, name: 'Custom' };
            loadFilesPage(path);
        }
    }
}

function filterFiles() {
    const searchTerm = document.getElementById('fileSearch').value.toLowerCase();
    const fileItems = document.querySelectorAll('.file-item');

    fileItems.forEach(item => {
        const fileName = item.querySelector('.file-name')?.textContent.toLowerCase() || '';
        if (fileName === '..' || fileName === 'back to volumes' || fileName.includes(searchTerm)) {
            item.style.display = '';
        } else {
            item.style.display = 'none';
        }
    });
}

function updateToolbarState() {
    const btnBack = document.getElementById('btnBack');
    const btnForward = document.getElementById('btnForward');
    const btnUp = document.getElementById('btnUp');
    const pathInput = document.getElementById('pathInput');

    if (btnBack) btnBack.disabled = historyIndex <= 0;
    if (btnForward) btnForward.disabled = historyIndex >= navigationHistory.length - 1;
    if (btnUp) btnUp.disabled = !currentVolume;
    if (pathInput) pathInput.value = currentVolume ? currentPath : 'Volumes';
}

function addToHistory(path, volume) {
    // Remove forward history when navigating to new path
    if (historyIndex < navigationHistory.length - 1) {
        navigationHistory = navigationHistory.slice(0, historyIndex + 1);
    }
    navigationHistory.push({ path, volume });
    historyIndex = navigationHistory.length - 1;
    updateToolbarState();
}

async function loadFilesPage(path = null) {
    loadFilesPageInternal(path, true);
}

async function loadFilesPageInternal(path = null, addHistory = true) {
    const container = document.getElementById('filesList');
    const breadcrumb = document.getElementById('fileBreadcrumb');

    // If no path specified and no current volume, show volumes list
    if (!path && !currentVolume) {
        try {
            const volumes = await fetchAPI('/settings/volumes');

            if (!volumes || volumes.length === 0) {
                container.innerHTML = `
                    <div style="text-align: center; padding: 60px 20px;">
                        <p style="color: var(--text-secondary); margin-bottom: 16px;">No volumes configured</p>
                        <button class="btn btn-primary" onclick="navigateTo('settings')">Add Volume in Settings</button>
                    </div>
                `;
                breadcrumb.innerHTML = '<span class="breadcrumb-item">Volumes</span>';
                return;
            }

            breadcrumb.innerHTML = '<span class="breadcrumb-item">Volumes</span>';

            container.innerHTML = `
                <div class="volumes-grid">
                    ${volumes.map(vol => `
                        <div class="volume-card" onclick="browseVolume('${vol.id}', '${vol.hostPath}', '${vol.name}')">
                            <div class="volume-card-icon">
                                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                    <ellipse cx="12" cy="5" rx="9" ry="3"/>
                                    <path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/>
                                    <path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/>
                                </svg>
                            </div>
                            <div class="volume-card-name">${vol.name}</div>
                            <div class="volume-card-path">${vol.hostPath}</div>
                        </div>
                    `).join('')}
                </div>
            `;
        } catch (error) {
            console.error('Failed to load volumes:', error);
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Failed to load volumes</p>';
        }
        return;
    }

    // Browse directory within volume
    const browsePath = path || (currentVolume ? currentVolume.path : currentPath);

    try {
        const data = await fetchAPI(`/storage/browse?path=${encodeURIComponent(browsePath)}`);
        currentPath = data.currentPath;

        // Add to history if needed
        if (addHistory) {
            addToHistory(currentPath, currentVolume);
        }
        updateToolbarState();

        // Update breadcrumb with volume name
        updateBreadcrumb(data.currentPath);

        if (!data.items || data.items.length === 0) {
            container.innerHTML = '<p style="text-align: center; color: var(--text-secondary); padding: 40px;">Empty directory</p>';
            return;
        }

        // Add back to volumes or parent directory link
        let html = '';

        // Check if we're at volume root
        if (currentVolume && data.currentPath === currentVolume.path) {
            html += `
                <div class="file-item" onclick="backToVolumes()">
                    <svg class="file-icon folder" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M15 18l-6-6 6-6"/>
                    </svg>
                    <span class="file-name">Back to Volumes</span>
                    <span class="file-size"></span>
                </div>
            `;
        } else if (data.parent && data.parent !== data.currentPath) {
            // Don't allow going above volume root
            const canGoUp = !currentVolume || data.parent.startsWith(currentVolume.path);
            if (canGoUp) {
                html += `
                    <div class="file-item" onclick="loadFilesPage('${data.parent}')">
                        <svg class="file-icon folder" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"/>
                        </svg>
                        <span class="file-name">..</span>
                        <span class="file-size">Parent Directory</span>
                    </div>
                `;
            } else {
                html += `
                    <div class="file-item" onclick="backToVolumes()">
                        <svg class="file-icon folder" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M15 18l-6-6 6-6"/>
                        </svg>
                        <span class="file-name">Back to Volumes</span>
                        <span class="file-size"></span>
                    </div>
                `;
            }
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
        container.innerHTML = '<p style="text-align: center; color: var(--text-secondary);">Failed to load directory</p>';
    }
}

function browseVolume(id, path, name) {
    currentVolume = { id, path, name };
    currentPath = path;
    loadFilesPage(path);
}

function backToVolumes() {
    currentVolume = null;
    currentPath = '/media';
    loadFilesPage(null);
}

function updateBreadcrumb(path) {
    const breadcrumb = document.getElementById('fileBreadcrumb');

    if (!currentVolume) {
        breadcrumb.innerHTML = '<span class="breadcrumb-item">Volumes</span>';
        return;
    }

    // Calculate relative path from volume root
    const relativePath = path.replace(currentVolume.path, '').replace(/^\//, '');
    const parts = relativePath ? relativePath.split('/').filter(p => p) : [];

    let currentBuildPath = currentVolume.path;
    const items = parts.map((part, index) => {
        currentBuildPath += '/' + part;
        const isLast = index === parts.length - 1;
        return `
            <span class="breadcrumb-item" ${isLast ? '' : `onclick="loadFilesPage('${currentBuildPath}')"`} style="cursor: ${isLast ? 'default' : 'pointer'}; ${isLast ? 'color: var(--text-primary)' : ''}">${part}</span>
            ${isLast ? '' : '<span class="breadcrumb-separator">/</span>'}
        `;
    });

    breadcrumb.innerHTML = `
        <span class="breadcrumb-item" onclick="backToVolumes()" style="cursor: pointer;">Volumes</span>
        <span class="breadcrumb-separator">/</span>
        <span class="breadcrumb-item" onclick="loadFilesPage('${currentVolume.path}')" style="cursor: pointer;">${currentVolume.name}</span>
        ${items.length > 0 ? '<span class="breadcrumb-separator">/</span>' : ''}
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


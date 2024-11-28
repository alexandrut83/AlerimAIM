// Admin Panel JavaScript

// Global variables
let hashrateChart = null;
let hashrateData = [];
let activeMiners = [];
let users = [];
let wallets = [];

// Initialize the admin panel
document.addEventListener('DOMContentLoaded', () => {
    initializeNavigation();
    initializeHashrateChart();
    loadDashboardData();
    setupEventListeners();
    
    // Load initial data
    loadUsers();
    loadWallets();
    loadMiners();

    // Start periodic updates
    setInterval(updateDashboardData, 5000);
    setInterval(updateMinersData, 10000);
});

// Navigation
function initializeNavigation() {
    const navLinks = document.querySelectorAll('.admin-nav a');
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            const targetId = e.target.getAttribute('href').substring(1);
            showSection(targetId);
        });
    });
}

function showSection(sectionId) {
    const sections = document.querySelectorAll('section');
    sections.forEach(section => {
        section.classList.remove('active');
    });
    document.getElementById(sectionId).classList.add('active');
    
    const navLinks = document.querySelectorAll('.admin-nav a');
    navLinks.forEach(link => {
        link.classList.remove('active');
        if (link.getAttribute('href') === `#${sectionId}`) {
            link.classList.add('active');
        }
    });
}

// Dashboard
function initializeHashrateChart() {
    const ctx = document.getElementById('hashrate-chart').getContext('2d');
    hashrateChart = new Chart(ctx, {
        type: 'line',
        data: {
            labels: [],
            datasets: [{
                label: 'Network Hashrate (H/s)',
                data: [],
                borderColor: '#3498db',
                tension: 0.4,
                fill: false
            }]
        },
        options: {
            responsive: true,
            scales: {
                y: {
                    beginAtZero: true,
                    title: {
                        display: true,
                        text: 'Hashrate (H/s)'
                    }
                },
                x: {
                    title: {
                        display: true,
                        text: 'Time'
                    }
                }
            }
        }
    });
}

async function loadDashboardData() {
    try {
        const response = await fetch('/api/stats');
        const data = await response.json();
        
        updateDashboardStats(data);
        updateHashrateChart(data.hashrate);
    } catch (error) {
        console.error('Error loading dashboard data:', error);
    }
}

function updateDashboardStats(data) {
    document.getElementById('total-hashrate').textContent = formatHashrate(data.hashrate);
    document.getElementById('active-miners').textContent = data.activeMiners;
    document.getElementById('total-users').textContent = data.totalUsers;
    document.getElementById('network-difficulty').textContent = formatDifficulty(data.difficulty);
}

function updateHashrateChart(hashrate) {
    const now = new Date().toLocaleTimeString();
    hashrateData.push({ time: now, hashrate: hashrate });
    
    if (hashrateData.length > 20) {
        hashrateData.shift();
    }
    
    hashrateChart.data.labels = hashrateData.map(d => d.time);
    hashrateChart.data.datasets[0].data = hashrateData.map(d => d.hashrate);
    hashrateChart.update();
}

// Users Management
async function loadUsers() {
    try {
        const response = await fetch('/api/users');
        users = await response.json();
        renderUsersTable();
    } catch (error) {
        console.error('Error loading users:', error);
    }
}

function renderUsersTable() {
    const tbody = document.getElementById('users-table-body');
    tbody.innerHTML = users.map(user => `
        <tr>
            <td>${user.username}</td>
            <td>${user.email}</td>
            <td>${user.role}</td>
            <td>${user.status}</td>
            <td>
                <button onclick="editUser('${user.id}')">Edit</button>
                <button onclick="deleteUser('${user.id}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

// Wallets Management
async function loadWallets() {
    try {
        const response = await fetch('/api/wallets');
        wallets = await response.json();
        renderWalletsTable();
    } catch (error) {
        console.error('Error loading wallets:', error);
    }
}

function renderWalletsTable() {
    const tbody = document.getElementById('wallets-table-body');
    tbody.innerHTML = wallets.map(wallet => `
        <tr>
            <td>${wallet.address}</td>
            <td>${formatBalance(wallet.balance)}</td>
            <td>${wallet.owner}</td>
            <td>${new Date(wallet.created).toLocaleDateString()}</td>
            <td>
                <button onclick="viewWallet('${wallet.address}')">View</button>
                <button onclick="exportPrivateKey('${wallet.address}')">Export</button>
            </td>
        </tr>
    `).join('');
}

// Miners Management
async function loadMiners() {
    try {
        const response = await fetch('/api/miners');
        activeMiners = await response.json();
        renderMinersTable();
    } catch (error) {
        console.error('Error loading miners:', error);
    }
}

function renderMinersTable() {
    const tbody = document.getElementById('miners-table-body');
    tbody.innerHTML = activeMiners.map(miner => `
        <tr>
            <td>${miner.workerId}</td>
            <td>${formatHashrate(miner.hashrate)}</td>
            <td>${miner.status}</td>
            <td>${new Date(miner.lastShare).toLocaleString()}</td>
            <td>
                <button onclick="stopMiner('${miner.workerId}')">Stop</button>
                <button onclick="deleteMiner('${miner.workerId}')">Delete</button>
            </td>
        </tr>
    `).join('');
}

// Modal Functions
function showAddUserModal() {
    document.getElementById('add-user-modal').style.display = 'block';
}

function showAddMinerModal() {
    document.getElementById('add-miner-modal').style.display = 'block';
}

function closeModal(modalId) {
    document.getElementById(modalId).style.display = 'none';
}

// Utility Functions
function formatHashrate(hashrate) {
    if (hashrate >= 1e12) return `${(hashrate / 1e12).toFixed(2)} TH/s`;
    if (hashrate >= 1e9) return `${(hashrate / 1e9).toFixed(2)} GH/s`;
    if (hashrate >= 1e6) return `${(hashrate / 1e6).toFixed(2)} MH/s`;
    if (hashrate >= 1e3) return `${(hashrate / 1e3).toFixed(2)} KH/s`;
    return `${hashrate.toFixed(2)} H/s`;
}

function formatBalance(balance) {
    return `${(balance / 1e8).toFixed(8)} AIM`;
}

function formatDifficulty(difficulty) {
    return difficulty.toExponential(2);
}

// Event Listeners
function setupEventListeners() {
    // Add User Form
    document.getElementById('add-user-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        try {
            const response = await fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(Object.fromEntries(formData))
            });
            if (response.ok) {
                closeModal('add-user-modal');
                loadUsers();
            }
        } catch (error) {
            console.error('Error adding user:', error);
        }
    });

    // Add Miner Form
    document.getElementById('add-miner-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const formData = new FormData(e.target);
        try {
            const response = await fetch('/api/miners', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(Object.fromEntries(formData))
            });
            if (response.ok) {
                closeModal('add-miner-modal');
                loadMiners();
            }
        } catch (error) {
            console.error('Error adding miner:', error);
        }
    });

    // User Search
    document.getElementById('user-search').addEventListener('input', (e) => {
        const searchTerm = e.target.value.toLowerCase();
        const filteredUsers = users.filter(user => 
            user.username.toLowerCase().includes(searchTerm) ||
            user.email.toLowerCase().includes(searchTerm)
        );
        renderUsersTable(filteredUsers);
    });
}

// Authentication
function logout() {
    // Clear session/token
    localStorage.removeItem('admin_token');
    // Redirect to login page
    window.location.href = '/login.html';
}

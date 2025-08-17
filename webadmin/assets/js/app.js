// Main application initialization and utilities

class App {
    constructor() {
        this.bearerToken = this.getBearerToken();
        this.currentUser = null;
        this.systemHealth = null;
    }

    getBearerToken() {
        return localStorage.getItem('gomail_token');
    }

    setBearerToken(token) {
        this.bearerToken = token;
        localStorage.setItem('gomail_token', token);
    }

    async initialize() {
        // Check if user is logged in
        if (!this.bearerToken) {
            this.redirectToLogin();
            return;
        }
        
        try {
            // Check if we have a valid token by testing the API
            await window.api.getHealth();
            
            // Start SSE connection for real-time updates
            window.sse.connect();
            
            // Load initial system health
            await this.loadSystemHealth();
            
            console.log('App initialized successfully');
        } catch (error) {
            console.error('Failed to initialize app:', error);
            this.handleAuthError();
        }
    }

    redirectToLogin() {
        window.location.href = '/login.html';
    }

    logout() {
        localStorage.removeItem('gomail_token');
        localStorage.removeItem('gomail_user');
        this.redirectToLogin();
    }

    async loadSystemHealth() {
        try {
            this.systemHealth = await window.api.getHealth();
            this.updateSystemStatus();
        } catch (error) {
            console.error('Failed to load system health:', error);
        }
    }

    updateSystemStatus() {
        const statusElement = document.querySelector('.system-status');
        if (statusElement && this.systemHealth) {
            const isHealthy = this.systemHealth.status === 'healthy';
            statusElement.innerHTML = `
                <div class="flex items-center space-x-2">
                    <div class="w-2 h-2 ${isHealthy ? 'bg-green-400' : 'bg-red-400'} rounded-full"></div>
                    <span>System ${isHealthy ? 'Healthy' : 'Issues'}</span>
                </div>
            `;
        }
    }

    handleAuthError() {
        this.logout();
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `fixed top-4 right-4 max-w-sm w-full bg-white shadow-lg rounded-lg pointer-events-auto z-50 transform transition-transform duration-300 translate-x-full`;
        
        const bgColor = {
            'success': 'bg-green-50 border-green-200',
            'error': 'bg-red-50 border-red-200',
            'warning': 'bg-yellow-50 border-yellow-200',
            'info': 'bg-blue-50 border-blue-200'
        }[type] || 'bg-blue-50 border-blue-200';
        
        const textColor = {
            'success': 'text-green-800',
            'error': 'text-red-800',
            'warning': 'text-yellow-800',
            'info': 'text-blue-800'
        }[type] || 'text-blue-800';

        notification.innerHTML = `
            <div class="p-4 border ${bgColor} rounded-lg">
                <div class="flex">
                    <div class="flex-shrink-0">
                        <svg class="h-5 w-5 ${textColor}" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd"/>
                        </svg>
                    </div>
                    <div class="ml-3">
                        <p class="text-sm font-medium ${textColor}">${message}</p>
                    </div>
                    <div class="ml-auto pl-3">
                        <button onclick="this.parentElement.parentElement.parentElement.remove()" class="text-gray-400 hover:text-gray-600">
                            <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                                <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd"/>
                            </svg>
                        </button>
                    </div>
                </div>
            </div>
        `;

        document.body.appendChild(notification);
        
        // Animate in
        setTimeout(() => {
            notification.classList.remove('translate-x-full');
        }, 100);

        // Auto remove after 5 seconds
        setTimeout(() => {
            notification.classList.add('translate-x-full');
            setTimeout(() => notification.remove(), 300);
        }, 5000);
    }

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    }

    formatFileSize(bytes) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const sizes = ['Bytes', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    }

    formatDuration(seconds) {
        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const secs = seconds % 60;
        
        if (hours > 0) {
            return `${hours}h ${minutes}m ${secs}s`;
        } else if (minutes > 0) {
            return `${minutes}m ${secs}s`;
        } else {
            return `${secs}s`;
        }
    }
}

// Global app instance
window.app = new App();

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    window.app.initialize();
});
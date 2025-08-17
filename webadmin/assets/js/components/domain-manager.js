// Domain Manager Component for domain configuration and health monitoring

class DomainManager {
    constructor(container) {
        this.container = container;
        this.domains = [];
    }

    render(domainsData) {
        this.domains = domainsData.domains || [];
        
        this.container.innerHTML = `
            <div class="space-y-6">
                <!-- Domains List -->
                <div class="bg-white rounded-lg border border-gray-200">
                    ${this.domains.length === 0 ? this.renderEmptyState() : this.renderDomainsTable()}
                </div>
                
                <!-- Add Domain Modal (hidden by default) -->
                <div id="add-domain-modal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
                    <div class="bg-white rounded-lg p-6 w-full max-w-md">
                        <h3 class="text-lg font-semibold mb-4">Add New Domain</h3>
                        ${this.renderAddDomainForm()}
                    </div>
                </div>
            </div>
        `;

        this.attachEventListeners();
    }

    renderEmptyState() {
        return `
            <div class="text-center py-12">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9 3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
                </svg>
                <h3 class="mt-2 text-sm font-medium text-gray-900">No domains configured</h3>
                <p class="mt-1 text-sm text-gray-500">Get started by adding your first domain.</p>
                <div class="mt-6">
                    <button onclick="document.getElementById('add-domain-modal').classList.remove('hidden')" 
                            class="btn-primary">
                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                        </svg>
                        Add Domain
                    </button>
                </div>
            </div>
        `;
    }

    renderDomainsTable() {
        return `
            <div class="px-6 py-4 border-b border-gray-200">
                <div class="flex justify-between items-center">
                    <h3 class="text-lg font-semibold text-gray-900">Configured Domains (${this.domains.length})</h3>
                    <button onclick="document.getElementById('add-domain-modal').classList.remove('hidden')" 
                            class="btn-primary">
                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                        </svg>
                        Add Domain
                    </button>
                </div>
            </div>
            
            <div class="overflow-x-auto">
                <table class="min-w-full divide-y divide-gray-200">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Domain</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Health Checks</th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Health Score</th>
                            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">
                        ${this.domains.map(domain => this.renderDomainRow(domain)).join('')}
                    </tbody>
                </table>
            </div>
        `;
    }

    renderDomainRow(domain) {
        const actionBadgeClass = {
            'store': 'bg-green-100 text-green-800',
            'forward': 'bg-blue-100 text-blue-800',
            'discard': 'bg-gray-100 text-gray-800',
            'bounce': 'bg-red-100 text-red-800'
        }[domain.action] || 'bg-gray-100 text-gray-800';

        return `
            <tr class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="flex items-center">
                        <div>
                            <div class="text-sm font-medium text-gray-900">${domain.domain}</div>
                            ${domain.forward_to && domain.forward_to.length > 0 ? `
                                <div class="text-sm text-gray-500">Forwards to: ${domain.forward_to.join(', ')}</div>
                            ` : ''}
                        </div>
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full ${actionBadgeClass}">
                        ${domain.action}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="text-sm ${domain.health_checks ? 'text-green-600' : 'text-gray-400'}">
                        ${domain.health_checks ? 'Enabled' : 'Disabled'}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <div id="health-score-${domain.domain}" class="text-sm text-gray-500">
                        <button onclick="this.loadHealthScore('${domain.domain}')" 
                                class="text-gomail-600 hover:text-gomail-700 text-sm">
                            Check Health
                        </button>
                    </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <a href="/domains/${domain.domain}/health" 
                       class="text-gomail-600 hover:text-gomail-700">View Health</a>
                    <button onclick="this.editDomain('${domain.domain}')" 
                            class="text-indigo-600 hover:text-indigo-700">Edit</button>
                    <button onclick="this.deleteDomain('${domain.domain}')" 
                            class="text-red-600 hover:text-red-700">Delete</button>
                </td>
            </tr>
        `;
    }

    renderAddDomainForm() {
        return `
            <form id="add-domain-form" class="space-y-4">
                <div>
                    <label for="domain-name" class="form-label">Domain Name</label>
                    <input type="text" id="domain-name" name="domain" placeholder="example.com" 
                           class="form-input" required>
                </div>
                
                <div>
                    <label for="domain-action" class="form-label">Action</label>
                    <select id="domain-action" name="action" class="form-input" required>
                        <option value="store">Store emails</option>
                        <option value="forward">Forward emails</option>
                        <option value="discard">Discard emails</option>
                        <option value="bounce">Bounce emails</option>
                    </select>
                </div>
                
                <div id="forward-to-section" class="hidden">
                    <label for="forward-to" class="form-label">Forward To</label>
                    <input type="email" id="forward-to" name="forward_to" 
                           placeholder="user@example.com" class="form-input">
                    <p class="text-sm text-gray-500 mt-1">Enter email address to forward to</p>
                </div>
                
                <div id="bounce-message-section" class="hidden">
                    <label for="bounce-message" class="form-label">Bounce Message</label>
                    <textarea id="bounce-message" name="bounce_message" rows="3" 
                              placeholder="Email delivery failed..." class="form-input"></textarea>
                </div>
                
                <div class="flex items-center">
                    <input type="checkbox" id="health-checks" name="health_checks" 
                           class="h-4 w-4 text-gomail-600 border-gray-300 rounded" checked>
                    <label for="health-checks" class="ml-2 text-sm text-gray-700">
                        Enable health monitoring
                    </label>
                </div>
                
                <div class="flex justify-end space-x-3 pt-4">
                    <button type="button" onclick="document.getElementById('add-domain-modal').classList.add('hidden')" 
                            class="btn-secondary">Cancel</button>
                    <button type="submit" class="btn-primary">Add Domain</button>
                </div>
            </form>
        `;
    }

    attachEventListeners() {
        // Action selector change handler
        const actionSelect = document.getElementById('domain-action');
        if (actionSelect) {
            actionSelect.addEventListener('change', this.handleActionChange.bind(this));
        }

        // Form submission handler
        const addDomainForm = document.getElementById('add-domain-form');
        if (addDomainForm) {
            addDomainForm.addEventListener('submit', this.handleAddDomain.bind(this));
        }

        // Attach this instance to the global scope for onclick handlers
        window.domainManager = this;
    }

    handleActionChange(event) {
        const action = event.target.value;
        const forwardSection = document.getElementById('forward-to-section');
        const bounceSection = document.getElementById('bounce-message-section');
        
        forwardSection.classList.add('hidden');
        bounceSection.classList.add('hidden');
        
        if (action === 'forward') {
            forwardSection.classList.remove('hidden');
        } else if (action === 'bounce') {
            bounceSection.classList.remove('hidden');
        }
    }

    async handleAddDomain(event) {
        event.preventDefault();
        
        const formData = new FormData(event.target);
        const domainData = {
            domain: formData.get('domain'),
            action: formData.get('action'),
            health_checks: formData.has('health_checks')
        };
        
        if (domainData.action === 'forward') {
            domainData.forward_to = [formData.get('forward_to')];
        } else if (domainData.action === 'bounce') {
            domainData.bounce_message = formData.get('bounce_message');
        }
        
        try {
            await window.api.createDomain(domainData);
            window.app.showNotification('Domain added successfully', 'success');
            
            // Close modal and refresh
            document.getElementById('add-domain-modal').classList.add('hidden');
            this.refreshDomainsList();
            
        } catch (error) {
            console.error('Failed to add domain:', error);
            window.app.showNotification('Failed to add domain: ' + error.message, 'error');
        }
    }

    async loadHealthScore(domain) {
        const scoreElement = document.getElementById(`health-score-${domain}`);
        if (!scoreElement) return;
        
        scoreElement.innerHTML = '<div class="animate-spin rounded-full h-4 w-4 border-b-2 border-gomail-600"></div>';
        
        try {
            const health = await window.api.getDomainHealth(domain);
            const overallStatus = this.getHealthStatus(health.overall_score);
            
            scoreElement.innerHTML = `
                <div class="flex items-center space-x-2">
                    <span class="health-score-${overallStatus.class} text-sm">${health.overall_score}/100</span>
                    <span class="status-${overallStatus.status} text-xs">${overallStatus.text}</span>
                </div>
            `;
        } catch (error) {
            console.error('Failed to load health score:', error);
            scoreElement.innerHTML = '<span class="text-red-600 text-sm">Error</span>';
        }
    }

    async editDomain(domain) {
        try {
            const domainData = await window.api.getDomain(domain);
            // TODO: Implement edit domain modal
            window.app.showNotification('Edit domain functionality coming soon', 'info');
        } catch (error) {
            console.error('Failed to load domain data:', error);
            window.app.showNotification('Failed to load domain data', 'error');
        }
    }

    async deleteDomain(domain) {
        if (!confirm(`Are you sure you want to delete domain "${domain}"? This action cannot be undone.`)) {
            return;
        }
        
        try {
            await window.api.deleteDomain(domain);
            window.app.showNotification('Domain deleted successfully', 'success');
            this.refreshDomainsList();
        } catch (error) {
            console.error('Failed to delete domain:', error);
            window.app.showNotification('Failed to delete domain: ' + error.message, 'error');
        }
    }

    async refreshDomainsList() {
        try {
            const domainsData = await window.api.getDomains();
            this.render(domainsData);
        } catch (error) {
            console.error('Failed to refresh domains list:', error);
        }
    }

    getHealthStatus(score) {
        if (score >= 80) {
            return { status: 'healthy', class: 'good', text: 'Excellent' };
        } else if (score >= 60) {
            return { status: 'warning', class: 'fair', text: 'Good' };
        } else if (score >= 40) {
            return { status: 'warning', class: 'fair', text: 'Fair' };
        } else {
            return { status: 'error', class: 'poor', text: 'Poor' };
        }
    }
}

// Make it globally available
window.DomainManager = DomainManager;
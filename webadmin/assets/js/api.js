// API client for communicating with the GoMail backend

class APIClient {
    constructor() {
        this.baseURL = '/api';
        this.bearerToken = null;
    }

    setToken(token) {
        this.bearerToken = token;
    }

    getToken() {
        return this.bearerToken || window.app?.bearerToken;
    }

    async request(method, endpoint, data = null) {
        const url = `${this.baseURL}${endpoint}`;
        const options = {
            method,
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${this.getToken()}`
            }
        };

        if (data) {
            options.body = JSON.stringify(data);
        }

        try {
            const response = await fetch(url, options);
            
            if (!response.ok) {
                if (response.status === 401) {
                    throw new Error('Authentication failed');
                }
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }

            // Handle empty responses
            const contentType = response.headers.get('content-type');
            if (contentType && contentType.includes('application/json')) {
                return await response.json();
            } else {
                return await response.text();
            }
        } catch (error) {
            console.error(`API request failed: ${method} ${url}`, error);
            throw error;
        }
    }

    // System Health
    async getHealth() {
        return this.request('GET', '/health');
    }

    // Domain Management
    async getDomains() {
        return this.request('GET', '/domains');
    }

    async getDomain(domain) {
        return this.request('GET', `/domains/${encodeURIComponent(domain)}`);
    }

    async createDomain(domainData) {
        return this.request('POST', '/domains', domainData);
    }

    async updateDomain(domain, domainData) {
        return this.request('PUT', `/domains/${encodeURIComponent(domain)}`, domainData);
    }

    async deleteDomain(domain) {
        return this.request('DELETE', `/domains/${encodeURIComponent(domain)}`);
    }

    // Domain Health
    async getDomainHealth(domain) {
        return this.request('GET', `/domains/${encodeURIComponent(domain)}/health`);
    }

    async refreshDomainHealth(domain) {
        return this.request('POST', `/domains/${encodeURIComponent(domain)}/health/refresh`);
    }

    // Email Management
    async getEmails(params = {}) {
        const queryString = new URLSearchParams(params).toString();
        const endpoint = queryString ? `/emails?${queryString}` : '/emails';
        return this.request('GET', endpoint);
    }

    async getEmail(id) {
        return this.request('GET', `/emails/${encodeURIComponent(id)}`);
    }

    async deleteEmail(id) {
        return this.request('DELETE', `/emails/${encodeURIComponent(id)}`);
    }

    async getEmailRaw(id) {
        const url = `${this.baseURL}/emails/${encodeURIComponent(id)}/raw`;
        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${this.getToken()}`
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
        
        return response.blob();
    }

    // Routing Rules
    async getRoutingRules() {
        return this.request('GET', '/routing/rules');
    }

    async createRoutingRule(ruleData) {
        return this.request('POST', '/routing/rules', ruleData);
    }

    async updateRoutingRule(id, ruleData) {
        return this.request('PUT', `/routing/rules/${encodeURIComponent(id)}`, ruleData);
    }

    async deleteRoutingRule(id) {
        return this.request('DELETE', `/routing/rules/${encodeURIComponent(id)}`);
    }

    // Utility Methods
    downloadEmailRaw(id, filename) {
        this.getEmailRaw(id).then(blob => {
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.style.display = 'none';
            a.href = url;
            a.download = filename || `email-${id}.eml`;
            document.body.appendChild(a);
            a.click();
            window.URL.revokeObjectURL(url);
            document.body.removeChild(a);
        }).catch(error => {
            console.error('Failed to download email:', error);
            window.app.showNotification('Failed to download email', 'error');
        });
    }

    async testConnection() {
        try {
            await this.getHealth();
            return true;
        } catch (error) {
            return false;
        }
    }
}
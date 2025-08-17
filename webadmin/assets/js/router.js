// SPA Router for navigation without hash URLs

class Router {
    constructor() {
        this.routes = new Map();
        this.currentRoute = null;
        this.contentElement = document.getElementById('content');
        
        // Define routes
        this.defineRoutes();
        
        // Handle browser navigation
        window.addEventListener('popstate', () => {
            this.navigate(window.location.pathname, false);
        });
        
        // Handle navigation clicks
        document.addEventListener('click', (e) => {
            if (e.target.matches('[data-route]') || e.target.closest('[data-route]')) {
                e.preventDefault();
                const link = e.target.matches('[data-route]') ? e.target : e.target.closest('[data-route]');
                this.navigate(link.getAttribute('data-route') || link.href);
            }
        });
    }

    defineRoutes() {
        this.routes.set('/', {
            title: 'Dashboard',
            component: this.renderDashboard
        });
        
        this.routes.set('/emails', {
            title: 'Emails',
            component: this.renderEmails
        });
        
        this.routes.set('/domains', {
            title: 'Domains',
            component: this.renderDomains
        });
        
        this.routes.set('/domains/:domain', {
            title: 'Domain Details',
            component: this.renderDomainDetails
        });
        
        this.routes.set('/domains/:domain/health', {
            title: 'Domain Health',
            component: this.renderDomainHealth
        });
        
        this.routes.set('/routing', {
            title: 'Routing Rules',
            component: this.renderRouting
        });
        
        this.routes.set('/settings', {
            title: 'Settings',
            component: this.renderSettings
        });
    }

    start() {
        this.navigate(window.location.pathname, false);
    }

    navigate(path, pushState = true) {
        const route = this.matchRoute(path);
        
        if (!route) {
            this.navigate('/', true);
            return;
        }
        
        this.currentRoute = { path, route, params: this.extractParams(path, route.pattern) };
        
        if (pushState) {
            history.pushState(null, '', path);
        }
        
        // Update document title
        document.title = `${route.config.title} - GoMail Admin`;
        
        // Update navigation
        this.updateNavigation(path);
        
        // Render the component
        this.renderComponent(route.config.component);
    }

    matchRoute(path) {
        for (const [pattern, config] of this.routes) {
            const regex = this.patternToRegex(pattern);
            if (regex.test(path)) {
                return { pattern, config };
            }
        }
        return null;
    }

    patternToRegex(pattern) {
        const regexPattern = pattern
            .replace(/:[^/]+/g, '([^/]+)')
            .replace(/\//g, '\\/');
        return new RegExp(`^${regexPattern}$`);
    }

    extractParams(path, pattern) {
        const regex = this.patternToRegex(pattern);
        const matches = path.match(regex);
        const params = {};
        
        if (matches) {
            const paramNames = pattern.match(/:[^/]+/g) || [];
            paramNames.forEach((param, index) => {
                const paramName = param.substring(1);
                params[paramName] = decodeURIComponent(matches[index + 1]);
            });
        }
        
        return params;
    }

    updateNavigation(currentPath) {
        // Remove active class from all nav links
        document.querySelectorAll('.nav-link, .nav-link-active').forEach(link => {
            link.className = 'nav-link';
        });
        
        // Add active class to current route
        const activeLink = document.querySelector(`[data-route="${currentPath}"]`) ||
                          document.querySelector(`[data-route="${currentPath.split('/')[1] ? '/' + currentPath.split('/')[1] : '/'}"]`);
        
        if (activeLink) {
            activeLink.className = 'nav-link-active';
        }
    }

    async renderComponent(componentFunction) {
        this.showLoading();
        
        try {
            const html = await componentFunction.call(this);
            this.contentElement.innerHTML = html;
            
            // Execute any component-specific JavaScript
            this.executeComponentScripts();
        } catch (error) {
            console.error('Failed to render component:', error);
            this.contentElement.innerHTML = `
                <div class="text-center py-8">
                    <div class="text-red-600 text-xl mb-2">Error</div>
                    <p class="text-gray-600">Failed to load page: ${error.message}</p>
                </div>
            `;
        }
    }

    showLoading() {
        this.contentElement.innerHTML = `
            <div class="text-center py-8">
                <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-gomail-600 mx-auto"></div>
                <p class="mt-2 text-gray-600">Loading...</p>
            </div>
        `;
    }

    executeComponentScripts() {
        // Execute any JavaScript specific to the loaded component
        const scripts = this.contentElement.querySelectorAll('script');
        scripts.forEach(script => {
            const newScript = document.createElement('script');
            newScript.textContent = script.textContent;
            document.head.appendChild(newScript);
            document.head.removeChild(newScript);
        });
    }

    // Route Components
    async renderDashboard() {
        const health = await window.api.getHealth();
        const domains = await window.api.getDomains();
        const emails = await window.api.getEmails({ limit: 5 });
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>
                    <div class="text-sm text-gray-500">
                        Last updated: ${new Date().toLocaleTimeString()}
                    </div>
                </div>
                
                <!-- System Health Cards -->
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                    <div class="card">
                        <div class="card-body">
                            <div class="flex items-center justify-between">
                                <div>
                                    <p class="text-sm font-medium text-gray-600">System Status</p>
                                    <p class="text-2xl font-bold text-green-600">${health.status}</p>
                                </div>
                                <div class="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                                    <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                                    </svg>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-body">
                            <div class="flex items-center justify-between">
                                <div>
                                    <p class="text-sm font-medium text-gray-600">Total Domains</p>
                                    <p class="text-2xl font-bold text-gray-900">${domains.total || 0}</p>
                                </div>
                                <div class="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
                                    <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9 3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
                                    </svg>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-body">
                            <div class="flex items-center justify-between">
                                <div>
                                    <p class="text-sm font-medium text-gray-600">Recent Emails</p>
                                    <p class="text-2xl font-bold text-gray-900">${emails.total || 0}</p>
                                </div>
                                <div class="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
                                    <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                                    </svg>
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="card">
                        <div class="card-body">
                            <div class="flex items-center justify-between">
                                <div>
                                    <p class="text-sm font-medium text-gray-600">Uptime</p>
                                    <p class="text-2xl font-bold text-gray-900">99.9%</p>
                                </div>
                                <div class="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
                                    <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                                    </svg>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <!-- Quick Actions -->
                <div class="card">
                    <div class="card-header">
                        <h2 class="text-lg font-semibold">Quick Actions</h2>
                    </div>
                    <div class="card-body">
                        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
                            <a href="/domains" class="block p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                                <div class="flex items-center space-x-3">
                                    <svg class="w-6 h-6 text-gomail-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                                    </svg>
                                    <span class="font-medium">Add Domain</span>
                                </div>
                            </a>
                            
                            <a href="/emails" class="block p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                                <div class="flex items-center space-x-3">
                                    <svg class="w-6 h-6 text-gomail-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path>
                                    </svg>
                                    <span class="font-medium">View Emails</span>
                                </div>
                            </a>
                            
                            <a href="/routing" class="block p-4 border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors">
                                <div class="flex items-center space-x-3">
                                    <svg class="w-6 h-6 text-gomail-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                                    </svg>
                                    <span class="font-medium">Configure Routing</span>
                                </div>
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        `;
    }

    async renderEmails() {
        const emails = await window.api.getEmails();
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">Emails</h1>
                    <div class="flex space-x-2">
                        <input type="search" placeholder="Search emails..." class="form-input">
                        <button class="btn-primary">Search</button>
                    </div>
                </div>
                
                <div class="card">
                    <div class="card-body">
                        <p class="text-gray-600">Email management interface will be implemented here.</p>
                        <p class="text-sm text-gray-500 mt-2">Total emails: ${emails.total || 0}</p>
                    </div>
                </div>
            </div>
        `;
    }

    async renderDomains() {
        const domains = await window.api.getDomains();
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">Domains</h1>
                    <button class="btn-primary">Add Domain</button>
                </div>
                
                <div class="card">
                    <div class="card-body">
                        <div id="domain-manager"></div>
                    </div>
                </div>
            </div>
            
            <script>
                if (window.DomainManager) {
                    const domainManager = new DomainManager(document.getElementById('domain-manager'));
                    domainManager.render(${JSON.stringify(domains)});
                }
            </script>
        `;
    }

    async renderDomainDetails() {
        const domain = this.currentRoute.params.domain;
        const domainData = await window.api.getDomain(domain);
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">${domain}</h1>
                    <div class="flex space-x-2">
                        <a href="/domains/${domain}/health" class="btn-secondary">View Health</a>
                        <button class="btn-primary">Edit Domain</button>
                    </div>
                </div>
                
                <div class="card">
                    <div class="card-body">
                        <p class="text-gray-600">Domain details for ${domain}</p>
                        <pre class="mt-4 text-sm bg-gray-100 p-4 rounded">${JSON.stringify(domainData, null, 2)}</pre>
                    </div>
                </div>
            </div>
        `;
    }

    async renderDomainHealth() {
        const domain = this.currentRoute.params.domain;
        const health = await window.api.getDomainHealth(domain);
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">${domain} Health</h1>
                    <button onclick="window.api.refreshDomainHealth('${domain}').then(() => window.router.navigate(window.location.pathname, false))" 
                            class="btn-primary">Refresh</button>
                </div>
                
                <div id="health-dashboard"></div>
            </div>
            
            <script>
                if (window.HealthDashboard) {
                    const healthDashboard = new HealthDashboard(document.getElementById('health-dashboard'));
                    healthDashboard.render(${JSON.stringify(health)});
                }
            </script>
        `;
    }

    async renderRouting() {
        const rules = await window.api.getRoutingRules();
        
        return `
            <div class="space-y-6">
                <div class="flex justify-between items-center">
                    <h1 class="text-2xl font-bold text-gray-900">Routing Rules</h1>
                    <button class="btn-primary">Add Rule</button>
                </div>
                
                <div class="card">
                    <div class="card-body">
                        <div id="routing-rules"></div>
                    </div>
                </div>
            </div>
            
            <script>
                if (window.RoutingRules) {
                    const routingRules = new RoutingRules(document.getElementById('routing-rules'));
                    routingRules.render(${JSON.stringify(rules)});
                }
            </script>
        `;
    }

    async renderSettings() {
        return `
            <div class="space-y-6">
                <h1 class="text-2xl font-bold text-gray-900">Settings</h1>
                
                <div class="card">
                    <div class="card-header">
                        <h2 class="text-lg font-semibold">System Configuration</h2>
                    </div>
                    <div class="card-body">
                        <p class="text-gray-600">System settings interface will be implemented here.</p>
                    </div>
                </div>
            </div>
        `;
    }
}
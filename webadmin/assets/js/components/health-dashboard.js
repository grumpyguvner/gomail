// Health Dashboard Component for domain health visualization

class HealthDashboard {
    constructor(container) {
        this.container = container;
    }

    render(healthData) {
        if (!healthData) {
            this.container.innerHTML = '<p class="text-gray-500">No health data available</p>';
            return;
        }

        const overallStatus = this.getOverallStatus(healthData.overall_score);
        
        this.container.innerHTML = `
            <div class="space-y-6">
                <!-- Overall Health Score -->
                <div class="card">
                    <div class="card-body">
                        <div class="flex items-center justify-between">
                            <div>
                                <h3 class="text-lg font-semibold text-gray-900">Overall Health Score</h3>
                                <p class="text-sm text-gray-600">Last checked: ${this.formatDate(healthData.last_checked)}</p>
                            </div>
                            <div class="text-center">
                                <div class="health-score-${overallStatus.class}">${healthData.overall_score}/100</div>
                                <span class="status-${overallStatus.status}">${overallStatus.text}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Health Categories Grid -->
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                    ${this.renderDNSHealth(healthData.dns)}
                    ${this.renderSPFHealth(healthData.spf)}
                    ${this.renderDKIMHealth(healthData.dkim)}
                    ${this.renderDMARCHealth(healthData.dmarc)}
                    ${this.renderSSLHealth(healthData.ssl)}
                    ${this.renderDeliverabilityHealth(healthData.deliverability)}
                </div>

                <!-- Issues Summary -->
                ${this.renderIssuesSummary(healthData)}
            </div>
        `;
    }

    renderDNSHealth(dns) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">DNS Records</h4>
                        <span class="status-${dns.status}">${dns.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${dns.score}/100</span>
                        </div>
                        
                        <div>
                            <p class="text-sm font-medium text-gray-700">A Records:</p>
                            <div class="text-sm text-gray-600">
                                ${dns.a_records.length > 0 ? 
                                    dns.a_records.map(record => `<div class="font-mono">${record}</div>`).join('') :
                                    '<div class="text-red-600">None found</div>'
                                }
                            </div>
                        </div>
                        
                        <div>
                            <p class="text-sm font-medium text-gray-700">MX Records:</p>
                            <div class="text-sm text-gray-600">
                                ${dns.mx_records.length > 0 ? 
                                    dns.mx_records.map(record => `<div class="font-mono">${record}</div>`).join('') :
                                    '<div class="text-red-600">None found</div>'
                                }
                            </div>
                        </div>
                        
                        ${dns.ptr_record ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">PTR Record:</p>
                                <div class="text-sm text-gray-600 font-mono">${dns.ptr_record}</div>
                            </div>
                        ` : ''}
                        
                        ${this.renderIssues(dns.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderSPFHealth(spf) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">SPF Record</h4>
                        <span class="status-${spf.status}">${spf.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${spf.score}/100</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Valid:</span>
                            <span class="font-semibold ${spf.valid ? 'text-green-600' : 'text-red-600'}">
                                ${spf.valid ? 'Yes' : 'No'}
                            </span>
                        </div>
                        
                        ${spf.record ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">Record:</p>
                                <div class="text-xs text-gray-600 font-mono bg-gray-50 p-2 rounded break-all">
                                    ${spf.record}
                                </div>
                            </div>
                        ` : ''}
                        
                        ${spf.includes && spf.includes.length > 0 ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">Includes:</p>
                                <div class="text-sm text-gray-600">
                                    ${spf.includes.map(include => `<div class="font-mono">${include}</div>`).join('')}
                                </div>
                            </div>
                        ` : ''}
                        
                        ${this.renderIssues(spf.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderDKIMHealth(dkim) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">DKIM Records</h4>
                        <span class="status-${dkim.status}">${dkim.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${dkim.score}/100</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Selectors Found:</span>
                            <span class="font-semibold">${dkim.selectors.length}</span>
                        </div>
                        
                        ${dkim.selectors.length > 0 ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">Selectors:</p>
                                <div class="space-y-2">
                                    ${dkim.selectors.map(selector => `
                                        <div class="text-sm border border-gray-200 rounded p-2">
                                            <div class="flex justify-between items-center">
                                                <span class="font-mono font-semibold">${selector.selector}</span>
                                                <span class="text-xs ${selector.valid ? 'text-green-600' : 'text-red-600'}">
                                                    ${selector.valid ? 'Valid' : 'Invalid'}
                                                </span>
                                            </div>
                                            ${selector.key_type ? `
                                                <div class="text-xs text-gray-600 mt-1">
                                                    ${selector.key_type.toUpperCase()} ${selector.key_size} bits
                                                </div>
                                            ` : ''}
                                        </div>
                                    `).join('')}
                                </div>
                            </div>
                        ` : ''}
                        
                        ${this.renderIssues(dkim.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderDMARCHealth(dmarc) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">DMARC Policy</h4>
                        <span class="status-${dmarc.status}">${dmarc.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${dmarc.score}/100</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Valid:</span>
                            <span class="font-semibold ${dmarc.valid ? 'text-green-600' : 'text-red-600'}">
                                ${dmarc.valid ? 'Yes' : 'No'}
                            </span>
                        </div>
                        
                        ${dmarc.policy ? `
                            <div class="flex justify-between">
                                <span class="text-sm text-gray-600">Policy:</span>
                                <span class="font-semibold font-mono">${dmarc.policy}</span>
                            </div>
                        ` : ''}
                        
                        ${dmarc.percent !== undefined ? `
                            <div class="flex justify-between">
                                <span class="text-sm text-gray-600">Coverage:</span>
                                <span class="font-semibold">${dmarc.percent}%</span>
                            </div>
                        ` : ''}
                        
                        ${dmarc.record ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">Record:</p>
                                <div class="text-xs text-gray-600 font-mono bg-gray-50 p-2 rounded break-all">
                                    ${dmarc.record}
                                </div>
                            </div>
                        ` : ''}
                        
                        ${this.renderIssues(dmarc.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderSSLHealth(ssl) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">SSL Certificate</h4>
                        <span class="status-${ssl.status}">${ssl.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${ssl.score}/100</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Valid:</span>
                            <span class="font-semibold ${ssl.valid ? 'text-green-600' : 'text-red-600'}">
                                ${ssl.valid ? 'Yes' : 'No'}
                            </span>
                        </div>
                        
                        ${ssl.valid ? `
                            <div class="flex justify-between">
                                <span class="text-sm text-gray-600">Days Left:</span>
                                <span class="font-semibold ${ssl.days_left < 30 ? 'text-red-600' : ssl.days_left < 60 ? 'text-yellow-600' : 'text-green-600'}">
                                    ${ssl.days_left}
                                </span>
                            </div>
                            
                            ${ssl.expiry ? `
                                <div>
                                    <p class="text-sm font-medium text-gray-700">Expires:</p>
                                    <div class="text-sm text-gray-600">${this.formatDate(ssl.expiry)}</div>
                                </div>
                            ` : ''}
                            
                            ${ssl.issuer ? `
                                <div>
                                    <p class="text-sm font-medium text-gray-700">Issuer:</p>
                                    <div class="text-sm text-gray-600">${ssl.issuer}</div>
                                </div>
                            ` : ''}
                        ` : ''}
                        
                        ${this.renderIssues(ssl.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderDeliverabilityHealth(deliverability) {
        return `
            <div class="card">
                <div class="card-header">
                    <div class="flex items-center justify-between">
                        <h4 class="font-semibold">Deliverability</h4>
                        <span class="status-${deliverability.status}">${deliverability.status}</span>
                    </div>
                </div>
                <div class="card-body">
                    <div class="space-y-3">
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Score:</span>
                            <span class="font-semibold">${deliverability.score}/100</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Reputation:</span>
                            <span class="font-semibold capitalize">${deliverability.reputation}</span>
                        </div>
                        
                        <div class="flex justify-between">
                            <span class="text-sm text-gray-600">Blacklisted:</span>
                            <span class="font-semibold ${deliverability.blacklisted ? 'text-red-600' : 'text-green-600'}">
                                ${deliverability.blacklisted ? 'Yes' : 'No'}
                            </span>
                        </div>
                        
                        ${deliverability.blacklists && deliverability.blacklists.length > 0 ? `
                            <div>
                                <p class="text-sm font-medium text-gray-700">Blacklists:</p>
                                <div class="text-sm text-gray-600">
                                    ${deliverability.blacklists.map(bl => `<div class="font-mono text-red-600">${bl}</div>`).join('')}
                                </div>
                            </div>
                        ` : ''}
                        
                        ${this.renderIssues(deliverability.issues)}
                    </div>
                </div>
            </div>
        `;
    }

    renderIssuesSummary(healthData) {
        const allIssues = [
            ...healthData.dns.issues,
            ...healthData.spf.issues,
            ...healthData.dkim.issues,
            ...healthData.dmarc.issues,
            ...healthData.ssl.issues,
            ...healthData.deliverability.issues
        ];

        if (allIssues.length === 0) {
            return `
                <div class="card">
                    <div class="card-body">
                        <div class="flex items-center space-x-3">
                            <svg class="w-8 h-8 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                            </svg>
                            <div>
                                <h3 class="text-lg font-semibold text-green-600">All Checks Passed!</h3>
                                <p class="text-sm text-gray-600">Your domain configuration looks excellent.</p>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }

        return `
            <div class="card">
                <div class="card-header">
                    <h3 class="text-lg font-semibold text-red-600">Issues Found (${allIssues.length})</h3>
                </div>
                <div class="card-body">
                    <div class="space-y-2">
                        ${allIssues.map(issue => `
                            <div class="flex items-start space-x-2 p-3 bg-red-50 border border-red-200 rounded">
                                <svg class="w-5 h-5 text-red-600 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                                </svg>
                                <span class="text-sm text-red-800">${issue}</span>
                            </div>
                        `).join('')}
                    </div>
                </div>
            </div>
        `;
    }

    renderIssues(issues) {
        if (!issues || issues.length === 0) {
            return '';
        }

        return `
            <div>
                <p class="text-sm font-medium text-red-700">Issues:</p>
                <div class="space-y-1">
                    ${issues.map(issue => `
                        <div class="text-xs text-red-600 bg-red-50 p-1 rounded">${issue}</div>
                    `).join('')}
                </div>
            </div>
        `;
    }

    getOverallStatus(score) {
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

    formatDate(dateString) {
        const date = new Date(dateString);
        return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
    }
}

// Make it globally available
window.HealthDashboard = HealthDashboard;
// Routing Rules Component for email routing configuration

class RoutingRules {
    constructor(container) {
        this.container = container;
        this.rules = [];
    }

    render(rulesData) {
        this.rules = rulesData.rules || [];
        
        this.container.innerHTML = `
            <div class="space-y-6">
                ${this.rules.length === 0 ? this.renderEmptyState() : this.renderRulesTable()}
            </div>
        `;
    }

    renderEmptyState() {
        return `
            <div class="text-center py-12">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path>
                </svg>
                <h3 class="mt-2 text-sm font-medium text-gray-900">No routing rules configured</h3>
                <p class="mt-1 text-sm text-gray-500">Create rules to control how emails are processed.</p>
                <div class="mt-6">
                    <button class="btn-primary">
                        <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                        </svg>
                        Add Routing Rule
                    </button>
                </div>
            </div>
        `;
    }

    renderRulesTable() {
        return `
            <div class="bg-white rounded-lg border border-gray-200">
                <div class="px-6 py-4 border-b border-gray-200">
                    <div class="flex justify-between items-center">
                        <h3 class="text-lg font-semibold text-gray-900">Routing Rules (${this.rules.length})</h3>
                        <button class="btn-primary">
                            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
                            </svg>
                            Add Rule
                        </button>
                    </div>
                </div>
                
                <div class="overflow-x-auto">
                    <table class="min-w-full divide-y divide-gray-200">
                        <thead class="bg-gray-50">
                            <tr>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Priority</th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Pattern</th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
                                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
                                <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                            </tr>
                        </thead>
                        <tbody class="bg-white divide-y divide-gray-200">
                            ${this.rules.map(rule => this.renderRuleRow(rule)).join('')}
                        </tbody>
                    </table>
                </div>
            </div>
        `;
    }

    renderRuleRow(rule) {
        return `
            <tr class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                    ${rule.priority || 'N/A'}
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <div class="text-sm font-mono text-gray-900">${rule.pattern || 'N/A'}</div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="inline-flex px-2 py-1 text-xs font-semibold rounded-full bg-blue-100 text-blue-800">
                        ${rule.action || 'N/A'}
                    </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap">
                    <span class="status-healthy">Active</span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <button class="text-indigo-600 hover:text-indigo-700">Edit</button>
                    <button class="text-red-600 hover:text-red-700">Delete</button>
                </td>
            </tr>
        `;
    }
}

// Make it globally available
window.RoutingRules = RoutingRules;
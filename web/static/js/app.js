let expenseChart;
let previewData = null;
let currentPage = 1;
let totalPages = 1;
let csrfToken = null;

document.addEventListener('DOMContentLoaded', async function() {
    await fetchCSRFToken();
    loadExpenses();
    setupFileUpload();
    setupAddExpenseForm();
    loadCategoryRules();
    loadDynamicCategories();
});

async function fetchCSRFToken() {
    try {
        const response = await fetch('/api/csrf-token');
        const data = await response.json();
        csrfToken = data.csrf_token;
    } catch (error) {
        console.error('Failed to fetch CSRF token:', error);
    }
}

async function apiRequest(url, options = {}) {
    // Ensure we have a CSRF token for non-GET requests
    if (options.method && options.method !== 'GET' && !csrfToken) {
        await fetchCSRFToken();
    }
    
    // Add CSRF token to headers for non-GET requests
    if (options.method && options.method !== 'GET' && csrfToken) {
        options.headers = {
            ...options.headers,
            'X-CSRF-Token': csrfToken
        };
    }
    
    const response = await fetch(url, options);
    
    // If we get a 403, it might be due to an invalid/expired CSRF token
    if (response.status === 403) {
        await fetchCSRFToken();
        if (options.method && options.method !== 'GET' && csrfToken) {
            options.headers = {
                ...options.headers,
                'X-CSRF-Token': csrfToken
            };
        }
        return fetch(url, options);
    }
    
    return response;
}

function setupFileUpload() {
    const fileInput = document.getElementById('csvFile');
    const uploadBtn = document.getElementById('uploadBtn');
    const fileName = document.getElementById('fileName');
    
    fileInput.addEventListener('change', function(e) {
        const file = e.target.files[0];
        if (file) {
            fileName.textContent = file.name;
            uploadBtn.disabled = false;
        } else {
            fileName.textContent = '';
            uploadBtn.disabled = true;
        }
    });
}

function setupAddExpenseForm() {
    const form = document.getElementById('addExpenseForm');
    const dateInput = document.getElementById('expenseDate');
    
    // Set default date to today
    dateInput.value = new Date().toISOString().split('T')[0];
    
    form.addEventListener('submit', async function(e) {
        e.preventDefault();
        await addExpense();
    });
}

async function addExpense() {
    const form = document.getElementById('addExpenseForm');
    const status = document.getElementById('addExpenseStatus');
    const submitBtn = form.querySelector('button[type="submit"]');
    
    // Get form data
    const expenseData = {
        date: document.getElementById('expenseDate').value,
        category: document.getElementById('expenseCategory').value,
        description: document.getElementById('expenseDescription').value,
        amount: parseFloat(document.getElementById('expenseAmount').value),
        vendor: document.getElementById('expenseVendor').value || '',
        payment_method: document.getElementById('expensePaymentMethod').value
    };
    
    // Validate required fields
    if (!expenseData.date || !expenseData.category || !expenseData.description || !expenseData.amount) {
        status.innerHTML = '<div class="error">Please fill in all required fields</div>';
        return;
    }
    
    submitBtn.disabled = true;
    status.innerHTML = '<div class="loading">Adding expense...</div>';
    
    try {
        const response = await apiRequest('/api/expenses', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(expenseData)
        });
        
        if (response.ok) {
            const newExpense = await response.json();
            status.innerHTML = '<div class="success">Expense added successfully!</div>';
            form.reset();
            document.getElementById('expenseDate').value = new Date().toISOString().split('T')[0];
            loadExpenses(); // Reload the expenses table
        } else {
            const error = await response.text();
            throw new Error(error);
        }
    } catch (error) {
        status.innerHTML = `<div class="error">Error: ${error.message}</div>`;
        console.error('Add expense error:', error);
    } finally {
        submitBtn.disabled = false;
        // Clear status after 5 seconds
        setTimeout(() => {
            status.innerHTML = '';
        }, 5000);
    }
}

async function uploadCSV() {
    const fileInput = document.getElementById('csvFile');
    const uploadStatus = document.getElementById('uploadStatus');
    const uploadBtn = document.getElementById('uploadBtn');
    
    if (!fileInput.files[0]) {
        uploadStatus.innerHTML = '<div class="error">Please select a CSV file</div>';
        return;
    }
    
    uploadBtn.disabled = true;
    uploadStatus.innerHTML = '<div class="loading">Parsing CSV...</div>';
    
    const formData = new FormData();
    formData.append('csv', fileInput.files[0]);
    
    try {
        const response = await apiRequest('/api/import/csv', {
            method: 'POST',
            body: formData
        });
        
        // Log the response for debugging
        console.log('Response status:', response.status);
        console.log('Response headers:', response.headers);
        
        const responseText = await response.text();
        console.log('Response text:', responseText);
        
        let result;
        try {
            result = JSON.parse(responseText);
        } catch (jsonError) {
            // If it's not JSON, treat the response text as an error message
            throw new Error(responseText);
        }
        
        if (response.ok) {
            previewData = result.expenses;
            displayPreview(result);
            uploadStatus.innerHTML = `<div class="success">${result.message}</div>`;
        } else {
            throw new Error(result.error || result.message || 'Upload failed');
        }
    } catch (error) {
        uploadStatus.innerHTML = `<div class="error">Error: ${error.message}</div>`;
        console.error('Upload error:', error);
    } finally {
        uploadBtn.disabled = false;
    }
}

function displayPreview(result) {
    const previewSection = document.getElementById('previewSection');
    const previewCount = document.getElementById('previewCount');
    const previewTable = document.getElementById('previewTable');
    const tbody = previewTable.querySelector('tbody');
    
    previewCount.textContent = `Found ${result.count} transactions`;
    tbody.innerHTML = '';
    
    result.expenses.forEach((expense, index) => {
        const row = document.createElement('tr');
        row.setAttribute('data-index', index);
        
        const amount = expense.amount < 0 ? 
            `<span class="negative">-$${Math.abs(expense.amount).toFixed(2)}</span>` :
            `$${expense.amount.toFixed(2)}`;
        
        // Format date to DD/MM/YYYY
        const date = new Date(expense.date);
        const formattedDate = `${date.getDate().toString().padStart(2, '0')}/${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getFullYear()}`;
            
        row.innerHTML = `
            <td class="date-cell">
                <span class="display-value">${formattedDate}</span>
                <input type="text" class="edit-input date-input" value="${formattedDate}" style="display: none;">
                <div class="date-error" style="display: none; color: red; font-size: 12px;"></div>
            </td>
            <td class="vendor-cell">
                <span class="display-value">${expense.vendor || '-'}</span>
                <input type="text" class="edit-input vendor-input" value="${expense.vendor || ''}" style="display: none;">
            </td>
            <td class="description-cell">
                <span class="display-value">${expense.description}</span>
                <input type="text" class="edit-input description-input" value="${expense.description}" style="display: none;">
            </td>
            <td class="category-cell">
                <span class="display-value">${expense.category}</span>
                <select class="edit-input category-select" style="display: none;">
                    <option value="Food & Dining" ${expense.category === 'Food & Dining' ? 'selected' : ''}>Food & Dining</option>
                    <option value="Transportation" ${expense.category === 'Transportation' ? 'selected' : ''}>Transportation</option>                  
                    <option value="Shopping" ${expense.category === 'Shopping' ? 'selected' : ''}>Shopping</option>
                    <option value="Utilities" ${expense.category === 'Utilities' ? 'selected' : ''}>Utilities</option>
                    <option value="Healthcare" ${expense.category === 'Healthcare' ? 'selected' : ''}>Healthcare</option>
                    <option value="Groceries" ${expense.category === 'Groceries' ? 'selected' : ''}>Groceries</option>
                    <option value="Splurge" ${expense.category === 'Splurge' ? 'selected' : ''}>Splurge</option>
                    <option value="Others" ${expense.category === 'Others' ? 'selected' : ''}>Others</option>
                </select>
            </td>
            <td class="amount-cell">${amount}</td>
            <td class="payment-method-cell">
                <span class="display-value">${expense.payment_method || 'CSV Import'}</span>
                <input type="text" class="edit-input payment-method-input" value="${expense.payment_method || 'CSV Import'}" style="display: none;">
            </td>
            <td class="actions-cell">
                <button class="edit-btn" onclick="editRow(${index})">Edit</button>
                <button class="save-btn" onclick="saveRow(${index})" style="display: none;">Save</button>
                <button class="cancel-btn" onclick="cancelEdit(${index})" style="display: none;">Cancel</button>
            </td>
        `;
        tbody.appendChild(row);
    });
    
    updateTotalDisplay();
    previewSection.style.display = 'block';
}

async function confirmImport() {
    // Check if any rows are currently being edited
    const editingRows = document.querySelectorAll('.save-btn[style*="inline-block"]');
    if (editingRows.length > 0) {
        alert('Please save or cancel all pending edits before confirming the import.');
        return;
    }
    
    const uploadStatus = document.getElementById('uploadStatus');
    const confirmBtn = document.querySelector('.confirm-btn');
    
    // Disable confirm button and show loading state
    confirmBtn.disabled = true;
    uploadStatus.innerHTML = '<div class="loading">Saving transactions to database...</div>';
    
    try {
        // Send the edited preview data to the backend
        const response = await apiRequest('/api/import/confirm', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(previewData)
        });
        
        const result = await response.json();
        
        if (response.ok) {
            uploadStatus.innerHTML = `<div class="success">${result.message} - ${result.count} transactions totaling $${result.total.toFixed(2)}</div>`;
            
            // Clear the preview data
            previewData = null;
            
            // Hide preview section after a delay
            setTimeout(() => {
                cancelImport();
                // Reload the main expense list to show new entries
                loadExpenses();
            }, 2000);
        } else {
            throw new Error(result.message || 'Failed to save transactions');
        }
    } catch (error) {
        uploadStatus.innerHTML = `<div class="error">Error saving transactions: ${error.message}</div>`;
        console.error('Import confirmation error:', error);
    } finally {
        confirmBtn.disabled = false;
    }
}

function cancelImport() {
    const previewSection = document.getElementById('previewSection');
    const fileInput = document.getElementById('csvFile');
    const fileName = document.getElementById('fileName');
    const uploadBtn = document.getElementById('uploadBtn');
    
    previewSection.style.display = 'none';
    fileInput.value = '';
    fileName.textContent = '';
    uploadBtn.disabled = true;
    previewData = null;
}

async function loadExpenses(page = 1) {
    const startDate = document.getElementById('startDate').value;
    const endDate = document.getElementById('endDate').value;
    const category = document.getElementById('category').value;
    
    const params = new URLSearchParams();
    if (startDate) params.append('start_date', startDate);
    if (endDate) params.append('end_date', endDate);
    if (category) params.append('category', category);
    params.append('page', page);
    params.append('limit', '20');
    
    try {
        // Load expenses
        const expensesResponse = await fetch(`/api/expenses?${params}`);
        const data = await expensesResponse.json();
        
        currentPage = page;
        if (data.pagination) {
            totalPages = Math.ceil(data.pagination.total / data.pagination.limit);
            displayExpenses(data.expenses);
            displayPagination(data.pagination);
        } else {
            // Fallback for old format
            displayExpenses(data);
        }
        
        // Load stats if date range is provided
        if (startDate && endDate) {
            const statsResponse = await fetch(`/api/expenses/stats?start_date=${startDate}&end_date=${endDate}`);
            const stats = await statsResponse.json();
            displayChart(stats);
        }
    } catch (error) {
        console.error('Error loading expenses:', error);
    }
}

function displayExpenses(expenses) {
    const tbody = document.querySelector('#expenseTable tbody');
    tbody.innerHTML = '';
    
    if (!expenses || expenses.length === 0) {
        const row = document.createElement('tr');
        row.innerHTML = '<td colspan="8" style="text-align: center; color: #666;">No expenses found</td>';
        tbody.appendChild(row);
        return;
    }
    
    expenses.forEach(expense => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${new Date(expense.date).toLocaleDateString()}</td>
            <td>${expense.category}</td>
            <td>${expense.description}</td>
            <td>${expense.amount < 0 ? `<span class="negative">-$${Math.abs(expense.amount).toFixed(2)}</span>` : `$${expense.amount.toFixed(2)}`}</td>
            <td>${expense.vendor || '-'}</td>
            <td>${expense.payment_method || '-'}</td>
            <td>
                <button class="edit-btn" onclick="editExpense(${expense.id})">Edit</button>
                <button class="delete-btn" onclick="deleteExpense(${expense.id})">Delete</button>
            </td>
        `;
        tbody.appendChild(row);
    });
}

function displayPagination(pagination) {
    const paginationDiv = document.getElementById('pagination');
    if (!pagination || pagination.total === 0) {
        paginationDiv.innerHTML = '';
        return;
    }
    
    const totalPages = Math.ceil(pagination.total / pagination.limit);
    let paginationHTML = `
        <div class="pagination-info">
            Showing ${Math.min((pagination.page - 1) * pagination.limit + 1, pagination.total)} to 
            ${Math.min(pagination.page * pagination.limit, pagination.total)} of ${pagination.total} expenses
        </div>
        <div class="pagination-controls">
    `;
    
    // Previous button
    if (pagination.has_previous) {
        paginationHTML += `<button onclick="loadExpenses(${pagination.page - 1})" class="page-btn">Previous</button>`;
    }
    
    // Page numbers
    const startPage = Math.max(1, pagination.page - 2);
    const endPage = Math.min(totalPages, pagination.page + 2);
    
    if (startPage > 1) {
        paginationHTML += `<button onclick="loadExpenses(1)" class="page-btn">1</button>`;
        if (startPage > 2) {
            paginationHTML += `<span class="page-ellipsis">...</span>`;
        }
    }
    
    for (let i = startPage; i <= endPage; i++) {
        const activeClass = i === pagination.page ? 'active' : '';
        paginationHTML += `<button onclick="loadExpenses(${i})" class="page-btn ${activeClass}">${i}</button>`;
    }
    
    if (endPage < totalPages) {
        if (endPage < totalPages - 1) {
            paginationHTML += `<span class="page-ellipsis">...</span>`;
        }
        paginationHTML += `<button onclick="loadExpenses(${totalPages})" class="page-btn">${totalPages}</button>`;
    }
    
    // Next button
    if (pagination.has_next) {
        paginationHTML += `<button onclick="loadExpenses(${pagination.page + 1})" class="page-btn">Next</button>`;
    }
    
    paginationHTML += '</div>';
    paginationDiv.innerHTML = paginationHTML;
}

async function editExpense(id) {
    const row = document.querySelector(`tr:has(button[onclick="editExpense(${id})"])`);
    if (!row) return;
    
    const cells = row.querySelectorAll('td');
    const originalData = {
        date: cells[0].textContent,
        category: cells[1].textContent,
        description: cells[2].textContent,
        amount: cells[3].textContent.replace('$', ''),
        vendor: cells[4].textContent === '-' ? '' : cells[4].textContent,
        payment_method: cells[5].textContent === '-' ? '' : cells[5].textContent
    };
    
    // Convert to edit mode
    const dateValue = new Date(originalData.date).toISOString().split('T')[0];
    
    cells[0].innerHTML = `<input type="date" value="${dateValue}" class="edit-input">`;
    cells[1].innerHTML = `
        <select class="edit-input">
            <option value="Food & Dining" ${originalData.category === 'Food & Dining' ? 'selected' : ''}>Food & Dining</option>
            <option value="Food Delivery" ${originalData.category === 'Food Delivery' ? 'selected' : ''}>Food Delivery</option>
            <option value="Transportation" ${originalData.category === 'Transportation' ? 'selected' : ''}>Transportation</option>
            <option value="Shopping" ${originalData.category === 'Shopping' ? 'selected' : ''}>Shopping</option>
            <option value="Utilities" ${originalData.category === 'Utilities' ? 'selected' : ''}>Utilities</option>
            <option value="Mobile & Telecom" ${originalData.category === 'Mobile & Telecom' ? 'selected' : ''}>Mobile & Telecom</option>
            <option value="Healthcare" ${originalData.category === 'Healthcare' ? 'selected' : ''}>Healthcare</option>
            <option value="Other" ${originalData.category === 'Other' ? 'selected' : ''}>Other</option>
        </select>
    `;
    cells[2].innerHTML = `<input type="text" value="${originalData.description}" class="edit-input">`;
    cells[3].innerHTML = `<input type="number" value="${originalData.amount}" step="0.01" class="edit-input">`;
    cells[4].innerHTML = `<input type="text" value="${originalData.vendor}" class="edit-input">`;
    cells[5].innerHTML = `
        <select class="edit-input">
            <option value="Credit Card" ${originalData.payment_method === 'Credit Card' ? 'selected' : ''}>Credit Card</option>
            <option value="Debit Card" ${originalData.payment_method === 'Debit Card' ? 'selected' : ''}>Debit Card</option>
            <option value="Cash" ${originalData.payment_method === 'Cash' ? 'selected' : ''}>Cash</option>
            <option value="Bank Transfer" ${originalData.payment_method === 'Bank Transfer' ? 'selected' : ''}>Bank Transfer</option>
            <option value="Other" ${originalData.payment_method === 'Other' ? 'selected' : ''}>Other</option>
        </select>
    `;
    cells[6].innerHTML = `
        <button class="save-btn" onclick="saveExpense(${id})">Save</button>
        <button class="cancel-btn" onclick="cancelEdit(${id}, '${JSON.stringify(originalData).replace(/'/g, "\\'")}')">Cancel</button>
    `;
}

async function saveExpense(id) {
    const row = document.querySelector(`tr:has(button[onclick="saveExpense(${id})"])`);
    if (!row) return;
    
    const cells = row.querySelectorAll('td');
    const expenseData = {
        date: cells[0].querySelector('input').value,
        category: cells[1].querySelector('select').value,
        description: cells[2].querySelector('input').value,
        amount: parseFloat(cells[3].querySelector('input').value),
        vendor: cells[4].querySelector('input').value || '',
        payment_method: cells[5].querySelector('select').value
    };
    
    // Validate required fields
    if (!expenseData.date || !expenseData.category || !expenseData.description || !expenseData.amount) {
        alert('Please fill in all required fields');
        return;
    }
    
    try {
        const response = await apiRequest(`/api/expenses/${id}`, {
            method: 'PUT',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(expenseData)
        });
        
        if (response.ok) {
            loadExpenses(currentPage); // Reload the current page
        } else {
            const error = await response.text();
            alert(`Error updating expense: ${error}`);
        }
    } catch (error) {
        console.error('Update expense error:', error);
        alert(`Error updating expense: ${error.message}`);
    }
}

function cancelEdit(id, originalDataStr) {
    const originalData = JSON.parse(originalDataStr);
    const row = document.querySelector(`tr:has(button[onclick*="cancelEdit(${id}"])`);
    if (!row) return;
    
    const cells = row.querySelectorAll('td');
    
    // Restore original values
    cells[0].textContent = originalData.date;
    cells[1].textContent = originalData.category;
    cells[2].textContent = originalData.description;
    cells[3].textContent = '$' + parseFloat(originalData.amount).toFixed(2);
    cells[4].textContent = originalData.vendor || '-';
    cells[5].textContent = originalData.payment_method || '-';
    cells[6].innerHTML = `
        <button class="edit-btn" onclick="editExpense(${id})">Edit</button>
        <button class="delete-btn" onclick="deleteExpense(${id})">Delete</button>
    `;
}

async function deleteExpense(id) {
    if (!confirm('Are you sure you want to delete this expense?')) {
        return;
    }
    
    try {
        const response = await apiRequest(`/api/expenses/${id}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            loadExpenses(currentPage); // Reload current page
        } else {
            const error = await response.text();
            alert(`Error deleting expense: ${error}`);
        }
    } catch (error) {
        console.error('Delete expense error:', error);
        alert(`Error deleting expense: ${error.message}`);
    }
}

function displayChart(stats) {
    const ctx = document.getElementById('categoryChart').getContext('2d');
    
    if (expenseChart) {
        expenseChart.destroy();
    }
    
    const categories = Object.keys(stats.categories);
    const amounts = Object.values(stats.categories);
    
    expenseChart = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: categories,
            datasets: [{
                data: amounts,
                backgroundColor: [
                    '#FF6384',
                    '#36A2EB',
                    '#FFCE56',
                    '#4BC0C0',
                    '#9966FF',
                    '#FF9F40'
                ]
            }]
        },
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'bottom'
                },
                title: {
                    display: true,
                    text: `Total: $${stats.total.toFixed(2)}`
                }
            }
        }
    });
}

// Date validation function for DD/MM/YYYY format
function validateDate(dateStr) {
    const datePattern = /^(\d{2})\/(\d{2})\/(\d{4})$/;
    const match = dateStr.match(datePattern);
    
    if (!match) {
        return { valid: false, error: 'Date must be in DD/MM/YYYY format' };
    }
    
    const day = parseInt(match[1], 10);
    const month = parseInt(match[2], 10);
    const year = parseInt(match[3], 10);
    
    // Check valid ranges
    if (month < 1 || month > 12) {
        return { valid: false, error: 'Month must be between 01 and 12' };
    }
    
    if (day < 1 || day > 31) {
        return { valid: false, error: 'Day must be between 01 and 31' };
    }
    
    // Check if date is valid (handles leap years, different month lengths)
    const date = new Date(year, month - 1, day);
    if (date.getFullYear() !== year || date.getMonth() !== month - 1 || date.getDate() !== day) {
        return { valid: false, error: 'Invalid date' };
    }
    
    return { valid: true, date: date };
}

// Calculate total amount from preview data
function calculateTotal() {
    if (!previewData || previewData.length === 0) {
        return 0;
    }
    
    return previewData.reduce((total, expense) => {
        return total + (parseFloat(expense.amount) || 0);
    }, 0);
}

// Update total display
function updateTotalDisplay() {
    const totalElement = document.getElementById('totalAmount');
    const total = calculateTotal();
    totalElement.innerHTML = `<strong>Total Amount: $${total.toFixed(2)}</strong>`;
}

// Edit row functionality
function editRow(index) {
    const row = document.querySelector(`tr[data-index="${index}"]`);
    
    // Hide display values and show edit inputs
    row.querySelectorAll('.display-value').forEach(el => el.style.display = 'none');
    row.querySelectorAll('.edit-input').forEach(el => el.style.display = 'inline-block');
    
    // Toggle buttons
    row.querySelector('.edit-btn').style.display = 'none';
    row.querySelector('.save-btn').style.display = 'inline-block';
    row.querySelector('.cancel-btn').style.display = 'inline-block';
}

function saveRow(index) {
    const row = document.querySelector(`tr[data-index="${index}"]`);
    
    // Get input values
    const dateInput = row.querySelector('.date-input');
    const vendorInput = row.querySelector('.vendor-input');
    const descriptionInput = row.querySelector('.description-input');
    const categorySelect = row.querySelector('.category-select');
    const paymentMethodInput = row.querySelector('.payment-method-input');
    const dateError = row.querySelector('.date-error');
    
    // Validate date
    const dateValidation = validateDate(dateInput.value);
    if (!dateValidation.valid) {
        dateError.textContent = dateValidation.error;
        dateError.style.display = 'block';
        return;
    } else {
        dateError.style.display = 'none';
    }
    
    // Validate required fields
    if (!descriptionInput.value.trim()) {
        alert('Description is required');
        return;
    }
    
    // Update preview data
    previewData[index].date = dateValidation.date.toISOString();
    previewData[index].vendor = vendorInput.value.trim() || null;
    previewData[index].description = descriptionInput.value.trim();
    previewData[index].category = categorySelect.value;
    previewData[index].payment_method = paymentMethodInput.value.trim() || 'CSV Import';
    
    // Update display values
    row.querySelector('.date-cell .display-value').textContent = dateInput.value;
    row.querySelector('.vendor-cell .display-value').textContent = vendorInput.value.trim() || '-';
    row.querySelector('.description-cell .display-value').textContent = descriptionInput.value.trim();
    row.querySelector('.category-cell .display-value').textContent = categorySelect.value;
    row.querySelector('.payment-method-cell .display-value').textContent = paymentMethodInput.value.trim() || 'CSV Import';
    
    // Hide edit inputs and show display values
    row.querySelectorAll('.edit-input').forEach(el => el.style.display = 'none');
    row.querySelectorAll('.display-value').forEach(el => el.style.display = 'inline');
    
    // Toggle buttons
    row.querySelector('.edit-btn').style.display = 'inline-block';
    row.querySelector('.save-btn').style.display = 'none';
    row.querySelector('.cancel-btn').style.display = 'none';
}

function cancelEdit(index) {
    const row = document.querySelector(`tr[data-index="${index}"]`);
    const dateError = row.querySelector('.date-error');
    
    // Hide any error messages
    dateError.style.display = 'none';
    
    // Reset input values to current data
    const expense = previewData[index];
    const date = new Date(expense.date);
    const formattedDate = `${date.getDate().toString().padStart(2, '0')}/${(date.getMonth() + 1).toString().padStart(2, '0')}/${date.getFullYear()}`;
    
    row.querySelector('.date-input').value = formattedDate;
    row.querySelector('.vendor-input').value = expense.vendor || '';
    row.querySelector('.description-input').value = expense.description;
    row.querySelector('.category-select').value = expense.category;
    row.querySelector('.payment-method-input').value = expense.payment_method || 'CSV Import';
    
    // Hide edit inputs and show display values
    row.querySelectorAll('.edit-input').forEach(el => el.style.display = 'none');
    row.querySelectorAll('.display-value').forEach(el => el.style.display = 'inline');
    
    // Toggle buttons
    row.querySelector('.edit-btn').style.display = 'inline-block';
    row.querySelector('.save-btn').style.display = 'none';
    row.querySelector('.cancel-btn').style.display = 'none';
}

// Tab Management
function showTab(tabName) {
    // Hide all tab contents
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    
    // Remove active class from all tab buttons
    document.querySelectorAll('.tab-button').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Show selected tab and activate button
    document.getElementById(tabName + '-tab').classList.add('active');
    event.target.classList.add('active');
    
    // Load data for the active tab
    if (tabName === 'rules') {
        loadCategoryRules();
    } else if (tabName === 'expenses') {
        loadExpenses();
    }
}

// Category Rules Management
async function loadCategoryRules() {
    try {
        const response = await fetch('/api/categorization-rules');
        const rules = await response.json();
        
        const tbody = document.querySelector('#rulesTable tbody');
        tbody.innerHTML = '';
        
        rules.forEach(rule => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${rule.category}</td>
                <td>${rule.keyword}</td>
                <td>${rule.case_sensitive ? 'Yes' : 'No'}</td>
                <td>
                    <button class="delete-btn" onclick="deleteRule(${rule.id})">Delete</button>
                </td>
            `;
            tbody.appendChild(row);
        });
        
        // Also load categories for the dropdown
        loadRuleCategoryOptions();
    } catch (error) {
        console.error('Error loading category rules:', error);
    }
}

async function addRule() {
    const categorySelect = document.getElementById('ruleCategory').value.trim();
    const categoryCustom = document.getElementById('ruleCategoryCustom').value.trim();
    const category = categoryCustom || categorySelect;
    const keyword = document.getElementById('ruleKeyword').value.trim();
    const caseSensitive = document.getElementById('ruleCaseSensitive').checked;
    
    if (!category || !keyword) {
        alert('Please fill in both category and keyword fields');
        return;
    }
    
    try {
        const response = await apiRequest('/api/categorization-rules', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                category: category,
                keyword: keyword,
                case_sensitive: caseSensitive
            })
        });
        
        if (response.ok) {
            // Clear form
            document.getElementById('ruleCategory').value = '';
            document.getElementById('ruleCategoryCustom').value = '';
            document.getElementById('ruleKeyword').value = '';
            document.getElementById('ruleCaseSensitive').checked = false;
            
            // Reload rules table
            loadCategoryRules();
            
            // Reload dynamic categories for expenses tab
            loadDynamicCategories();
        } else {
            const error = await response.text();
            alert('Error adding rule: ' + error);
        }
    } catch (error) {
        console.error('Error adding rule:', error);
        alert('Error adding rule: ' + error.message);
    }
}

async function deleteRule(ruleId) {
    if (!confirm('Are you sure you want to delete this rule?')) {
        return;
    }
    
    try {
        const response = await apiRequest(`/api/categorization-rules/${ruleId}`, {
            method: 'DELETE'
        });
        
        if (response.ok) {
            loadCategoryRules();
            loadDynamicCategories();
        } else {
            const error = await response.text();
            alert('Error deleting rule: ' + error);
        }
    } catch (error) {
        console.error('Error deleting rule:', error);
        alert('Error deleting rule: ' + error.message);
    }
}

async function loadDynamicCategories() {
    try {
        const response = await fetch('/api/categories');
        const categories = await response.json();
        
        const categorySelect = document.getElementById('category');
        
        // Clear existing options except "All Categories"
        categorySelect.innerHTML = '<option value="">All Categories</option>';
        
        // Add categories from database
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category;
            option.textContent = category;
            categorySelect.appendChild(option);
        });
        
        // Add "Other" as fallback
        const otherOption = document.createElement('option');
        otherOption.value = 'Other';
        otherOption.textContent = 'Other';
        categorySelect.appendChild(otherOption);
        
    } catch (error) {
        console.error('Error loading categories:', error);
        // Fallback to hardcoded categories if API fails
        const categorySelect = document.getElementById('category');
        categorySelect.innerHTML = `
            <option value="">All Categories</option>
            <option value="Food & Dining">Food & Dining</option>
            <option value="Transportation">Transportation</option>
            <option value="Shopping">Shopping</option>
            <option value="Utilities">Utilities</option>
            <option value="Healthcare">Healthcare</option>
            <option value="Other">Other</option>
        `;
    }
}

async function loadRuleCategoryOptions() {
    try {
        const response = await fetch('/api/categories');
        const categories = await response.json();
        
        const categorySelect = document.getElementById('ruleCategory');
        
        // Clear existing options except the placeholder
        categorySelect.innerHTML = '<option value="">Select or type new category...</option>';
        
        // Add existing categories
        categories.forEach(category => {
            const option = document.createElement('option');
            option.value = category;
            option.textContent = category;
            categorySelect.appendChild(option);
        });
        
    } catch (error) {
        console.error('Error loading category options:', error);
    }
}
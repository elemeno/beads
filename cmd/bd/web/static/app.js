// Beads Web Interface - Client-side JavaScript

(function() {
    'use strict';

    // Debounce helper
    function debounce(func, wait) {
        let timeout;
        return function executedFunction(...args) {
            const later = () => {
                clearTimeout(timeout);
                func(...args);
            };
            clearTimeout(timeout);
            timeout = setTimeout(later, wait);
        };
    }

    // Show notification
    function showNotification(message, type = 'success') {
        const notification = document.getElementById('notification');
        if (!notification) return;

        notification.textContent = message;
        notification.className = 'notification ' + type;
        notification.style.display = 'block';

        setTimeout(() => {
            notification.style.display = 'none';
        }, 3000);
    }

    // Auto-save for select dropdowns
    function setupAutoSaveSelect() {
        const selects = document.querySelectorAll('.auto-save');

        selects.forEach(select => {
            select.addEventListener('change', async function() {
                const issueID = this.getAttribute('data-issue-id');
                const field = this.getAttribute('data-field');
                const value = this.value;

                this.classList.add('saving');

                try {
                    const response = await fetch(`/api/issues/${issueID}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            [field]: field === 'priority' ? parseInt(value) : value
                        })
                    });

                    if (response.ok) {
                        this.classList.remove('saving');
                        this.classList.add('saved');
                        setTimeout(() => this.classList.remove('saved'), 1000);
                        showNotification('Saved!', 'success');
                    } else {
                        throw new Error('Save failed');
                    }
                } catch (error) {
                    this.classList.remove('saving');
                    this.classList.add('error');
                    showNotification('Failed to save', 'error');
                    setTimeout(() => this.classList.remove('error'), 2000);
                }
            });
        });
    }

    // Auto-save for text inputs
    function setupAutoSaveInput() {
        const inputs = document.querySelectorAll('.auto-save-input');

        inputs.forEach(input => {
            const debouncedSave = debounce(async function(element) {
                const issueID = element.getAttribute('data-issue-id');
                const field = element.getAttribute('data-field');
                const value = element.value;

                element.classList.add('saving');

                try {
                    const response = await fetch(`/api/issues/${issueID}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            [field]: value
                        })
                    });

                    if (response.ok) {
                        element.classList.remove('saving');
                        element.classList.add('saved');
                        setTimeout(() => element.classList.remove('saved'), 1000);
                    } else {
                        throw new Error('Save failed');
                    }
                } catch (error) {
                    element.classList.remove('saving');
                    element.classList.add('error');
                    showNotification('Failed to save', 'error');
                    setTimeout(() => element.classList.remove('error'), 2000);
                }
            }, 1000);

            input.addEventListener('input', function() {
                debouncedSave(this);
            });
        });
    }

    // Auto-save for textareas
    function setupAutoSaveTextarea() {
        const textareas = document.querySelectorAll('.auto-save-textarea');

        textareas.forEach(textarea => {
            const debouncedSave = debounce(async function(element) {
                const issueID = element.getAttribute('data-issue-id');
                const field = element.getAttribute('data-field');
                const value = element.value;

                element.classList.add('saving');

                try {
                    const response = await fetch(`/api/issues/${issueID}`, {
                        method: 'PUT',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            [field]: value
                        })
                    });

                    if (response.ok) {
                        element.classList.remove('saving');
                        element.classList.add('saved');
                        setTimeout(() => element.classList.remove('saved'), 1000);
                    } else {
                        throw new Error('Save failed');
                    }
                } catch (error) {
                    element.classList.remove('saving');
                    element.classList.add('error');
                    showNotification('Failed to save', 'error');
                    setTimeout(() => element.classList.remove('error'), 2000);
                }
            }, 1000);

            textarea.addEventListener('input', function() {
                debouncedSave(this);
            });
        });
    }

    // Auto-save for editable title
    function setupEditableTitle() {
        const title = document.querySelector('.editable-title');
        if (!title) return;

        const debouncedSave = debounce(async function(element) {
            const issueID = element.getAttribute('data-issue-id');
            const field = element.getAttribute('data-field');
            const value = element.textContent.trim();

            if (!value) {
                element.classList.add('error');
                showNotification('Title cannot be empty', 'error');
                setTimeout(() => element.classList.remove('error'), 2000);
                return;
            }

            element.classList.add('saving');

            try {
                const response = await fetch(`/api/issues/${issueID}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        [field]: value
                    })
                });

                if (response.ok) {
                    element.classList.remove('saving');
                    element.classList.add('saved');
                    setTimeout(() => element.classList.remove('saved'), 1000);
                } else {
                    throw new Error('Save failed');
                }
            } catch (error) {
                element.classList.remove('saving');
                element.classList.add('error');
                showNotification('Failed to save title', 'error');
                setTimeout(() => element.classList.remove('error'), 2000);
            }
        }, 1000);

        title.addEventListener('input', function() {
            debouncedSave(this);
        });

        // Prevent newlines in title
        title.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.blur();
            }
        });
    }

    // Add dependency
    function setupAddDependency() {
        const form = document.getElementById('add-dep-form');
        if (!form) return;

        form.addEventListener('submit', async function(e) {
            e.preventDefault();

            const issueID = this.getAttribute('data-issue-id');
            const dependsOnID = document.getElementById('depends-on-id').value.trim();
            const depType = document.getElementById('dep-type').value;

            if (!dependsOnID) {
                showNotification('Please enter an issue ID', 'error');
                return;
            }

            try {
                const response = await fetch(`/api/issues/${issueID}/deps`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        depends_on_id: dependsOnID,
                        type: depType
                    })
                });

                if (response.ok) {
                    showNotification('Dependency added!', 'success');
                    // Reload page to show new dependency
                    setTimeout(() => location.reload(), 500);
                } else {
                    const data = await response.json();
                    showNotification(data.error || 'Failed to add dependency', 'error');
                }
            } catch (error) {
                showNotification('Failed to add dependency', 'error');
            }
        });
    }

    // Remove dependency
    function setupRemoveDependency() {
        const buttons = document.querySelectorAll('.remove-dep');

        buttons.forEach(button => {
            button.addEventListener('click', async function() {
                if (!confirm('Remove this dependency?')) {
                    return;
                }

                const issueID = this.getAttribute('data-issue-id');
                const dependsOnID = this.getAttribute('data-depends-on');
                const depType = this.getAttribute('data-type');

                try {
                    const response = await fetch(`/api/issues/${issueID}/deps`, {
                        method: 'DELETE',
                        headers: {
                            'Content-Type': 'application/json',
                        },
                        body: JSON.stringify({
                            depends_on_id: dependsOnID,
                            type: depType
                        })
                    });

                    if (response.ok) {
                        showNotification('Dependency removed!', 'success');
                        // Reload page to reflect changes
                        setTimeout(() => location.reload(), 500);
                    } else {
                        showNotification('Failed to remove dependency', 'error');
                    }
                } catch (error) {
                    showNotification('Failed to remove dependency', 'error');
                }
            });
        });
    }

    // Initialize on page load
    document.addEventListener('DOMContentLoaded', function() {
        setupAutoSaveSelect();
        setupAutoSaveInput();
        setupAutoSaveTextarea();
        setupEditableTitle();
        setupAddDependency();
        setupRemoveDependency();
    });
})();

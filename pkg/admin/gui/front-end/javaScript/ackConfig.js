document.addEventListener('DOMContentLoaded', function () {
    // Toggle ACK fields visibility when checkbox changes
    document.querySelectorAll('.ack-enabled-checkbox').forEach(function (checkbox) {
        var container = checkbox.closest('.protoConfigBlock');
        if (!container) return;
        var fields = container.querySelector('.ack-fields-container');
        if (!fields) return;

        checkbox.addEventListener('change', function () {
            fields.style.display = this.checked ? '' : 'none';
        });
    });

    // Cascade: populate account select when server changes
    document.querySelectorAll('.ack-server-select').forEach(function (serverSelect) {
        var container = serverSelect.closest('.ack-fields-container');
        if (!container) return;
        var accountSelect = container.querySelector('.ack-account-select');
        if (!accountSelect) return;

        // Parse existing replyTo value (format: "server:account" or "server")
        var currentReplyTo = serverSelect.dataset.currentReplyto || '';
        var currentServer = '';
        var currentAccount = '';
        if (currentReplyTo) {
            var parts = currentReplyTo.split(':');
            currentServer = parts[0] || '';
            currentAccount = parts[1] || '';
        }

        // Pre-select the current server
        if (currentServer) {
            for (var i = 0; i < serverSelect.options.length; i++) {
                if (serverSelect.options[i].value === currentServer) {
                    serverSelect.selectedIndex = i;
                    break;
                }
            }
        }

        serverSelect.addEventListener('change', function () {
            populateAccounts(serverSelect, accountSelect, '');
        });

        // Initialize accounts on page load
        if (serverSelect.value) {
            populateAccounts(serverSelect, accountSelect, currentAccount);
        }
    });

    function populateAccounts(serverSelect, accountSelect, preselect) {
        var selected = serverSelect.options[serverSelect.selectedIndex];
        accountSelect.innerHTML = '';

        if (!selected || !selected.value) {
            accountSelect.innerHTML = '<option value="">--</option>';
            return;
        }

        var accounts = [];
        try {
            accounts = JSON.parse(selected.dataset.accounts || '[]');
        } catch (e) {
            accounts = [];
        }

        if (accounts.length === 0) {
            accountSelect.innerHTML = '<option value="">--</option>';
            return;
        }

        // If only one account, select it automatically
        if (accounts.length === 1) {
            var opt = document.createElement('option');
            opt.value = accounts[0];
            opt.textContent = accounts[0];
            opt.selected = true;
            accountSelect.appendChild(opt);
            return;
        }

        accountSelect.innerHTML = '<option value="">--</option>';
        accounts.forEach(function (login) {
            var opt = document.createElement('option');
            opt.value = login;
            opt.textContent = login;
            if (login === preselect) opt.selected = true;
            accountSelect.appendChild(opt);
        });
    }
});

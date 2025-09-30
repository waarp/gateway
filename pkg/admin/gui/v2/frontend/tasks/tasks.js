function openTaskModal(ruleID, chain, rank) {
    const modalEl = document.getElementById('addTaskModal');
    if (!modalEl) throw new Error('Unable to find the Add Task modal element');

    if (modalEl.getAttribute("chain") === chain &&
        modalEl.getAttribute("rank") === String(rank)) {
        const modal = new bootstrap.Modal(modalEl);
        modal.show()
        return
    }

    fetch(`tasks/modal?ruleID=${ruleID}&chain=${chain}&rank=${rank}`)
        .then(collectFetchText)
        .then(html => {
            modalEl.setAttribute("chain", chain)
            modalEl.setAttribute("rank", rank)

            const scriptEl = document.createRange().createContextualFragment(html);
            modalEl.replaceChildren(scriptEl)
            reloadTooltips()

            const modal = new bootstrap.Modal(modalEl);
            modal.show();
        })
        .catch(catchFetchErr)
}

function deleteTask(ruleID, chain, rank, confirmMsg) {
    if (!confirm(confirmMsg))
        return;

    fetch(`tasks?ruleID=${ruleID}&chain=${chain}&rank=${rank}`, {method: 'DELETE'})
        .then(resp => {
            if (!resp.ok) return Promise.reject(resp)
            location.reload()
        })
        .catch(catchFetchErr)
}

function reloadTooltips() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]')
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl))
}

// Enable drag-and-drop reordering for tasks tables
document.addEventListener('DOMContentLoaded', () => {
    initTaskDragAndDrop();
});

function initTaskDragAndDrop() {
    // For each tasks table body, wire up DnD
    document.querySelectorAll('table tbody').forEach(tbody => {
        // Select all task rows irrespective of draggable attribute
        const rows = Array.from(tbody.querySelectorAll('tr[data-task-id]'));
        if (rows.length === 0) return;

        // Show/hide controls related to the table that contains this tbody
        const container = tbody.closest('div.collapse') || document; // section container
        const controls = container.querySelector('.drag-controls');
        const btnCancel = controls?.querySelector('.btn.btn-secondary');
        const btnApply = controls?.querySelector('.btn.btn-success');

        let dragSrc = null;
        let dirty = false;

        const setControlsVisible = visible => {
            if (!controls) return;
            controls.style.display = visible ? 'inline-block' : 'none';
        };

        const markDirty = () => {
            if (dirty) return;
            dirty = true;
            setControlsVisible(true);
        };

        // DnD handlers
        const handleDragStart = (e) => {
            dragSrc = e.currentTarget;
            e.dataTransfer.effectAllowed = 'move';
            // For Firefox compatibility
            e.dataTransfer.setData('text/plain', '');
            dragSrc.style.cursor = 'grabbing';
        };

        const handleDragOver = (e) => {
            e.preventDefault(); // allow drop
            if (!dragSrc) return;
            e.dataTransfer.dropEffect = 'move';
            const targetRow = e.currentTarget;
            if (targetRow === dragSrc) return;

            const bounding = targetRow.getBoundingClientRect();
            const offset = e.clientY - bounding.top;
            const shouldInsertBefore = offset < bounding.height / 2;

            if (shouldInsertBefore) {
                if (targetRow.previousElementSibling !== dragSrc) {
                    tbody.insertBefore(dragSrc, targetRow);
                }
            } else {
                if (targetRow.nextElementSibling !== dragSrc) {
                    tbody.insertBefore(dragSrc, targetRow.nextElementSibling);
                }
            }
        };

        const handleDrop = (e) => {
            e.preventDefault();
            if (!dragSrc) return;
            markDirty();
            dragSrc.removeAttribute('draggable'); // disable again
            dragSrc.style.cursor = '';
            dragSrc = null;
        };

        const handleDragEnd = (e) => {
            e.currentTarget.removeAttribute('draggable'); // disable again
            e.currentTarget.style.cursor = '';
            dragSrc = null;
        };

        // Attach listeners to existing rows
        const attachRowDnD = (row) => {
            // Enable draggable only while interacting with the handle
            const handle = row.querySelector('.drag-handle');
            if (handle) {
                const enableDraggable = () => {
                    row.setAttribute('draggable', 'true');
                    // User started reordering again -> show controls
                    setControlsVisible(true);
                };
                const disableDraggable = () => row.removeAttribute('draggable');

                handle.addEventListener('mousedown', enableDraggable);
                handle.addEventListener('touchstart', enableDraggable, { passive: true });
                document.addEventListener('mouseup', disableDraggable);
                document.addEventListener('touchend', disableDraggable, { passive: true });
                document.addEventListener('touchcancel', disableDraggable, { passive: true });
            }

            row.addEventListener('dragstart', handleDragStart);
            row.addEventListener('dragover', handleDragOver);
            row.addEventListener('drop', handleDrop);
            row.addEventListener('dragend', handleDragEnd);
        };
        rows.forEach(attachRowDnD);

        // Cancel -> reload page to discard changes
        btnCancel?.addEventListener('click', () => {
            location.reload();
        });

        // Apply -> collect new order and POST
        btnApply?.addEventListener('click', async () => {
            const ordered = Array.from(tbody.querySelectorAll('tr[data-task-id]')).map((tr, idx) => {
                const ruleID = tr.getAttribute('data-task-id');
                const chain = tr.getAttribute('data-chain');
                const rank = tr.getAttribute('data-rank');
                if (!ruleID || !chain || !rank) return null;
                return { ruleID: Number(ruleID), chain, oldRank: Number(rank), newRank: idx, tr };
            }).filter(Boolean);

            if (ordered.length === 0) return;

            const { ruleID, chain } = ordered[0];
            const payload = {
                ruleID,
                chain,
                // Send the original ranks in their new order
                ranks: ordered.map(o => o.oldRank)
            };

            fetch('tasks', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            }).then(resp => {
                if (!resp.ok) return Promise.reject(resp)
                // On success: hide controls and reset dirty state
                dirty = false;
                setControlsVisible(false);
            }).catch (catchFetchErr)
        });
    });
}

document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('table tbody').forEach(tbody => {
        const collapseSection = tbody.closest('.no-anim-collapse');
        if (!collapseSection)
            return;

        const dragControls = collapseSection.querySelector('.dragControls');
        const btnCancel = dragControls.querySelector('button.btn-secondary');
        const btnApply = dragControls.querySelector('button.btn-success');
        let draggedRow = null;

        // grab cursor
        tbody.querySelectorAll('tr[draggable="true"]').forEach(row => {
            row.style.cursor = 'grab';
        });

        // drag started
        tbody.addEventListener('dragstart', e => {
            const row = e.target.closest('tr[draggable="true"]');
            if (!row)
                return;
            draggedRow = row;
            row.style.cursor = 'grabbing';
            e.dataTransfer.setData('text/plain', '');
            e.dataTransfer.effectAllowed = 'move';
            row.classList.add('opacity-25');
            dragControls.style.display = 'block';
        });

        // drag over
        tbody.addEventListener('dragover', e => {
            e.preventDefault();
            if (!draggedRow)
                return;
            const targetRow = e.target.closest('tr[draggable="true"]');
            if (!targetRow || targetRow === draggedRow)
                return;
            const rect = targetRow.getBoundingClientRect();
            const after = (e.clientY - rect.top) > rect.height / 2;
            after ? targetRow.after(draggedRow) : targetRow.before(draggedRow);
        });

        tbody.addEventListener('drop', e => e.preventDefault());

        tbody.addEventListener('dragend', () => {
            if (!draggedRow)
                return;
            draggedRow.classList.remove('opacity-25');
            draggedRow.style.cursor = 'grab';
            [...tbody.rows].forEach((row, idx) => {
                row.dataset.rank = idx;
            });
            draggedRow = null;
        });

        btnCancel.addEventListener('click', () => location.reload());

        btnApply.addEventListener('click', () => {
            const orderArr = Array.from(tbody.querySelectorAll('tr')).map(row => row.dataset.taskId);
            let fieldName = '';
            switch (collapseSection.id) {
                case 'preTasksCollapse': fieldName = 'newOrderPreTasks'; break;
                case 'postTasksCollapse': fieldName = 'newOrderPostTasks'; break;
                case 'errorTasksCollapse': fieldName = 'newOrderErrorTasks'; break;
            }
            const form = document.createElement('form');
            form.method = 'post';
            form.action = window.location.pathname + window.location.search;
            const input = document.createElement('input');
            input.type = 'hidden';
            input.name = fieldName;
            input.value = orderArr.join(',');
            form.appendChild(input);
            document.body.appendChild(form);
            form.submit();
        });
    });
});
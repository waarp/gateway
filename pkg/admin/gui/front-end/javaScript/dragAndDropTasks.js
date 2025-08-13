document.addEventListener('DOMContentLoaded', () => {
    document.querySelectorAll('table tbody').forEach(tbody => {
        const section   = tbody.closest('.no-anim-collapse');
        if (!section)
            return;
        const controls  = section.querySelector('.dragControls');
        const btnCancel = controls.querySelector('button.btn-secondary');
        const btnApply  = controls.querySelector('button.btn-success');
        let draggedRow  = null;

        tbody.querySelectorAll('tr[draggable="true"]').forEach(tr => {
            tr.style.cursor = 'grab';
        });

        tbody.addEventListener('dragstart', e => {
            const tr = e.target.closest('tr[draggable="true"]');
            if (!tr) return;
            draggedRow = tr;
            tr.style.cursor = 'grabbing';
            e.dataTransfer.setData('text/plain', '');
            e.dataTransfer.effectAllowed = 'move';
            tr.classList.add('opacity-25');
            controls.style.display = 'block';
        });

        tbody.addEventListener('dragover', e => {
            e.preventDefault();
            if (!draggedRow) return;
            const target = e.target.closest('tr[draggable="true"]');
            if (!target || target === draggedRow) return;
            const rect  = target.getBoundingClientRect();
            const after = (e.clientY - rect.top) > rect.height / 2;
            after ? target.after(draggedRow) : target.before(draggedRow);
        });

        tbody.addEventListener('drop', e => e.preventDefault());

        tbody.addEventListener('dragend', () => {
            if (!draggedRow) return;
            draggedRow.classList.remove('opacity-25');
            draggedRow.style.cursor = 'grab';
            [...tbody.rows].forEach((tr, idx) => {
                tr.dataset.rank = idx;
            });
            draggedRow = null;
        });

        btnCancel.addEventListener('click', () => location.reload());

        btnApply.addEventListener('click', () => {
            const orderArr = Array.from(tbody.querySelectorAll('tr')).map(tr => tr.dataset.taskId);
            let fieldName = '';
            switch (section.id) {
                case 'preTasksCollapse':  fieldName = 'newOrderPreTasks';  break;
                case 'postTasksCollapse': fieldName = 'newOrderPostTasks'; break;
                case 'errorTasksCollapse':fieldName = 'newOrderErrorTasks';break;
            }
            const form  = document.createElement('form');
            form.method = 'post';
            form.action = window.location.pathname + window.location.search;
            const input = document.createElement('input');
            input.type  = 'hidden';
            input.name  = fieldName;
            input.value = orderArr.join(',');
            form.appendChild(input);
            document.body.appendChild(form);
            form.submit();
        });
    });
});
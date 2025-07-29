document.addEventListener('DOMContentLoaded', function () {
    document.querySelectorAll('.taskType-select').forEach(select => {
        const ctx = select.closest('.modal-body');
        function updateTaskBlocks() {
            ctx.querySelectorAll('.taskBlock').forEach(b => b.style.display = 'none');
            const type = select.value;
            if (!type) return;
            const block = ctx.querySelector(`[id="task${type}"]`);
            if (block) block.style.display = 'block';
        }
        select.addEventListener('change', updateTaskBlocks);
        updateTaskBlocks();
    });
});
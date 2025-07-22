document.addEventListener('DOMContentLoaded', function () {
    const select = document.getElementById('addPreTaskType');
    if (!select) return;

    function hideAllTaskBlocks() {
        document.querySelectorAll('.taskBlock').forEach(block => {
            block.style.display = 'none';
        });
    }

    function showTaskBlockForType(type) {
        hideAllTaskBlocks();
        if (!type) return;
        const block = document.getElementById('task' + type);
        if (block) block.style.display = 'block';
    }

    select.addEventListener('change', function () {
        showTaskBlockForType(this.value);
    });

    showTaskBlockForType(select.value);
});
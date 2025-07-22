document.addEventListener("DOMContentLoaded", function() {
    if (window.modalOpen && window.modalOpen !== "") {
        var modal = document.getElementById(window.modalOpen);
        if (modal) {
            // 1) si la modal est dans une collapse, on l’ouvre d’abord
            var collapseParent = modal.closest('.no-anim-collapse');
            if (collapseParent) {
                bootstrap.Collapse.getOrCreateInstance(collapseParent).show();
            }
            // 2) puis on affiche la modal
            var reOpenModal = new bootstrap.Modal(modal);
            reOpenModal.show();
        }
    }

    var popup = document.getElementById('errorPopup');
    if (popup) {
        setTimeout(function() {
            popup.classList.remove('show');
            setTimeout(function() { 
                popup.remove(); 
            }, 500);
        }, 5000);
    }
});
function filterPermissions() {
    let permissions = document.getElementById('permissions');
    let permissionsType = document.getElementById('permissionsType');
    let permissionsValue = document.getElementById('permissionsValue');
    let applyBtn = document.getElementById('applyBtn');

    function changeValueSelect() {
        if ((permissions.value === "" && permissionsType.value === "" && permissionsValue.value === "") || (permissions.value && permissionsType.value && permissionsValue.value ))  {
            applyBtn.disabled = false
        } else {
            applyBtn.disabled = true
        }
    }
    permissions.addEventListener('change', changeValueSelect);
    permissionsType.addEventListener('change', changeValueSelect);
    permissionsValue.addEventListener('change', changeValueSelect);
}

function tooltip() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]');
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl));
}

function confirmPassword(password, passwordConfirm, language) {
    const pwd = document.getElementById(password);
    const confirmPwd = document.getElementById(passwordConfirm);
    if (pwd.value !== confirmPwd.value) {
        confirmPwd.classList.add('is-invalid');
        if (language === "en")
            confirmPwd.setCustomValidity('Passwords do not match');
        else if (language === "fr")
            confirmPwd.setCustomValidity('Les mots de passe ne correspondent pas');
        return false;
    } else {
        confirmPwd.classList.remove('is-invalid');
        confirmPwd.setCustomValidity('');
        return true;
    }
}

function eyePassword(id, btn) {
    const password = document.getElementById(id);

    password.type = password.type === "password" ? "text" : "password"

    const icon = btn.querySelector("i");
    icon.classList.toggle("fa-eye");
    icon.classList.toggle("fa-eye-slash");
}

function autoCompleteSearch() {
    const input  = document.getElementById("searchUser");
    const list   = document.getElementById("autocomplete");
    const button = document.querySelector('.btn-navbar');
    list.style.right = `${button.offsetWidth}px`;
    list.style.width = `${input.offsetWidth + 10}px`;

    input.addEventListener("input", async function () {
        const query = this.value.trim();
        list.innerHTML = "";
        if (query.length === 0) {
            return;
        }

        try {
            const response = await fetch(`/webui/autocompletion?q=${encodeURIComponent(query)}`);    
            const names = await response.json();

            names.forEach(name => {
                const li = document.createElement("li");
                li.className = "list-group-item list-group-item-action";
                li.textContent = name;
                li.onclick = () => {
                    input.value = name;
                    list.classList.add("d-none");
                };
                list.appendChild(li);
            });

            list.classList.toggle("d-none", names.length === 0);
        } catch {}
    });
}


document.addEventListener('DOMContentLoaded', function () {
    filterPermissions();
    tooltip();
    autoCompleteSearch();
});
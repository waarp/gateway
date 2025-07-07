function autoCompleteSearch() {
    const input  = document.getElementById("search");
    const list   = document.getElementById("autocomplete");
    const button = document.querySelector('.btn-navbar');
    const pageType = document.getElementById("pageType").value;
    list.style.right = `${button.offsetWidth}px`;
    list.style.width = `${input.offsetWidth + 10}px`;

    input.addEventListener("input", async function () {
        const query = this.value.trim();
        list.innerHTML = "";
        if (query.length === 0) {
            return;
        }

        let url = `/webui/autocompletion/${pageType}?q=${encodeURIComponent(query)}`;
        if (pageType === "credentialPartner" || pageType === "remoteAccount") {
            const partnerID = document.querySelector('input[name="partnerID"]');
            url += `&partnerID=${encodeURIComponent(partnerID.value)}`;
        }
<<<<<<< HEAD
        if (pageType === "credentialServer" || pageType === "localAccount") {
            const serverID = document.querySelector('input[name="serverID"]');
            url += `&serverID=${encodeURIComponent(serverID.value)}`;
        }
        if (pageType === "credentialRemoteAccount") {
            const partnerID = document.querySelector('input[name="partnerID"]');
            const accountID = document.querySelector('input[name="accountID"]');
            if (partnerID && accountID) {
                url += `&partnerID=${encodeURIComponent(partnerID.value)}&accountID=${encodeURIComponent(accountID.value)}`;
            }
        }
        if (pageType === "credentialLocalAccount") {
            const serverID = document.querySelector('input[name="serverID"]');
            const accountID = document.querySelector('input[name="accountID"]');
            if (serverID && accountID) {
                url += `&serverID=${encodeURIComponent(serverID.value)}&accountID=${encodeURIComponent(accountID.value)}`;
            }
        }

        if (pageType === "credentialAccount") {
            const partnerID = document.querySelector('input[name="partnerID"]');
            const accountID = document.querySelector('input[name="accountID"]');
            if (partnerID && accountID) {
                url += `&partnerID=${encodeURIComponent(partnerID.value)}&accountID=${encodeURIComponent(accountID.value)}`;
            }
        }
        if (pageType === "credentialLocalAccount") {
            const serverID = document.querySelector('input[name="serverID"]');
            const accountID = document.querySelector('input[name="accountID"]');
            if (serverID && accountID) {
                url += `&serverID=${encodeURIComponent(serverID.value)}&accountID=${encodeURIComponent(accountID.value)}`;
            }
        }
=======
>>>>>>> 86cb8059 (feat/cleanup project)

        if (pageType === "credentialAccount") {
            const partnerID = document.querySelector('input[name="partnerID"]');
            const accountID = document.querySelector('input[name="accountID"]');
            if (partnerID && accountID) {
                url += `&partnerID=${encodeURIComponent(partnerID.value)}&accountID=${encodeURIComponent(accountID.value)}`;
            }
        }

        try {
            const response = await fetch(url);
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
    autoCompleteSearch();
    document.addEventListener('click', function(event) {
        const navbar = document.querySelector('.navbar');
        const list = document.getElementById('autocomplete');
        if (navbar && list && !navbar.contains(event.target)) {
            list.classList.add('d-none');
        }
    });
});
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
});
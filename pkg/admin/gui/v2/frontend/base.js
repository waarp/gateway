function collectFetchText(resp) {
    if (!resp.ok) return Promise.reject(resp)
    return resp.text()
}

function collectFetchJson(resp) {
    if (!resp.ok) return Promise.reject(resp)
    return resp.json()
}

function popupErr(msg) {
    const alertDiv = document.createElement('div');
    alertDiv.className = 'alert alert-danger alert-dismissible fade show position-fixed top-0 start-50 translate-middle-x mt-3';
    alertDiv.setAttribute('role', 'alert');
    alertDiv.style.zIndex = '9999';

    alertDiv.innerHTML = `
        ERROR: ${msg}
        <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
    `;

    document.body.appendChild(alertDiv);

    // Auto dismiss after 5 seconds
    setTimeout(() => {
        const bsAlert = new bootstrap.Alert(alertDiv);
        bsAlert.close();
    }, 5000);
}

function catchFetchErr(error) {
    if (typeof error.text === "function") {
        error.text().then(msg => {
            console.error(msg)
            popupErr(msg)
        }).catch(unknownError => {
            console.error(unknownError.statusText)
            popupErr(unknownError.statusText)
        });
    } else {
        console.error(error)
        popupErr(error)
    }
}

function disableElem(id) {
    const elem = document.getElementById(id)
    if (!elem) return

    elem.hidden = true
    const children = elem.querySelectorAll("input")
    children.forEach(child => {
        child.value = ""
    })
}

function enableElem(id) {
    const elem = document.getElementById(id)
    if (!elem) return

    elem.hidden = false
}
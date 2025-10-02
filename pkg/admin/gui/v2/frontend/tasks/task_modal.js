function changeTaskForm(ruleID, chain, rank, type) {
    type = encodeURIComponent(type)

    fetch(`tasks/forms?ruleID=${ruleID}&chain=${chain}&rank=${rank}&type=${type}`)
        .then(collectFetchText)
        .then(html => {
            const formEl = document.getElementById('taskTypeFields');
            const scriptEl = document.createRange().createContextualFragment(html);

            formEl.replaceChildren(scriptEl)
            reloadTooltips()
        })
        .catch(catchFetchErr)
}

function reloadTooltips() {
    const tooltipTriggerList = document.querySelectorAll('[data-bs-toggle="tooltip"]')
    const tooltipList = [...tooltipTriggerList].map(tooltipTriggerEl => new bootstrap.Tooltip(tooltipTriggerEl))
}

function onSubmitForm(form) {
    const elems = form.querySelectorAll('input[onsubmit]');
    elems.forEach(elem => {
        elem.onsubmit()
    });

    const formData = new FormData(form)

    fetch(form.action, {method: form.method, body: formData})
        .then(resp => {
            if (!resp.ok) return Promise.reject(resp)
            window.location.reload()
        })
        .catch(catchFetchErr)

    return false
}

function emptySelect(selectEl, value) {
    const placeholder = selectEl.querySelector('option[value=""]')
    if (!value)
        placeholder.selected = true

    const curOpts = selectEl.querySelectorAll('option[value]:not([value=""])')
    curOpts.forEach(opt => {selectEl.removeChild(opt)})
}

function fillSelect(url, selectEl, value) {
    fetch(url)
        .then(collectFetchJson)
        .then(elems => {
            // Enable the select
            selectEl.disabled = false

            // Re-select the placeholder and remove the previous options
            emptySelect(selectEl)

            // Insert the new options
            elems.forEach(elem => {
                const option = document.createElement('option')
                option.value = elem
                option.text = elem
                if (elem === value)
                    option.selected = true

                selectEl.appendChild(option)
            })

            if (selectEl.onchange != null)
                selectEl.onchange()
        })
        .catch(catchFetchErr)
}

function uptKeyListWith(operation, method, value, selectId) {
    const keyElem = document.getElementById(selectId)
    if (!method) {
        keyElem.disabled = true
        return
    }

    fillSelect(`listing/keys?operation=${operation}&method=${method}`, keyElem, value)
}

function aggregateDuration(timeRow) {
    const units = ["d","h","m","s","ms"]

    let duration = ''
    units.forEach(unit => {
        const el = timeRow.querySelector(`.time-input-${unit}`)
        if (el && el.value && el.value !== '0')
            duration += `${el.value}${unit}`
    })

    return duration
}

function toggleInputs(value, elemId) {
    const elem = document.getElementById(elemId)
    const inputs = elem.querySelectorAll('input:not([type="hidden"])')

    if (value) {
        elem.hidden = false
        inputs.forEach(input => {
            input.required = true
        })
    } else {
        elem.hidden = true
        inputs.forEach(input => {
            input.value = ""
            input.required = false
        })
    }
}
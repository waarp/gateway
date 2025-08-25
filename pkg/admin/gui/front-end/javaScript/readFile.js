function readFile(hiddenName, fileName) {
    const file = document.querySelector('input[type="file"][name="' + fileName + '"]');
    const hidden = document.getElementById(hiddenName);
    if (file && hidden) {
        file.addEventListener('change', e => {
            const file = e.target.files[0];
            if (!file)
                return hidden.value = "";
            const reader = new FileReader();
            reader.onload = event => hidden.value = event.target.result;
            reader.readAsText(file);
        });
    }
}
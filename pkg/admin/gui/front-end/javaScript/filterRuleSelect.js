function filterRuleSelect() {
  const send = document.getElementById('filterRuleSend');
  const receive = document.getElementById('filterRuleReceive');

  send.addEventListener('change', function() {
    if (send.value && receive.value) {
      receive.value = "";
    }
  });

  receive.addEventListener('change', function() {
    if (receive.value && send.value) {
      send.value = "";
    }
  });
}

window.addEventListener('DOMContentLoaded', filterRuleSelect);
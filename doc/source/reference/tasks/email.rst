.. _ref-task-email:

EMAIL
=====

Le traitement ``EMAIL`` utilise SMTP pour envoyer un email aux destinataires
spécifiés. Les paramètres de la tâche sont:

* ``sender`` (*string*) - L'adresse email de l'expéditeur de l'email. Cette
  adresse doit préalablement avoir été renseignée via :ref:`REST<ref-rest-smtp>`
  ou bien via :ref:`CLI<ref-cli-smtp>`.
* ``recipients`` (*string*) - Les adresses email auxquelles l'email sera destiné.
  Les adresses doivent être séparées par des virgules (`,`).
* ``template`` (*string*) - Le nom du template d'email à utiliser pour
  le message. Ce template doit préalablement avoir été renseigné via
  :ref:`REST<ref-rest-emails>` ou bien via :ref:`CLI<ref-cli-emails>`.
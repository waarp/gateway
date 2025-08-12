============================
Modifier un identifiant SMTP
============================

.. program:: waarp-gateway email credential update

Modifie un identifiant SMTP existant. Les champs omits resteront inchangés.

**Commande**

.. code-block:: shell

   waarp-gateway email credential update <EMAIL>

**Options**

.. option:: -e <EMAIL>, --email-address=<EMAIL>

   La nouvelle adresse email qui servira à envoyer les emails (expéditeur). Doit
   être unique.

.. option:: -s <ADDRESS>, --server--address=<ADDRESS>

   L'adresse (port inclus) du serveur SMTP auquel Gateway se connectera pour
   envoyer les emails.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login à utiliser pour s'authentifier auprès du serveur SMTP. Pour rappel,
   le serveur SMTP doit supporter l'extension "AUTH" du protocole SMTP.
   Laisser vide pour désactiver l'authentification (non recommandé).

.. option:: -p <PASSWORD>, --password=<PASSWORD>

   Le mot de passe à utiliser pour s'authentifier auprès du serveur SMTP. Pour
   rappel, Gateway supporte les mécanismes "PLAIN", "LOGIN" et "CRAM-MD5".
   Laisser vide pour désactiver l'authentification (non recommandé).

**Exemple**

.. code-block:: shell

   waarp-gateway email credential update "old@example.com" -e "waarp@example.com" -s "smtp.example.com:587" -l "waarp" -p "sesame"

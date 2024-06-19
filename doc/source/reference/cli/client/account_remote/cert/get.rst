===================================================
[OBSOLÈTE] Afficher un certificat de compte distant
===================================================

.. program:: waarp-gateway account remote cert get

Affiche les informations du certificat demandé. Les noms du partenaire, du compte
et du certificat doivent être spécifiés en arguments de programme.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" cert "<LOGIN>" get "<CERT>"

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'waarp_sftp' cert 'titi' get 'key_titi'

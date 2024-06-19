=================================================
[OBSOLÈTE] Afficher un certificat de compte local
=================================================

.. program:: waarp-gateway account local cert get

Affiche les informations du certificat demandé. Les noms du serveur, du compte
et du certificat doivent être spécifiés en arguments de programme.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" cert "<LOGIN>" get "<CERT>"

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' cert 'tata' get 'key_tata'

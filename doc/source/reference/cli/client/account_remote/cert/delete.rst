====================================================
[OBSOLÈTE] Supprimer un certificat de compte distant
====================================================

.. program:: waarp-gateway account remote cert delete

Supprime le certificat demandé. Les noms du partenaire, du compte et du certificat
doivent être spécifiés en arguments de programme.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" cert "<LOGIN>" delete "<CERT>"

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'waarp_sftp' cert 'titi' delete 'key_titi'

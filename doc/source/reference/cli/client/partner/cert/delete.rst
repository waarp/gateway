================================================
[OBSOLÈTE] Supprimer un certificat de partenaire
================================================

.. program:: waarp-gateway partner cert delete

.. describe:: waarp-gateway partner cert <PARTNER> delete <CERT>

Supprime le certificat demandé. Les noms du partenaire et du certificat doivent
être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner cert 'waarp_sftp' delete 'waarp_hostkey'
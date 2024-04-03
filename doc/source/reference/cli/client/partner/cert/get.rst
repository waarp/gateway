===============================================
[OBSOLÈTE] Afficher un certificat de partenaire
===============================================

.. program:: waarp-gateway partner cert get

.. describe:: waarp-gateway partner cert <PARTNER> get <CERT>

Affiche les informations du certificat demandé. Les noms du partenaire et du
certificat doivent être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner cert 'waarp_sftp' get 'waarp_hostkey'
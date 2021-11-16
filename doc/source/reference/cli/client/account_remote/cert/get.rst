========================================
Afficher un certificat de compte distant
========================================

.. program:: waarp-gateway account remote cert get

.. describe:: waarp-gateway account remote <PARTNER> cert <LOGIN> get <CERT>

Affiche les informations du certificat demandé. Les noms du partenaire, du compte
et du certificat doivent être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 account remote waarp_sftp cert titi get cert_sftp
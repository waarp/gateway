========================================
Supprime un certificat de compte distant
========================================

.. program:: waarp-gateway account remote cert delete

.. describe:: waarp-gateway <ADDR> account remote <PARTNER> cert <LOGIN> delete <CERT>

Supprime le certificat demandé. Les noms du partenaire, du compte et du certificat
doivent être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 account remote waarp_sftp cert titi delete cert_titi
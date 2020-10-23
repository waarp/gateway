====================================
Supprime un certificat de partenaire
====================================

.. program:: waarp-gateway partner cert delete

.. describe:: waarp-gateway <ADDR> partner cert <PARTNER> delete <CERT>

Supprime le certificat demandé. Les noms du partenaire et du certificat doivent
être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 partner cert waarp_sftp delete cert_waarp
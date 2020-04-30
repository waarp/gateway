=================================
Supprime un certificat de serveur
=================================

.. program:: waarp-gateway server cert delete

.. describe:: waarp-gateway <ADDR> server cert <SERVER> delete <CERT>

Supprime le certificat demandé. Les noms du serveur et du certificat doivent
être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 server cert serveur_sftp delete cert_sftp
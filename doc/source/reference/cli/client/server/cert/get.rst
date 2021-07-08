=================================
Afficher un certificat de serveur
=================================

.. program:: waarp-gateway server cert get

.. describe:: waarp-gateway <ADDR> server cert <SERVER> get <CERT>

Affiche les informations du certificat demandé. Les noms du serveur et du
certificat doivent être spécifiés en arguments de programme.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 serveur cert gw_r66 get cert_r66
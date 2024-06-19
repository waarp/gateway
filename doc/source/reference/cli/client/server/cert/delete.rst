=============================================
[OBSOLÈTE] Supprimer un certificat de serveur
=============================================

.. program:: waarp-gateway server cert delete

Supprime le certificat demandé. Les noms du serveur et du certificat doivent
être spécifiés en arguments de programme.

**Commande**

.. code-block:: shell

   waarp-gateway server cert "<SERVER>" delete "<CERT>"

**Exemple**

.. code-block:: shell

   waarp-gateway server cert 'gw_r66' delete 'cert_r66'

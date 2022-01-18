========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway server auth delete

.. describe:: waarp-gateway server <SERVER> auth delete <AUTH>

Supprime la valeur d'authentification données du serveur.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' server auth 'server_sftp' delete 'sftp_hostkey'

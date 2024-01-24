========================================
Consulter une méthode d'authentification
========================================

.. program:: waarp-gateway server credential get

.. describe:: waarp-gateway server <SERVER> credential get <CRED_NAME>

Affiche les informations de la méthode d'authentification donnée.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' server credential 'server_sftp' get 'sftp_hostkey'

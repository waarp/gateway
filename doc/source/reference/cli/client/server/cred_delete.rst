========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway server credential delete

Supprime la valeur d'authentification données du serveur.

**Commande**

.. code-block:: shell

   waarp-gateway server credential "<SERVER>" delete "<CREDENTIAL>"

**Exemple**

.. code-block:: shell

   waarp-gateway server credential 'server_sftp' delete 'sftp_hostkey'

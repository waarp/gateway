========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway account local credential delete

Supprime la valeur d'authentification donnée du compte local.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<SERVER>" credential "<LOGIN>" delete "<CREDENTIAL>"

**Exemple**

.. code-block:: shell

   waarp-gateway account local "sftp_server" credential "openssh" delete "openssh_hostkey"

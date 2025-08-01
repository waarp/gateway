========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway account remote credential delete

Supprime la valeur d'authentification donnée du compte distant.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" credential "<LOGIN>" delete "<CREDENTIAL>"

**Exemple**

.. code-block:: shell

   waarp-gateway account remote "openssh" credential "waarp_ssh" delete "password"

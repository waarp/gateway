========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway account local credential delete

Supprime la valeur d'authentification données du compte.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" credential "<LOGIN>" delete "<CREDENTIAL>"

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'gw_r66' credential 'tata' delete 'r66_password'

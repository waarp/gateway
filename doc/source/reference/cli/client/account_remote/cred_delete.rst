========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway account local auth delete

Supprime la valeur d'authentification données du compte.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" credential "<LOGIN>" delete "<CREDENTIAL>"

**Options**

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'gw_r66' auth 'tata' delete 'r66_password'

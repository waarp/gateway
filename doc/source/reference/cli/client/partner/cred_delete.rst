========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway partner credential delete

Supprime la valeur d'authentification donnée du partenaire.

**Commande**

.. code-block:: shell

   waarp-gateway partner credential "<PARTNER>" delete "<CREDENTIAL>"

**Exemple**

.. code-block:: shell

   waarp-gateway partner credential 'openssh' delete 'openssh_hostkey'

========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway partner credential delete

.. describe:: waarp-gateway partner <PARTNER> credential delete <CRED_NAME>

Supprime la valeur d'authentification donnée du partenaire.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner credential 'openssh' delete 'openssh_hostkey'

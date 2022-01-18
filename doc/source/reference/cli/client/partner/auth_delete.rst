========================================
Supprimer une méthode d'authentification
========================================

.. program:: waarp-gateway partner auth delete

.. describe:: waarp-gateway partner <PARTNER> auth delete <AUTH>

Supprime la valeur d'authentification données du partenaire.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner auth 'openssh' delete 'openssh_hostkey'

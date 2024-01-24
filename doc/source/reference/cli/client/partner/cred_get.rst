========================================
Consulter une méthode d'authentification
========================================

.. program:: waarp-gateway partner credential get

.. describe:: waarp-gateway partner <PARTNER> credential get <CRED_NAME>

Affiche les informations de la méthode d'authentification donnée.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' partner credential 'openssh' get 'openssh_hostkey'

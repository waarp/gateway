=======================
Modifier un utilisateur
=======================

.. program:: waarp-gateway user update

.. describe:: waarp-gateway <ADDR> user update <USER>

Remplace les attributs de l'utilisateur demandé avec ceux donnés. Les attributs
omis restent inchangés.

.. option:: -u <USERNAME>, --username=<USERNAME>

   Le nom de l'utilisateur. Doit être unique.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe de l'utilisateur.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 user update toto -u toto2 -p sésame2
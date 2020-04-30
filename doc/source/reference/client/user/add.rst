======================
Ajouter un utilisateur
======================

.. program:: waarp-gateway user add

.. describe:: waarp-gateway <ADDR> user add

Ajoute un nouvel utilisateur avec les identifiants donnés.

.. option:: -u <USERNAME>, --username=<USERNAME>

   Le nom du nouvel utilisateur créé. Doit être unique.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du nouvel utilisateur.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 user add -u toto -p sésame

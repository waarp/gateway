==============================
Ajouter un compte à un serveur
==============================

.. program:: waarp-gateway account local add

.. describe:: waarp-gateway <ADDR> account local <SERVER> add

Attache un nouveau compte au serveur donné à partir des informations renseignées.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un serveur donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 account local serveur_sftp add -l tata -p password
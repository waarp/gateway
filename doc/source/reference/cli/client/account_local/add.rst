==============================
Ajouter un compte à un serveur
==============================

.. program:: waarp-gateway account local add

Attache un nouveau compte au serveur donné à partir des informations renseignées.

**Commande**

.. code-block:: shell

   waarp-gateway account local "<PARTNER>" add

**Options**

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un serveur donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

**Exemple**

.. code-block:: shell

   waarp-gateway account local 'serveur_sftp' add -l 'tata' -p 'password'

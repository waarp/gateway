=================================
Ajouter un compte à un partenaire
=================================

.. program:: waarp-gateway account remote add

.. describe:: waarp-gateway <ADDR> account remote <PARTNER> add

Ajoute un nouveau compte sur le partenaire donné avec les identifiants ci-dessous.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un partenaire donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@remotehost:8080 account remote waarp_sftp add -l titi -p password
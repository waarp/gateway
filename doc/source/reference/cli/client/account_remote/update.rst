==========================
Modifier un compte distant
==========================

.. program:: waarp-gateway account remote update

.. describe:: waarp-gateway account remote <PARTNER> update <LOGIN>

Remplace les informations du compte donné par celles fournies ci-dessous.

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un partenaire donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a http://user:password@remotehost:8080 account remote 'openssh' update 'titi' -l 'titi2' -p 'password2'
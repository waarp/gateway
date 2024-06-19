==========================
Modifier un compte distant
==========================

.. program:: waarp-gateway account remote update

Remplace les informations du compte donné par celles fournies ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" update "<LOGIN>"

**Options**

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un partenaire donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'openssh' update 'titi' -l 'titi2' -p 'password2'

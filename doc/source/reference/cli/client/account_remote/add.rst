=================================
Ajouter un compte à un partenaire
=================================

.. program:: waarp-gateway account remote add

Ajoute un nouveau compte sur le partenaire donné avec les identifiants ci-dessous.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" add

**Options**

.. option:: -l <LOGIN>, --login=<LOGIN>

   Le login du compte. Doit être unique pour un partenaire donné.

.. option:: -p <PASS>, --password=<PASS>

   Le mot de passe du compte.

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'openssh' add -l 'titi' -p 'password'

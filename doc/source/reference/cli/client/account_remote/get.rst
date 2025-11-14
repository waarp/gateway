==========================
Afficher un compte distant
==========================

.. program:: waarp-gateway account remote get

Affiche les informations du compte demandé en paramètre de commande.

**Commande**

.. code-block:: shell

   waarp-gateway account remote "<PARTNER>" get "<LOGIN>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway account remote 'openssh' get 'titi'

=====================
Afficher un transfert
=====================

.. program:: waarp-gateway transfer get

Affiche les informations du transfert demandé.

**Commande**

.. code-block:: shell

   waarp-gateway transfer get "<TRANSFER_ID>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway transfer get 1234

======================
Consulter une autorité
======================

.. program:: waarp-gateway authority get

Affiche les informations de l'autorité demandée.

**Commande**

.. code-block:: shell

   waarp-gateway authority get "<AUTHORITY>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway authority get 'waarp_ca'

=================================
Lister les indirections d'adresse
=================================

.. program:: waarp-gateway override address list

Affiche une liste de toutes les indirections d'adresse existantes sur l'instance
de Waarp Gateway interrogée.

**Commande**

.. code-block:: shell

   waarp-gateway override address list

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway override address list

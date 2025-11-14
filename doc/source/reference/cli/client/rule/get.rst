==================
Afficher une règle
==================

.. program:: waarp-gateway rule get

Affiche les informations de la règle donnée en paramètre de commande.

**Commande**

.. code-block:: shell

   waarp-gateway rule get "<RULE>" "<DIRECTION>"

``DIRECTION`` peut être ``send`` ou ``receive``.

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway rule get 'règle_1' 'send'

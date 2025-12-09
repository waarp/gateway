=======================
Afficher un utilisateur
=======================

.. program:: waarp-gateway user get

Affiche les informations de l'utilisateur donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway user get "<USERNAME>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway user get 'toto'

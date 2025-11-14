===========================
Afficher une instance cloud
===========================

.. program:: waarp-gateway cloud get

Affiche les informations de l'instance cloud donnée en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway cloud get "<NAME>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway cloud get "aws"

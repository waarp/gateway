===========================
Afficher un filewatcher
===========================

.. program:: waarp-gateway filewatcher get

Affiche les informations du *filewatcher* donné en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway filewatcher get "<FLOW>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway filewatcher get 'my-filewatcher'

===================
Afficher un serveur
===================

.. program:: waarp-gateway server get

Affiche les informations du serveur donné en paramètre de commande.

**Commande**

.. code-block:: shell

   waarp-gateway server get "<SERVER>"

**Options**

.. option:: --format=<FORMAT>

   Spécifie le format du retour de la commande. Les valeurs acceptées sont :
   ``human``, ``json`` et ``yaml``. Par défaut, le format sera le format pour
   humain (``human``).

|

**Exemple**

.. code-block:: shell

   waarp-gateway server get 'gw_r66'

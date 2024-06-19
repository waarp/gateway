=================================
Ajouter une indirection d'adresse
=================================

.. program:: waarp-gateway override address set

Ajoute une indirection sur une adresse pour la remplacer par une autre.

**Commande**

.. code-block:: shell

   waarp-gateway override address set

**Options**

.. option:: -t <ADDRESS>, --target=<ADDRESS>

   L'adresse cible à remplacer. Si cette adresse possède déjà une indirection,
   celle-ci sera écrasée par la nouvelle.

.. option:: -r <ADDRESS>, --replace-by=<ADDRESS>

   L'adresse de remplacement.

**Exemple**

.. code-block:: shell

   waarp-gateway override address set -t 'waarp.fr' -r '192.168.1.1'

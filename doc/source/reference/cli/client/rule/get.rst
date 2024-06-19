==================
Afficher une règle
==================

.. program:: waarp-gateway rule get

Affiche les informations de la règle donnée en paramètre de commande.

**Commande**

.. code-block:: shell

   waarp-gateway rule get "<RULE>" "<DIRECTION>"

``DIRECTION`` peut être ``send`` ou ``receive``.

**Exemple**

.. code-block:: shell

   waarp-gateway rule get 'règle_1' 'send'

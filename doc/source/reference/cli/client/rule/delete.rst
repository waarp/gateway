===================
Supprimer une règle
===================

.. program:: waarp-gateway rule delete

Supprime la règle donnée en paramètre.

**Commande**

.. code-block:: shell

   waarp-gateway rule delete "<RULE>" "<DIRECTION>"

``DIRECTION`` peut être ``send`` ou ``receive``.

**Exemple**

.. code-block:: shell

   waarp-gateway rule delete 'règle_1'

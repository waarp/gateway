==================
Afficher une règle
==================

.. program:: waarp-gateway rule get

.. describe:: waarp-gateway <ADDR> rule list <RULE> <DIRECTION>

Affiche les informations de la règle donnée en paramètre de commande.

``DIRECTION`` peut être ``SEND`` ou ``RECEIVE``.

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 rule get règle_1 SEND

==================
Afficher une règle
==================

.. program:: waarp-gateway rule get

.. describe:: waarp-gateway rule list <RULE> <DIRECTION>

Affiche les informations de la règle donnée en paramètre de commande.

``DIRECTION`` peut être ``send`` ou ``receive``.

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 rule get "règle_1" "send"

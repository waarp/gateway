===================================
Supprimer une indirection d'adresse
===================================

.. program:: waarp-gateway override address delete

.. describe:: waarp-gateway override address delete <TARGET>

Supprime l'indirection d'adresse sur l'adresse donnée (si elle existe).

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' override address delete 'waarp.fr'
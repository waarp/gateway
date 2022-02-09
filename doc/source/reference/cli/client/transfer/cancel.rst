====================
Annuler un transfert
====================

.. program:: waarp-gateway transfer cancel

.. describe:: waarp-gateway transfer cancel <TRANSFER_ID>

Annule le transfert donné. Une fois annulé, le transfert est déplacé dans
l'historique.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' transfer cancel 1234
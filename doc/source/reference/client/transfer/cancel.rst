====================
Annuler un transfert
====================

.. program:: waarp-gateway transfer cancel

.. describe:: waarp-gateway <ADDR> transfer cancel <TRANS>

Annule le transfert donné. Une fois annulé, le transfert est déplacé dans
l'historique.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 transfer cancel 1234
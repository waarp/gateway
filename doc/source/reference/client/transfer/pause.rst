======================
Suspendre un transfert
======================

.. program:: waarp-gateway transfer pause

.. describe:: waarp-gateway <ADDR> transfer pause <TRANS>

Pause le transfert donné. Seuls les transferts en cours ou en attente peuvent
être mis en pause. Pour reprendre le transfert, voir :doc:`resume`.

|

**Exemple**

.. code-block:: bash

   waarp-gateway http://user:password@localhost:8080 transfer pause 1234
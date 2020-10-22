=====================
Reprendre un transfer
=====================

.. program:: waarp-gateway transfer resume

.. describe:: waarp-gateway <ADDR> transfer resume <TRANS>

Reprend le transfert donné. Seuls les transferts interrompus ou en pause peuvent
être repris.

|

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 transfer resume 1234
######################
Suspendre un transfert
######################

.. program:: waarp-gateway transfer pause

.. describe:: waarp-gateway transfer pause <TRANS>

Pause le transfert donné. Seuls les transferts en cours ou en attente peuvent
être mis en pause. Pour reprendre le transfert, voir
:any:`reference-cli-client-transfers-resume`.

**Exemple**

.. code-block:: shell

   waarp-gateway http://user:password@localhost:8080 transfer pause 1234

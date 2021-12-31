######################
Suspendre un transfert
######################

.. program:: waarp-gateway transfer pause

.. describe:: waarp-gateway transfer pause <TRANSFER_ID>

Pause le transfert donné. Un transfer en pause n'est pas considéré comme terminé.
Seuls les transferts en cours ou en attente peuvent être mis en pause. Pour
reprendre le transfert, voir :any:`reference-cli-client-transfers-resume`.

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' transfer pause 1234

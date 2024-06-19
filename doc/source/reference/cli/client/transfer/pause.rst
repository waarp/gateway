######################
Suspendre un transfert
######################

.. program:: waarp-gateway transfer pause

Pause le transfert donné. Un transfert en pause n'est pas considéré comme terminé.
Seuls les transferts en cours ou en attente peuvent être mis en pause. Pour
reprendre le transfert, voir :any:`reference-cli-client-transfers-resume`.

**Commande**

.. code-block:: shell

   waarp-gateway transfer pause "<TRANSFER_ID>"


**Exemple**

.. code-block:: shell

   waarp-gateway transfer pause 1234

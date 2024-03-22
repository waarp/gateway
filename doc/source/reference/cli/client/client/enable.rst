==============================
Activer un client au démarrage
==============================

.. program:: waarp-gateway client enable

.. describe:: waarp-gateway client enable <CLIENT>

Active le client donné, signifiant que celui-ci pourra être démarré automatiquement
lors du prochain lancement de la *gateway*. Par défaut, les clients nouvellement
créés sont actifs.

|

**Exemple**

.. code-block:: shell

   waarp-gateway enable 'sftp_client'
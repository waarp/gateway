==============================
Activer un client au démarrage
==============================

.. program:: waarp-gateway client enable

Active le client donné, signifiant que celui-ci pourra être démarré automatiquement
lors du prochain lancement de la *gateway*. Par défaut, les clients nouvellement
créés sont actifs.

**Commande**

.. code-block:: shell

   waarp-gateway client enable "<CLIENT>"

**Exemple**

.. code-block:: shell

   waarp-gateway client enable 'sftp_client'
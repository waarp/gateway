===============================
Activer un serveur au démarrage
===============================

.. program:: waarp-gateway server enable

Active le serveur donné, signifiant que celui-ci pourra être démarré automatiquement
lors du prochain lancement de Waarp Gateway. Par défaut, les serveurs nouvellement
créés sont actifs.

**Commande**

.. code-block:: shell

   waarp-gateway server enable "<SERVER>"

**Exemple**

.. code-block:: shell

   waarp-gateway server enable 'serveur_sftp'

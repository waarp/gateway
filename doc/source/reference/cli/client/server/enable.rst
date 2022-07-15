===============================
Activer un serveur au démarrage
===============================

.. program:: waarp-gateway server enable

.. describe:: waarp-gateway server enable <SERVER>

Active le serveur donné, signifiant que celui-ci pourra être démarré automatiquement
lors du prochain lancement de la *gateway*. Par défaut, les serveurs nouvellement
créés sont actifs.

|

**Exemple**

.. code-block:: shell

   waarp-gateway -a 'http://user:password@localhost:8080' enable 'serveur_sftp'
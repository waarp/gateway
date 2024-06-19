==================================
Désactiver un serveur au démarrage
==================================

.. program:: waarp-gateway server disable

Désactive le serveur donné, signifiant que celui-ci ne sera pas démarré automatiquement
lors du prochain lancement de Waarp Gateway. Le serveur restera cependant actif
jusqu'à l'arrêt de Waarp Gateway.

**Commande**

.. code-block:: shell

   waarp-gateway server disable "<SERVER>"

**Exemple**

.. code-block:: shell

   waarp-gateway server disable 'serveur_sftp'

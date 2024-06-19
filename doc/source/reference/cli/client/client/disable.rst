=================================
Désactiver un client au démarrage
=================================

.. program:: waarp-gateway client disable

Désactive le client donné, signifiant que celui-ci ne sera pas démarré automatiquement
lors du prochain lancement de la *gateway*. Le client restera cependant actif
jusqu'à ce qu'il soit arrêté.

**Commande**

.. code-block:: shell

   waarp-gateway client disable "<CLIENT>"

**Exemple**

.. code-block:: shell

   waarp-gateway client disable 'sftp_client'